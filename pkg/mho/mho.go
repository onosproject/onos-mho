// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: LicenseRef-ONF-Member-1.0

package mho

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

var log = logging.GetLogger("mho")

type UeData struct {
	UeID      string
	E2NodeID  string
	CGI       *e2sm_mho.CellGlobalId
	CGIString string
	RrcState  string
	RsrpServing int32
	RsrpNeighbors map[string]int32
}

type CellData struct {
	CGIString              string
	NumberRrcIdle          int
	NumberRrcConnected     int
	CumulativeHandoversIn  int
	CumulativeHandoversOut int
}

type E2NodeIndication struct {
	NodeID      string
	TriggerType e2sm_mho.MhoTriggerType
	IndMsg      indication.Indication
}

// MhoCtrl is the controller for MHO
type MhoCtrl struct {
	IndChan      chan *E2NodeIndication
	CtrlReqChans map[string]chan *e2api.ControlMessage
	HoCtrl       *HandOverController
	ueStore      store.Store
	cellStore    store.Store
}

// NewMhoController returns the struct for MHO logic
func NewMhoController(cfg appConfig.Config, indChan chan *E2NodeIndication, ctrlReqChs map[string]chan *e2api.ControlMessage, ueStore store.Store, cellStore store.Store) *MhoCtrl {
	log.Info("Init MhoController")
	return &MhoCtrl{
		IndChan:      indChan,
		CtrlReqChans: ctrlReqChs,
		HoCtrl:       NewHandOverController(cfg),
		ueStore:      ueStore,
		cellStore:    cellStore,
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

	u, err := c.ueStore.Get(ctx, ueID)
	if err != nil {
		return
	}

	ueData := u.Value.(UeData)

	// Validate UeID
	if ueID != ueData.UeID {
		log.Warn("bad store")
		return
	}

	newCGIString := getCGIFromIndicationHeader(header)
	log.Infof("TRACE: cgi:%v", newCGIString)

	// Validate CGIString
	if newCGIString != ueData.CGIString {
		log.Warn("bad store")
		return
	}

	// Update RSRP
	servingNci := getNciFromCellGlobalId(header.GetCgi())
	ueData.RsrpServing, ueData.RsrpNeighbors = getRsrpFromMeasReport(servingNci, message.MeasReport)

	_, err = c.ueStore.Put(ctx, ueID, ueData)
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

	servingNci := getNciFromCellGlobalId(header.GetCgi())
	RsrpServing, RsrpNeighbors := getRsrpFromMeasReport(servingNci, message.MeasReport)

	//measSCellFound := false
	for _, measReport := range message.MeasReport {
		if measReport.GetCgi().GetNrCgi().GetNRcellIdentity().GetValue().GetValue() == header.GetCgi().GetNrCgi().GetNRcellIdentity().GetValue().GetValue() {
			ue.GetMeasurements()[ecgiSCell.String()] = measurement.NewMeasEventA3(ecgiSCell, measurement.RSRP(measReport.GetRsrp().GetValue()))
			//measSCellFound = true
		} else {
			ecgiCSCell := rrmid.NewECGI(measReport.GetCgi().GetNrCgi().GetNRcellIdentity().GetValue().GetValue())
			//TODO - Get values from config
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
	//if !measSCellFound {
	//	log.Warnf("serving cell measurement not present in report, e2NodeID:%v", header.GetCgi().GetNrCgi().GetNRcellIdentity().GetValue().GetValue())
	//	ecgiSCell := rrmid.NewECGI(header.GetCgi().GetNrCgi().GetNRcellIdentity().GetValue().GetValue())
	//	scell := device.NewCell(
	//		ecgiSCell,
	//		meastype.A3OffsetRange(c.HoCtrl.A3OffsetRange),
	//		meastype.HysteresisRange(c.HoCtrl.HysteresisRange),
	//		meastype.QOffsetRange(c.HoCtrl.CellIndividualOffset),
	//		meastype.QOffsetRange(c.HoCtrl.FrequencyOffset),
	//		meastype.TimeToTriggerRange(c.HoCtrl.TimeToTrigger))
	//	cscellList = append(cscellList, scell)
	//	ue.GetMeasurements()[ecgiSCell.String()] = measurement.NewMeasEventA3(ecgiSCell, measurement.RSRP(-1000))
	//}

	ue.SetCSCells(cscellList)

	var ueData *UeData
	u, err := c.ueStore.Get(ctx, ueID)
	if err != nil {
		ueData = &UeData{
			RsrpNeighbors: make(map[string]int32),
		}
	} else {
		t := u.Value.(UeData)
		ueData = &t
	}

	//TODO - don't update store if not needed
	ueData.UeID = ueID
	ueData.E2NodeID = e2NodeID

	ueData.CGI = header.GetCgi()
	ueData.CGIString = getCGIFromIndicationHeader(header)
	log.Infof("TRACE: cgi:%v", ueData.CGIString)

	ueData.RsrpServing = RsrpServing
	for cgi, rsrp := range RsrpNeighbors {
		ueData.RsrpNeighbors[cgi] = rsrp
	}

	_, err = c.ueStore.Put(ctx, ueID, *ueData)
	if err != nil {
		log.Warn(err)
	}

	c.HoCtrl.A3Handler.Chans.InputChan <- ue

}

func (c *MhoCtrl) handleRrcState(ctx context.Context, header *e2sm_mho.E2SmMhoIndicationHeaderFormat1, message *e2sm_mho.E2SmMhoIndicationMessageFormat2, e2NodeID string) {
	ueID := message.GetUeId().GetValue()
	newRrcState := message.GetRrcStatus().String()
	newCGIString := getCGIFromIndicationHeader(header)
	log.Infof("TRACE: cgi:%v", newCGIString)

	var ueData *UeData
	var oldCGIString string
	u, err := c.ueStore.Get(ctx, ueID)
	if err != nil {
		ueData = &UeData{
			RsrpNeighbors: make(map[string]int32),
		}
		ueData.CGIString = newCGIString
		oldCGIString = newCGIString
		ueData.RrcState = e2sm_mho.Rrcstatus_name[int32(e2sm_mho.Rrcstatus_RRCSTATUS_IDLE)]
		ueData.UeID = ueID
	} else {
		t := u.Value.(UeData)
		ueData = &t
		oldCGIString = ueData.CGIString
	}

	oldRrcState := ueData.RrcState

	if oldRrcState == newRrcState {
		log.Infof("IGNORE RRC STATE CHANGE - rx rrc, e2NodeID:%v, ueID:%v, newRrcState:%v, oldRrcState", e2NodeID, ueID, newRrcState, oldRrcState)
		return
	}
	log.Infof("rx rrc, e2NodeID:%v, ueID:%v, newRrcState:%v, oldRrcState", e2NodeID, ueID, newRrcState, oldRrcState)

	// Update ue store
	ueData.UeID = ueID
	ueData.E2NodeID = e2NodeID
	ueData.CGI = header.GetCgi()

	ueData.CGIString = newCGIString
	ueData.RrcState = newRrcState
	_, err = c.ueStore.Put(ctx, ueID, *ueData)
	if err != nil {
		log.Warn(err)
	}

	// Validate RRC state
	if oldCGIString != newCGIString {
		// Idle UE from another cell surfaced
		if oldRrcState != e2sm_mho.Rrcstatus_name[int32(e2sm_mho.Rrcstatus_RRCSTATUS_IDLE)] ||
			newRrcState != e2sm_mho.Rrcstatus_name[int32(e2sm_mho.Rrcstatus_RRCSTATUS_CONNECTED)] {
			log.Warnf("bad rrc state, %v %v %v %v", oldCGIString, newCGIString, oldRrcState, newRrcState)
		}
	} else if oldRrcState == newRrcState {
		log.Warnf("ignore rrc state")
		return
	}

	//------------------
	// Update cell store
	//------------------

	// Get cell data record
	var newCellData *CellData
	newCell, err := c.cellStore.Get(ctx, ueData.CGIString)

	// Create data record, if not found
	if err != nil {
		newCellData = &CellData{}
		log.Infof("TRACE: handleRrcState(): cell not found, cgi:%v", ueData.CGIString)
	} else {
		t := newCell.Value.(CellData)
		if t.CGIString != ueData.CGIString {
			log.Warnf("bad store data: expected:%v, actual:%v", ueData.CGIString, t.CGIString)
		}
		newCellData = &t
	}

	newCellData.CGIString = ueData.CGIString

	if newRrcState == e2sm_mho.Rrcstatus_name[int32(e2sm_mho.Rrcstatus_RRCSTATUS_IDLE)] {
		newCellData.NumberRrcIdle++
		newCellData.NumberRrcConnected--
		if newCellData.NumberRrcConnected < 0 {
			newCellData.NumberRrcConnected = 0
		}
	} else if newRrcState == e2sm_mho.Rrcstatus_name[int32(e2sm_mho.Rrcstatus_RRCSTATUS_CONNECTED)] {
		newCellData.NumberRrcConnected++

		if newCGIString == oldCGIString {
			newCellData.NumberRrcIdle--
			if newCellData.NumberRrcIdle < 0 {
				newCellData.NumberRrcIdle = 0
			}
		} else {
			var oldCellData *CellData
			oldCell, err := c.cellStore.Get(ctx, oldCGIString)
			if err != nil {
				oldCellData = &CellData{}
				log.Infof("TRACE: handleRrcState(): cell not found, cgi:%v", oldCGIString)
			} else {
				t := oldCell.Value.(CellData)
				if t.CGIString != oldCGIString {
					log.Warnf("bad store data: expected:%v, actual:%v", oldCGIString, t.CGIString)
				}
				oldCellData = &t
			}
			oldCellData.CGIString = oldCGIString
			oldCellData.NumberRrcIdle--
			if oldCellData.NumberRrcIdle < 0 {
				oldCellData.NumberRrcIdle = 0
			}
			_, err = c.cellStore.Put(ctx, oldCGIString, *oldCellData)
			if err != nil {
				log.Warn(err)
			}
		}
	}
	_, err = c.cellStore.Put(ctx, newCellData.CGIString, *newCellData)
	if err != nil {
		log.Warn(err)
	}

}

func (c *MhoCtrl) control(ctx context.Context, ho handover.A3HandoverDecision) error {
	var err error

	// Get ueData from store
	id := ho.UE.GetID().GetID().(rrmid.UEID).IMSI
	ueID := strconv.Itoa(int(id))
	u, err := c.ueStore.Get(ctx, ueID)
	if err != nil {
		log.Warn(err)
		return err
	}
	ueData := u.Value.(UeData)

	e2NodeID := ueData.E2NodeID

	// Gather all ye IDs
	servingCGI := ueData.CGI
	servingPlmnIDBytes := servingCGI.GetNrCgi().GetPLmnIdentity().GetValue()
	servingPlmnID := plmnIDBytesToInt(servingPlmnIDBytes)
	servingNCI := servingCGI.GetNrCgi().GetNRcellIdentity().GetValue().GetValue()
	servingNCILen := servingCGI.GetNrCgi().GetNRcellIdentity().GetValue().GetLen()
	targetPlmnIDBytes := servingPlmnIDBytes
	targetPlmnID := servingPlmnID
	targetNCI, err := strconv.Atoi(ho.TargetCell.GetID().String())
	if err != nil {
		return err
	}
	targetNCILen := 36

	e2smMhoControlHandler := &E2SmMhoControlHandler{
		NodeID:            e2NodeID,
		ControlAckRequest: e2tapi.ControlAckRequest_NO_ACK,
	}

	targetCGI := &e2sm_mho.CellGlobalId{
		CellGlobalId: &e2sm_mho.CellGlobalId_NrCgi{
			NrCgi: &e2sm_mho.Nrcgi{
				PLmnIdentity: &e2sm_mho.PlmnIdentity{
					Value: targetPlmnIDBytes,
				},
				NRcellIdentity: &e2sm_mho.NrcellIdentity{
					Value: &e2sm_mho.BitString{
						Value: uint64(targetNCI),
						Len:   uint32(targetNCILen),
					},
				},
			},
		},
	}

	ueIdentity := e2sm_mho.UeIdentity{
		Value: ueID,
	}

	go func() {
		if e2smMhoControlHandler.ControlHeader, err = e2smMhoControlHandler.CreateMhoControlHeader(servingNCI, servingNCILen, int32(ControlPriority), servingPlmnIDBytes); err == nil {
			if e2smMhoControlHandler.ControlMessage, err = e2smMhoControlHandler.CreateMhoControlMessage(servingCGI, &ueIdentity, targetCGI); err == nil {
				if controlRequest, err := e2smMhoControlHandler.CreateMhoControlRequest(); err == nil {
					c.CtrlReqChans[e2NodeID] <- controlRequest
					log.Infof("tx control, e2NodeID:%v, ueID:%v", e2NodeID, ueID)
				}
			}
		}
	}()

	// Update serving cell metrics
	var cellData *CellData
	cell, err := c.cellStore.Get(ctx, ueData.CGIString)
	if err != nil {
		log.Infof("TRACE: control(): serving cell not found, cgi:%v", ueData.CGIString)
		cellData = &CellData{}
	} else {
		t := cell.Value.(CellData)
		if t.CGIString != ueData.CGIString {
			log.Warnf("bad store data: expected:%v, actual:%v", ueData.CGIString, t.CGIString)
		}
		cellData = &t
	}
	cellData.CGIString = ueData.CGIString
	cellData.CumulativeHandoversOut++
	cellData.NumberRrcConnected--;
	if cellData.NumberRrcConnected < 0 {
		cellData.NumberRrcConnected = 0
	}
	_, err = c.cellStore.Put(ctx, cellData.CGIString, *cellData)
	if err != nil {
		log.Warn(err)
	}

	// Update target cell metrics
	targetCGIString := plmnIDNciToCGI(targetPlmnID, uint64(targetNCI))
	cell, err = c.cellStore.Get(ctx, targetCGIString)
	if err != nil {
		log.Infof("TRACE: control(): target cell not found, cgi:%v", targetCGIString)
		cellData = &CellData{}
	} else {
		t := cell.Value.(CellData)
		if t.CGIString != targetCGIString {
			log.Warnf("bad store data: expected:%v, actual:%v", ueData.CGIString, t.CGIString)
		}
		cellData = &t
	}
	cellData.CGIString = targetCGIString
	cellData.CumulativeHandoversIn++
	cellData.NumberRrcConnected++
	_, err = c.cellStore.Put(ctx, cellData.CGIString, *cellData)
	if err != nil {
		log.Warn(err)
	}

	// Update ue store
	ueData.CGIString = targetCGIString
	_, err = c.ueStore.Put(ctx, ueID, ueData)
	if err != nil {
		log.Warn(err)
	}

	return err

}

func plmnIDBytesToInt(b []byte) uint64 {
	return uint64(b[2]) << 16 | uint64(b[1]) << 8 | uint64(b[0])
}

func plmnIDNciToCGI(plmnID uint64, nci uint64) string {
	return strconv.FormatInt(int64(plmnID<<36 | (nci & 0xfffffffff)), 16)
}

//func getPlmnIDFromIndicationHeader(header *e2sm_mho.E2SmMhoIndicationHeaderFormat1) uint64 {
//	plmnIDBytes := header.GetCgi().GetNrCgi().GetPLmnIdentity().GetValue()
//	return plmnIDBytesToInt(plmnIDBytes)
//}

func getNciFromCellGlobalId(cellGlobalId *e2sm_mho.CellGlobalId) uint64 {
	return cellGlobalId.GetNrCgi().GetNRcellIdentity().GetValue().GetValue()
}

func getPlmnIDBytesFromCellGlobalId(cellGlobalId *e2sm_mho.CellGlobalId) []byte {
	return cellGlobalId.GetNrCgi().GetPLmnIdentity().GetValue()
}

func getCGIFromIndicationHeader(header *e2sm_mho.E2SmMhoIndicationHeaderFormat1) string {
	nci := getNciFromCellGlobalId(header.GetCgi())
	plmnIDBytes := getPlmnIDBytesFromCellGlobalId(header.GetCgi())
	plmnID := plmnIDBytesToInt(plmnIDBytes)
	return plmnIDNciToCGI(plmnID, nci)
}

func getCGIFromMeasReportItem(measReport *e2sm_mho.E2SmMhoMeasurementReportItem) string {
	nci := getNciFromCellGlobalId(measReport.GetCgi())
	plmnIDBytes := getPlmnIDBytesFromCellGlobalId(measReport.GetCgi())
	plmnID := plmnIDBytesToInt(plmnIDBytes)
	return plmnIDNciToCGI(plmnID, nci)
}

func getRsrpFromMeasReport(servingNci uint64, measReport []*e2sm_mho.E2SmMhoMeasurementReportItem) (int32, map[string]int32) {
	var rsrpServing int32
	rsrpNeighbors := make(map[string]int32)

	for _, measReportItem := range measReport {
		if getNciFromCellGlobalId(measReportItem.GetCgi()) == servingNci {
			rsrpServing = measReportItem.GetRsrp().GetValue()
		} else {
			CGIString := getCGIFromMeasReportItem(measReportItem)
			rsrpNeighbors[CGIString] = measReportItem.GetRsrp().GetValue()
		}
	}

	return rsrpServing, rsrpNeighbors
}
