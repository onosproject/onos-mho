// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: LicenseRef-ONF-Member-1.0

package controller

import (
	"context"
	e2tapi "github.com/onosproject/onos-api/go/onos/e2t/e2"
	e2api "github.com/onosproject/onos-api/go/onos/e2t/e2/v1beta1"
	e2sm_mho "github.com/onosproject/onos-e2-sm/servicemodels/e2sm_mho/v1/e2sm-mho"
	"github.com/onosproject/onos-lib-go/pkg/logging"
	appConfig "github.com/onosproject/onos-mho/pkg/config"
	"github.com/onosproject/onos-mho/pkg/store"
	"github.com/onosproject/onos-ric-sdk-go/pkg/e2/indication"
	"github.com/onosproject/rrm-son-lib/pkg/handover"
	"github.com/onosproject/rrm-son-lib/pkg/model/device"
	rrmid "github.com/onosproject/rrm-son-lib/pkg/model/id"
	"github.com/onosproject/rrm-son-lib/pkg/model/measurement"
	meastype "github.com/onosproject/rrm-son-lib/pkg/model/measurement/type"
	"google.golang.org/protobuf/proto"
	"strconv"
)

const (
	ControlPriority = 10
)

var log = logging.GetLogger("controller")

type UeData struct {
	UeID     string
	E2NodeID string
	CGI      *e2sm_mho.CellGlobalId
	RrcState string
}

type E2NodeIndication struct {
	NodeID      string
	TriggerType e2sm_mho.MhoTriggerType
	IndMsg      indication.Indication
}

// MhoCtrl is the controller for MHO
type MhoCtrl struct {
	IndChan          chan *E2NodeIndication
	CtrlReqChans     map[string]chan *e2api.ControlMessage
	HoCtrl           *HandOverController
	store store.Store
}

// NewMhoController returns the struct for MHO logic
func NewMhoController(cfg appConfig.Config, indChan chan *E2NodeIndication, ctrlReqChs map[string]chan *e2api.ControlMessage, store store.Store) *MhoCtrl {
	log.Info("Init MhoController")
	return &MhoCtrl{
		IndChan:          indChan,
		CtrlReqChans:     ctrlReqChs,
		HoCtrl:           NewHandOverController(cfg),
		store: store,
	}
}

// Run starts to listen Indication message and then save the result to its struct
func (c *MhoCtrl) Run(ctx context.Context) {
	log.Info("Start MhoController")
	go c.HoCtrl.Run()
	go c.listenIndChan(ctx)
	c.listenHandOver(ctx)
}

func (c *MhoCtrl) listenIndChan(ctx context.Context) {
	var err error
	for indMsg := range c.IndChan {

		indHeaderByte := indMsg.IndMsg.Payload.Header
		indMessageByte := indMsg.IndMsg.Payload.Message
		e2NodeID := indMsg.NodeID

		indHeader := e2sm_mho.E2SmMhoIndicationHeader{}
		if err = proto.Unmarshal(indHeaderByte, &indHeader); err == nil {
			indMessage := e2sm_mho.E2SmMhoIndicationMessage{}
			if err = proto.Unmarshal(indMessageByte, &indMessage); err == nil {
				switch x := indMessage.E2SmMhoIndicationMessage.(type) {
				case *e2sm_mho.E2SmMhoIndicationMessage_IndicationMessageFormat1:
					if indMsg.TriggerType == e2sm_mho.MhoTriggerType_MHO_TRIGGER_TYPE_UPON_RCV_MEAS_REPORT {
						go c.handleMeasReport(ctx, indHeader.GetIndicationHeaderFormat1(), indMessage.GetIndicationMessageFormat1(), e2NodeID)
					} else if indMsg.TriggerType == e2sm_mho.MhoTriggerType_MHO_TRIGGER_TYPE_PERIODIC {
						go c.handlePeriodicReport(ctx, indHeader.GetIndicationHeaderFormat1(), indMessage.GetIndicationMessageFormat1(), e2NodeID)
					}
				case *e2sm_mho.E2SmMhoIndicationMessage_IndicationMessageFormat2:
					go c.handleRrcState(ctx, indHeader.GetIndicationHeaderFormat1(), indMessage.GetIndicationMessageFormat2(), e2NodeID)
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

func (c *MhoCtrl) listenHandOver(ctx context.Context) {
	for hoDecision := range c.HoCtrl.HandoverHandler.Chans.OutputChan {
		go func(hoDecision handover.A3HandoverDecision) {
			if err := c.control(ctx, hoDecision); err != nil {
				log.Error(err)
			}
		}(hoDecision)
	}
}

func (c *MhoCtrl) handlePeriodicReport(ctx context.Context, header *e2sm_mho.E2SmMhoIndicationHeaderFormat1, message *e2sm_mho.E2SmMhoIndicationMessageFormat1, e2NodeID string) {
	ueID := message.GetUeId().GetValue()
	log.Infof("rx periodic, e2NodeID:%v, ueID:%v", e2NodeID, ueID)
	var ueData *UeData
	u, err := c.store.Get(ctx, store.Key{UeID: ueID})
	if err != nil {
		ueData = &UeData{}
	} else {
		t := u.Value.(UeData)
		ueData = &t
	}

	ueData.UeID = ueID
	ueData.E2NodeID = e2NodeID
	ueData.CGI = header.GetCgi()

	_, err = c.store.Put(ctx, store.Key{UeID: ueID}, *ueData)
	if err != nil {
		log.Warn(err)
	}
}

func (c *MhoCtrl) handleMeasReport(ctx context.Context, header *e2sm_mho.E2SmMhoIndicationHeaderFormat1, message *e2sm_mho.E2SmMhoIndicationMessageFormat1, e2NodeID string) {

	ueID := message.GetUeId().GetValue()

	log.Infof("rx measurement, e2NodeID:%v, ueID:%v", e2NodeID, ueID)

	ecgiSCell := rrmid.NewECGI(header.GetCgi().GetNrCgi().GetNRcellIdentity().GetValue().GetValue())

	scell := device.NewCell(
		ecgiSCell,
		meastype.A3OffsetRange(c.HoCtrl.A3OffsetRange),
		meastype.HysteresisRange(c.HoCtrl.HysteresisRange),
		meastype.QOffsetRange(c.HoCtrl.CellIndividualOffset),
		meastype.QOffsetRange(c.HoCtrl.FrequencyOffset),
		meastype.TimeToTriggerRange(c.HoCtrl.TimeToTrigger))

	cscellList := make([]device.Cell, 0)
	ueID_int, err := strconv.Atoi(ueID)
	if err != nil {
		log.Error(err)
		return
	}
	ue := device.NewUE(rrmid.NewUEID(uint64(ueID_int), uint32(0), header.GetCgi().GetNrCgi().GetNRcellIdentity().GetValue().GetValue()), scell, nil)
	measSCellFound := false
	for _, measReport := range message.MeasReport {
		if measReport.GetCgi().GetNrCgi().GetNRcellIdentity().GetValue().GetValue() == header.GetCgi().GetNrCgi().GetNRcellIdentity().GetValue().GetValue() {
			ue.GetMeasurements()[ecgiSCell.String()] = measurement.NewMeasEventA3(ecgiSCell, measurement.RSRP(measReport.GetRsrp().GetValue()))
			measSCellFound = true
		} else {
			ecgiCSCell := rrmid.NewECGI(measReport.GetCgi().GetNrCgi().GetNRcellIdentity().GetValue().GetValue())
			//TODO - Get values from config
			cscell := device.NewCell(
				ecgiCSCell,
				meastype.A3OffsetRange(1),
				meastype.HysteresisRange(2),
				meastype.QOffsetRange(18),
				meastype.QOffsetRange(19),
				meastype.TimeToTriggerRange(0))
			cscellList = append(cscellList, cscell)
			ue.GetMeasurements()[ecgiCSCell.String()] = measurement.NewMeasEventA3(ecgiCSCell, measurement.RSRP(measReport.GetRsrp().GetValue()))
		}
	}

	// TODO - hack for ran-simulator
	if !measSCellFound {
		log.Warnf("serving cell measurement not present in report, e2NodeID:%v", header.GetCgi().GetNrCgi().GetNRcellIdentity().GetValue().GetValue())
		ecgiSCell := rrmid.NewECGI(header.GetCgi().GetNrCgi().GetNRcellIdentity().GetValue().GetValue())
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

	var ueData *UeData
	u, err := c.store.Get(ctx, store.Key{UeID: ueID})
	if err != nil {
		ueData = &UeData{}
	} else {
		t := u.Value.(UeData)
		ueData = &t
	}

	//TODO - don't update store if not needed
	ueData.UeID = ueID
	ueData.E2NodeID = e2NodeID
	ueData.CGI = header.GetCgi()

	_, err = c.store.Put(ctx, store.Key{UeID: ueID}, *ueData)
	if err != nil {
		log.Warn(err)
	}

	c.HoCtrl.A3Handler.Chans.InputChan <- ue

}

func (c *MhoCtrl) handleRrcState(ctx context.Context, header *e2sm_mho.E2SmMhoIndicationHeaderFormat1, message *e2sm_mho.E2SmMhoIndicationMessageFormat2, e2NodeID string) {
	ueID := message.GetUeId().GetValue()
	rrcState := message.GetRrcStatus().String()
	log.Infof("rx rrc, e2NodeID:%v, ueID:%v, rrcState:%v", e2NodeID, ueID, rrcState)

	var ueData *UeData
	u, err := c.store.Get(ctx, store.Key{UeID: ueID})
	if err != nil {
		ueData = &UeData{}
	} else {
		t := u.Value.(UeData)
		ueData = &t
	}

	ueData.UeID = ueID
	ueData.E2NodeID = e2NodeID
	ueData.CGI = header.GetCgi()
	ueData.RrcState = rrcState

	_, err = c.store.Put(ctx, store.Key{UeID: ueID}, *ueData)
	if err != nil {
		log.Warn(err)
	}

}

func (c *MhoCtrl) control(ctx context.Context, ho handover.A3HandoverDecision) error {
	var err error
	id := ho.UE.GetID().GetID().(rrmid.UEID).IMSI

	ueID := strconv.Itoa(int(id))

	u, err := c.store.Get(ctx, store.Key{UeID: ueID})
	if err != nil {
		log.Warn(err)
	}

	ueData := u.Value.(UeData)

	e2NodeID := ueData.E2NodeID

	// TODO - check servingCGI

	CGI := ueData.CGI
	cellID := CGI.GetNrCgi().GetNRcellIdentity().GetValue().GetValue()
	cellIDLen := CGI.GetNrCgi().GetNRcellIdentity().GetValue().GetLen()
	plmnID := CGI.GetNrCgi().GetPLmnIdentity().GetValue()

	e2smMhoControlHandler := &E2SmMhoControlHandler{
		NodeID:            e2NodeID,
		ControlAckRequest: e2tapi.ControlAckRequest_NO_ACK,
	}

	nci, err := strconv.Atoi(ho.TargetCell.GetID().String())
	if err != nil {
		return err
	}
	targetCGI := &e2sm_mho.CellGlobalId{
		CellGlobalId: &e2sm_mho.CellGlobalId_NrCgi{
			NrCgi: &e2sm_mho.Nrcgi{
				PLmnIdentity: &e2sm_mho.PlmnIdentity{
					Value: plmnID,
				},
				NRcellIdentity: &e2sm_mho.NrcellIdentity{
					Value: &e2sm_mho.BitString{
						Value: uint64(nci),
						Len:   36,
					},
				},
			},
		},
	}

	ueIdentity := e2sm_mho.UeIdentity{
		Value: ueID,
	}

	go func() {
		if e2smMhoControlHandler.ControlHeader, err = e2smMhoControlHandler.CreateMhoControlHeader(cellID, cellIDLen, int32(ControlPriority), plmnID); err == nil {
			if e2smMhoControlHandler.ControlMessage, err = e2smMhoControlHandler.CreateMhoControlMessage(CGI, &ueIdentity, targetCGI); err == nil {
				if controlRequest, err := e2smMhoControlHandler.CreateMhoControlRequest(); err == nil {
					c.CtrlReqChans[e2NodeID] <- controlRequest
					log.Infof("tx control, e2NodeID:%v, ueID:%v", e2NodeID, ueID)
				}
			}
		}
	}()

	return err

}
