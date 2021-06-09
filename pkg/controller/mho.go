// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: LicenseRef-ONF-Member-1.0

package controller

import (
	"fmt"
	e2tapi "github.com/onosproject/onos-api/go/onos/e2t/e2"
	e2sm_mho "github.com/onosproject/onos-e2-sm/servicemodels/e2sm_mho/v1/e2sm-mho"
	"github.com/onosproject/onos-lib-go/pkg/logging"
	"github.com/onosproject/onos-mho/pkg/southbound/ricapie2"
	"github.com/onosproject/onos-mho/pkg/store"
	"github.com/onosproject/rrm-son-lib/pkg/handover"
	"github.com/onosproject/rrm-son-lib/pkg/model/device"
	"github.com/onosproject/rrm-son-lib/pkg/model/id"
	"github.com/onosproject/rrm-son-lib/pkg/model/measurement"
	meastype "github.com/onosproject/rrm-son-lib/pkg/model/measurement/type"
	"google.golang.org/protobuf/proto"
	"strconv"
)

const (
	ControlPriority = 10
)

var log = logging.GetLogger("controller", "mho")

type ueData struct {
	header    *e2sm_mho.E2SmMhoIndicationHeaderFormat1
	message   *e2sm_mho.E2SmMhoIndicationMessageFormat1
	ueID      *e2sm_mho.UeIdentity
	plmnID    []byte
	e2NodeID  string
	servingCGI *e2sm_mho.CellGlobalId
}

// MhoCtrl is the controller for MHO
type MhoCtrl struct {
	IndChan      chan *store.E2NodeIndication
	CtrlReqChans map[string]chan *e2tapi.ControlRequest
	HoCtrl       *HandOverController
	UeCache        map[id.ID]ueData
}

// NewMhoController returns the struct for MHO logic
func NewMhoController(indChan chan *store.E2NodeIndication, ctrlReqChs map[string]chan *e2tapi.ControlRequest) *MhoCtrl {
	log.Info("Start onos-mho Application Controller")
	return &MhoCtrl{
		IndChan:      indChan,
		CtrlReqChans: ctrlReqChs,
		HoCtrl:       NewHandOverController(),
		UeCache:        make(map[id.ID]ueData, 0),
	}
}

// Run starts to listen Indication message and then save the result to its struct
func (c *MhoCtrl) Run() {
	go c.HoCtrl.Run()
	go c.listenIndChan()
	c.listenHandOver()
}

func (c *MhoCtrl) listenIndChan() {
	var err error
	for indMsg := range c.IndChan {
		log.Debugf("Raw message: %v", indMsg)

		indHeaderByte := indMsg.IndMsg.Payload.Header
		indMessageByte := indMsg.IndMsg.Payload.Message
		e2NodeID := indMsg.NodeID

		indHeader := e2sm_mho.E2SmMhoIndicationHeader{}
		if err = proto.Unmarshal(indHeaderByte, &indHeader); err == nil {
			indMessage := e2sm_mho.E2SmMhoIndicationMessage{}
			if err = proto.Unmarshal(indMessageByte, &indMessage); err == nil {
				log.Debugf("MHO indication header: %v", indHeader.GetIndicationHeaderFormat1())
				log.Debugf("MHO indication message: %v", indMessage.GetIndicationMessageFormat1())
				switch x := indMessage.E2SmMhoIndicationMessage.(type) {
				case *e2sm_mho.E2SmMhoIndicationMessage_IndicationMessageFormat1:
					c.handleIndMsgFormat1(indHeader.GetIndicationHeaderFormat1(), indMessage.GetIndicationMessageFormat1(), e2NodeID)
				case *e2sm_mho.E2SmMhoIndicationMessage_IndicationMessageFormat2:
					c.handleIndMsgFormat2(indHeader.GetIndicationHeaderFormat1(), indMessage.GetIndicationMessageFormat1(), e2NodeID)
				default:
					log.Warnf("Unknown MHO indication message format, indication message: %v", x)
				}
			}
		}
		if err != nil {
			log.Error(err)
		}
	}
}

func (c *MhoCtrl) listenHandOver() {
	for hoDecision := range c.HoCtrl.HandoverHandler.Chans.OutputChan {
		if err := c.control(hoDecision); err != nil {
			log.Error(err)
		}
	}
}

//*E2SmMhoIndicationMessageFormat1
func (c *MhoCtrl) handleIndMsgFormat1(header *e2sm_mho.E2SmMhoIndicationHeaderFormat1, message *e2sm_mho.E2SmMhoIndicationMessageFormat1, e2NodeID string) {

	imsi, err := strconv.Atoi(message.GetUeId().GetValue())
	if err != nil {
		log.Error(err)
		return
	}
	ueid := id.NewUEID(uint64(imsi), uint32(0), header.GetCgi().GetNrCgi().GetNRcellIdentity().GetValue().GetValue())

	ecgiSCell := id.NewECGI(header.GetCgi().GetNrCgi().GetNRcellIdentity().GetValue().GetValue())

	scell := device.NewCell(
		ecgiSCell,
		meastype.A3OffsetRange(c.HoCtrl.A3OffsetRange),
		meastype.HysteresisRange(c.HoCtrl.HysteresisRange),
		meastype.QOffsetRange(c.HoCtrl.CellIndividualOffset),
		meastype.QOffsetRange(c.HoCtrl.FrequencyOffset),
		meastype.TimeToTriggerRange(c.HoCtrl.TimeToTrigger))

	cscellList := make([]device.Cell, 0)
	ue := device.NewUE(ueid, scell, nil)
	measSCellFound := false
	for _, measReport := range message.MeasReport {
		if measReport.GetCgi().GetNrCgi().GetNRcellIdentity().GetValue().GetValue() == header.GetCgi().GetNrCgi().GetNRcellIdentity().GetValue().GetValue() {
			ue.GetMeasurements()[ecgiSCell.String()] = measurement.NewMeasEventA3(ecgiSCell, measurement.RSRP(measReport.GetRsrp().GetValue()))
			measSCellFound = true
		} else {
			ecgiCSCell := id.NewECGI(measReport.GetCgi().GetNrCgi().GetNRcellIdentity().GetValue().GetValue())
			cscell := device.NewCell(
				ecgiCSCell,
				meastype.A3OffsetRange(c.HoCtrl.A3OffsetRange),
				meastype.HysteresisRange(c.HoCtrl.HysteresisRange),
				meastype.QOffsetRange(c.HoCtrl.CellIndividualOffset),
				meastype.QOffsetRange(c.HoCtrl.FrequencyOffset),
				meastype.TimeToTriggerRange(c.HoCtrl.TimeToTrigger))
			cscellList = append(cscellList, cscell)
			ue.GetMeasurements()[ecgiCSCell.String()] = measurement.NewMeasEventA3(ecgiCSCell, measurement.RSRP(measReport.GetRsrp().GetValue()))
		}
	}

	// TODO - hack for ran-simulator
	if !measSCellFound {
		log.Warnf("Serving cell measurement not found")
		ecgiSCell := id.NewECGI(header.GetCgi().GetNrCgi().GetNRcellIdentity().GetValue().GetValue())
		scell := device.NewCell(
			ecgiSCell,
			meastype.A3OffsetRange(c.HoCtrl.A3OffsetRange),
			meastype.HysteresisRange(c.HoCtrl.HysteresisRange),
			meastype.QOffsetRange(c.HoCtrl.CellIndividualOffset),
			meastype.QOffsetRange(c.HoCtrl.FrequencyOffset),
			meastype.TimeToTriggerRange(c.HoCtrl.TimeToTrigger))
		cscellList = append(cscellList, scell)
		ue.GetMeasurements()[ecgiSCell.String()] = measurement.NewMeasEventA3(ecgiSCell, measurement.RSRP(-1000))
	}

	ue.SetCSCells(cscellList)
	c.cacheUE(ue.GetID(), header, message, e2NodeID)
	c.HoCtrl.A3Handler.Chans.InputChan <- ue

}

func (c *MhoCtrl) cacheUE(id id.ID, header *e2sm_mho.E2SmMhoIndicationHeaderFormat1,
	message *e2sm_mho.E2SmMhoIndicationMessageFormat1, e2NodeID string) {
	c.UeCache[id] = ueData{
		header: header,
		message: message,
		ueID: message.GetUeId(),
		e2NodeID: e2NodeID,
		servingCGI: header.GetCgi(),
	}

}

func (c *MhoCtrl) handleIndMsgFormat2(header *e2sm_mho.E2SmMhoIndicationHeaderFormat1, message *e2sm_mho.E2SmMhoIndicationMessageFormat1, e2NodeID string) {
	// TODO
	log.Info("Ignore indication message format 2")

}

func (c *MhoCtrl) control(ho handover.A3HandoverDecision) error {
	var err error

	id := ho.UE.GetID()

	ue, ok := c.UeCache[id]
	if !ok {
		return fmt.Errorf("UE id %v not found in cache", id)
	}

	ueID := ue.ueID
	e2NodeID := ue.e2NodeID

	// TODO - check servingCGI

	servingCGI := ue.servingCGI
	cellID := servingCGI.GetNrCgi().GetNRcellIdentity().GetValue().GetValue()
	cellIDLen := servingCGI.GetNrCgi().GetNRcellIdentity().GetValue().GetLen()
    plmnID := servingCGI.GetNrCgi().GetPLmnIdentity().GetValue()

	e2smMhoControlHandler := &E2SmMhoControlHandler{
		NodeID:              e2NodeID,
		EncodingType:        e2tapi.EncodingType_PROTO,
		ServiceModelName:    ricapie2.ServiceModelName,
		ServiceModelVersion: ricapie2.ServiceModelVersion,
		ControlAckRequest:   e2tapi.ControlAckRequest_NO_ACK,
	}

	nci, err := strconv.Atoi(ho.TargetCell.GetID().String())
	targetCGI := &e2sm_mho.CellGlobalId{
		CellGlobalId: &e2sm_mho.CellGlobalId_NrCgi{
			NrCgi: &e2sm_mho.Nrcgi{
				PLmnIdentity: &e2sm_mho.PlmnIdentity{
					Value: plmnID,
				},
				NRcellIdentity: &e2sm_mho.NrcellIdentity{
					Value: &e2sm_mho.BitString{
						Value: uint64(nci),
						Len: 36,
					},
				},
			},
		},
	}

	if e2smMhoControlHandler.ControlHeader, err = e2smMhoControlHandler.CreateMhoControlHeader(cellID, cellIDLen, int32(ControlPriority), plmnID); err == nil {
		if e2smMhoControlHandler.ControlMessage, err = e2smMhoControlHandler.CreateMhoControlMessage(servingCGI, ueID, targetCGI); err == nil {
			if controlRequest, err := e2smMhoControlHandler.CreateMhoControlRequest(); err == nil {
				log.Debugf("Control Request message for e2NodeID %s: %v", e2NodeID, controlRequest)
				c.CtrlReqChans[e2NodeID] <- controlRequest
			}
		}
	}

	return err

}
