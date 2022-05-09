// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package mho

import (
	"context"
	"reflect"
	"strconv"
	"sync"

	e2api "github.com/onosproject/onos-api/go/onos/e2t/e2/v1beta1"
	e2sm_mho "github.com/onosproject/onos-e2-sm/servicemodels/e2sm_mho_go/v2/e2sm-mho-go"
	e2sm_v2_ies "github.com/onosproject/onos-e2-sm/servicemodels/e2sm_mho_go/v2/e2sm-v2-ies"
	"github.com/onosproject/onos-lib-go/pkg/logging"
	appConfig "github.com/onosproject/onos-mho/pkg/config"
	"github.com/onosproject/onos-mho/pkg/store"
	"github.com/onosproject/rrm-son-lib/pkg/handover"
	rrmid "github.com/onosproject/rrm-son-lib/pkg/model/id"
	"google.golang.org/protobuf/proto"
)

const (
	ControlPriority = 10
)

var log = logging.GetLogger()

type UeData struct {
	UeID string // assuming that string carries decimal number
	//ToDo - stop ignoring it once we'veset up UeID treatment all over the SD-RAN
	//UeIDtype      string // represents Gnb, Enb, ngEnb and etc..
	E2NodeID      string
	CGI           *e2sm_v2_ies.Cgi
	CGIString     string
	RrcState      string
	FiveQI        int32
	RsrpServing   int32
	RsrpNeighbors map[string]int32
}

type CellData struct {
	CGIString              string
	CumulativeHandoversIn  int
	CumulativeHandoversOut int
	Ues                    map[string]*UeData
}

type E2NodeIndication struct {
	NodeID      string
	TriggerType e2sm_mho.MhoTriggerType
	IndMsg      e2api.Indication
}

// Ctrl is the controller for MHO
type Ctrl struct {
	IndChan      chan *E2NodeIndication
	CtrlReqChans map[string]chan *e2api.ControlMessage
	HoCtrl       *HandOverController
	ueStore      store.Store
	cellStore    store.Store
	mu           sync.RWMutex
	cells        map[string]*CellData
}

// NewMhoController returns the struct for MHO logic
func NewMhoController(cfg appConfig.Config, indChan chan *E2NodeIndication, ctrlReqChs map[string]chan *e2api.ControlMessage, ueStore store.Store, cellStore store.Store) *Ctrl {
	log.Info("Init MhoController")
	return &Ctrl{
		IndChan:      indChan,
		CtrlReqChans: ctrlReqChs,
		HoCtrl:       NewHandOverController(cfg),
		ueStore:      ueStore,
		cellStore:    cellStore,
		cells:        make(map[string]*CellData),
	}
}

// Run starts to listen Indication message and then save the result to its struct
func (c *Ctrl) Run(ctx context.Context) {
	log.Info("Start MhoController")
	go c.HoCtrl.Run()
	go c.listenIndChan(ctx)
	c.listenHandOver(ctx)
}

func (c *Ctrl) listenIndChan(ctx context.Context) {
	var err error
	for indMsg := range c.IndChan {

		indHeaderByte := indMsg.IndMsg.Header
		indMessageByte := indMsg.IndMsg.Payload
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
					go c.handleRrcState(ctx, indHeader.GetIndicationHeaderFormat1(), indMessage.GetIndicationMessageFormat2())
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

func (c *Ctrl) listenHandOver(ctx context.Context) {
	for hoDecision := range c.HoCtrl.HandoverHandler.Chans.OutputChan {
		go func(hoDecision handover.A3HandoverDecision) {
			if err := c.control(ctx, hoDecision); err != nil {
				log.Error(err)
			}
		}(hoDecision)
	}
}

func (c *Ctrl) handlePeriodicReport(ctx context.Context, header *e2sm_mho.E2SmMhoIndicationHeaderFormat1, message *e2sm_mho.E2SmMhoIndicationMessageFormat1, e2NodeID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	ueID, err := GetUeID(message.GetUeId())
	if err != nil {
		log.Errorf("handlePeriodicReport() couldn't extract UeID: %v", err)
	}
	cgi := getCGIFromIndicationHeader(header)
	log.Infof("rx periodic ueID:%v cgi:%v", ueID, cgi)

	// get ue from store (create if it does not exist)
	var ueData *UeData
	newUe := false
	// Not completely correct conversion of UeID - may be a source of problems
	ueData = c.getUe(ctx, strconv.Itoa(int(ueID)))
	if ueData == nil {
		// Not completely correct conversion of UeID - may be a source of problems
		ueData = c.createUe(ctx, strconv.Itoa(int(ueID)))
		c.attachUe(ctx, ueData, cgi)
		newUe = true
	} else if ueData.CGIString != cgi {
		return
	}

	rsrpServing, rsrpNeighbors := getRsrpFromMeasReport(getNciFromCellGlobalID(header.GetCgi()), message.MeasReport)

	if !newUe && rsrpServing == ueData.RsrpServing && reflect.DeepEqual(rsrpNeighbors, ueData.RsrpNeighbors) {
		return
	}

	// update store
	ueData.RsrpServing, ueData.RsrpNeighbors = rsrpServing, rsrpNeighbors
	c.setUe(ctx, ueData)

}

func (c *Ctrl) handleMeasReport(ctx context.Context, header *e2sm_mho.E2SmMhoIndicationHeaderFormat1, message *e2sm_mho.E2SmMhoIndicationMessageFormat1, e2NodeID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	ueID, err := GetUeID(message.GetUeId())
	if err != nil {
		log.Errorf("handleMeasReport() couldn't extract UeID: %v", err)
	}
	cgi := getCGIFromIndicationHeader(header)
	log.Infof("rx a3 ueID:%v cgi:%v", ueID, cgi)

	// get ue from store (create if it does not exist)
	var ueData *UeData
	// Not completely correct conversion of UeID - may be a source of problems
	ueData = c.getUe(ctx, strconv.Itoa(int(ueID)))
	if ueData == nil {
		// Not completely correct conversion of UeID - may be a source of problems
		ueData = c.createUe(ctx, strconv.Itoa(int(ueID)))
		c.attachUe(ctx, ueData, cgi)
	} else if ueData.CGIString != cgi {
		return
	}

	// update info needed by control() later
	ueData.CGI = header.GetCgi()
	ueData.E2NodeID = e2NodeID

	// update rsrp
	ueData.RsrpServing, ueData.RsrpNeighbors = getRsrpFromMeasReport(getNciFromCellGlobalID(header.GetCgi()), message.MeasReport)

	// update 5QI
	log.Debugf("Going to update 5QI (%v) for UE %v", ueData.FiveQI, ueID)
	for _, item := range message.GetMeasReport() {
		if item.GetFiveQi() != nil && item.GetFiveQi().GetValue() > -1 {
			ueData.FiveQI = item.GetFiveQi().GetValue()
			log.Debugf("Obtained 5QI value %v for UE %v", ueData.FiveQI, ueID)
		}
	}

	// update store
	c.setUe(ctx, ueData)

	// do the real HO processing
	c.HoCtrl.Input(ctx, header, message)

}

func (c *Ctrl) handleRrcState(ctx context.Context, header *e2sm_mho.E2SmMhoIndicationHeaderFormat1, message *e2sm_mho.E2SmMhoIndicationMessageFormat2) {
	c.mu.Lock()
	defer c.mu.Unlock()
	ueID, err := GetUeID(message.GetUeId())
	if err != nil {
		log.Errorf("handleRrcState() couldn't extract UeID: %v", err)
	}
	cgi := getCGIFromIndicationHeader(header)
	log.Infof("rx rrc ueID:%v cgi:%v", ueID, cgi)

	// get ue from store (create if it does not exist)
	var ueData *UeData
	// Not completely correct conversion of UeID - may be a source of problems
	ueData = c.getUe(ctx, strconv.Itoa(int(ueID)))
	if ueData == nil {
		// Not completely correct conversion of UeID - may be a source of problems
		ueData = c.createUe(ctx, strconv.Itoa(int(ueID)))
		c.attachUe(ctx, ueData, cgi)
	} else if ueData.CGIString != cgi {
		return
	}

	// set rrc state (takes care of attach/detach as well)
	newRrcState := message.GetRrcStatus().String()
	c.setUeRrcState(ctx, ueData, newRrcState, cgi)

	// update store
	c.setUe(ctx, ueData)

}

func (c *Ctrl) control(ctx context.Context, ho handover.A3HandoverDecision) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	id := ho.UE.GetID().GetID().(rrmid.UEID).IMSI
	ueID := strconv.Itoa(int(id))

	ueData := c.getUe(ctx, ueID)
	if ueData == nil {
		panic("bad data")
	}

	// do HO
	targetCGIString := getCgiFromHO(ueData, ho)
	c.doHandover(ctx, ueData, targetCGIString)

	// send HO request
	SendHORequest(ueData, ho, c.CtrlReqChans[ueData.E2NodeID])

	return nil

}

func (c *Ctrl) doHandover(ctx context.Context, ueData *UeData, targetCgi string) {
	servingCgi := ueData.CGIString
	c.attachUe(ctx, ueData, targetCgi)

	targetCell := c.getCell(ctx, targetCgi)
	targetCell.CumulativeHandoversOut++
	c.setCell(ctx, targetCell)

	servingCell := c.getCell(ctx, servingCgi)
	servingCell.CumulativeHandoversIn++
	c.setCell(ctx, servingCell)
}

func (c *Ctrl) createUe(ctx context.Context, ueID string) *UeData {
	if len(ueID) == 0 {
		panic("bad data")
	}
	ueData := &UeData{
		UeID:          ueID,
		CGIString:     "",
		RrcState:      e2sm_mho.Rrcstatus_name[int32(e2sm_mho.Rrcstatus_RRCSTATUS_CONNECTED)],
		RsrpNeighbors: make(map[string]int32),
	}
	_, err := c.ueStore.Put(ctx, ueID, *ueData)
	if err != nil {
		log.Warn(err)
	}

	return ueData
}

func (c *Ctrl) getUe(ctx context.Context, ueID string) *UeData {
	var ueData *UeData
	u, err := c.ueStore.Get(ctx, ueID)
	if err != nil || u == nil {
		return nil
	}
	t := u.Value.(UeData)
	ueData = &t
	if ueData.UeID != ueID {
		panic("bad data")
	}

	return ueData
}

func (c *Ctrl) setUe(ctx context.Context, ueData *UeData) {
	_, err := c.ueStore.Put(ctx, ueData.UeID, *ueData)
	if err != nil {
		panic("bad data")
	}
}

func (c *Ctrl) attachUe(ctx context.Context, ueData *UeData, cgi string) {
	// detach ue from current cell
	c.detachUe(ctx, ueData)

	// attach ue to new cell
	ueData.CGIString = cgi
	c.setUe(ctx, ueData)
	cell := c.getCell(ctx, cgi)
	if cell == nil {
		cell = c.createCell(ctx, cgi)
	}
	cell.Ues[ueData.UeID] = ueData
	c.setCell(ctx, cell)
}

func (c *Ctrl) detachUe(ctx context.Context, ueData *UeData) {
	for _, cell := range c.cells {
		delete(cell.Ues, ueData.UeID)
	}
}

func (c *Ctrl) setUeRrcState(ctx context.Context, ueData *UeData, newRrcState string, cgi string) {
	oldRrcState := ueData.RrcState

	if oldRrcState == e2sm_mho.Rrcstatus_name[int32(e2sm_mho.Rrcstatus_RRCSTATUS_CONNECTED)] &&
		newRrcState == e2sm_mho.Rrcstatus_name[int32(e2sm_mho.Rrcstatus_RRCSTATUS_IDLE)] {
		c.detachUe(ctx, ueData)
	} else if oldRrcState == e2sm_mho.Rrcstatus_name[int32(e2sm_mho.Rrcstatus_RRCSTATUS_IDLE)] &&
		newRrcState == e2sm_mho.Rrcstatus_name[int32(e2sm_mho.Rrcstatus_RRCSTATUS_CONNECTED)] {
		c.attachUe(ctx, ueData, cgi)
	}
	ueData.RrcState = newRrcState
}

func (c *Ctrl) createCell(ctx context.Context, cgi string) *CellData {
	if len(cgi) == 0 {
		panic("bad data")
	}
	cellData := &CellData{
		CGIString: cgi,
		Ues:       make(map[string]*UeData),
	}
	_, err := c.cellStore.Put(ctx, cgi, *cellData)
	if err != nil {
		panic("bad data")
	}
	c.cells[cellData.CGIString] = cellData
	return cellData
}

func (c *Ctrl) getCell(ctx context.Context, cgi string) *CellData {
	var cellData *CellData
	cell, err := c.cellStore.Get(ctx, cgi)
	if err != nil || cell == nil {
		return nil
	}
	t := cell.Value.(CellData)
	if t.CGIString != cgi {
		panic("bad data")
	}
	cellData = &t
	return cellData
}

func (c *Ctrl) setCell(ctx context.Context, cellData *CellData) {
	if len(cellData.CGIString) == 0 {
		panic("bad data")
	}
	_, err := c.cellStore.Put(ctx, cellData.CGIString, *cellData)
	if err != nil {
		panic("bad data")
	}
}

func plmnIDBytesToInt(b []byte) uint64 {
	return uint64(b[2])<<16 | uint64(b[1])<<8 | uint64(b[0])
}

func plmnIDNciToCGI(plmnID uint64, nci uint64) string {
	return strconv.FormatInt(int64(plmnID<<36|(nci&0xfffffffff)), 16)
}

//func getPlmnIDFromIndicationHeader(header *e2sm_mho.E2SmMhoIndicationHeaderFormat1) uint64 {
//	plmnIDBytes := header.GetCgi().GetNrCgi().GetPLmnIdentity().GetValue()
//	return plmnIDBytesToInt(plmnIDBytes)
//}

func getNciFromCellGlobalID(cellGlobalID *e2sm_v2_ies.Cgi) uint64 {
	return BitStringToUint64(cellGlobalID.GetNRCgi().GetNRcellIdentity().GetValue().GetValue(), int(cellGlobalID.GetNRCgi().GetNRcellIdentity().GetValue().GetLen()))
}

func getPlmnIDBytesFromCellGlobalID(cellGlobalID *e2sm_v2_ies.Cgi) []byte {
	return cellGlobalID.GetNRCgi().GetPLmnidentity().GetValue()
}

func getCGIFromIndicationHeader(header *e2sm_mho.E2SmMhoIndicationHeaderFormat1) string {
	nci := getNciFromCellGlobalID(header.GetCgi())
	plmnIDBytes := getPlmnIDBytesFromCellGlobalID(header.GetCgi())
	plmnID := plmnIDBytesToInt(plmnIDBytes)
	return plmnIDNciToCGI(plmnID, nci)
}

func getCGIFromMeasReportItem(measReport *e2sm_mho.E2SmMhoMeasurementReportItem) string {
	nci := getNciFromCellGlobalID(measReport.GetCgi())
	plmnIDBytes := getPlmnIDBytesFromCellGlobalID(measReport.GetCgi())
	plmnID := plmnIDBytesToInt(plmnIDBytes)
	return plmnIDNciToCGI(plmnID, nci)
}

func getCgiFromHO(ueData *UeData, ho handover.A3HandoverDecision) string {
	servingCGI := ueData.CGI
	servingPlmnIDBytes := servingCGI.GetNRCgi().GetPLmnidentity().GetValue()
	servingPlmnID := plmnIDBytesToInt(servingPlmnIDBytes)
	targetPlmnID := servingPlmnID
	targetNCI, err := strconv.Atoi(ho.TargetCell.GetID().String())
	if err != nil {
		panic("bad data")
	}
	return plmnIDNciToCGI(targetPlmnID, uint64(targetNCI))
}

func getRsrpFromMeasReport(servingNci uint64, measReport []*e2sm_mho.E2SmMhoMeasurementReportItem) (int32, map[string]int32) {
	var rsrpServing int32
	rsrpNeighbors := make(map[string]int32)

	for _, measReportItem := range measReport {
		if getNciFromCellGlobalID(measReportItem.GetCgi()) == servingNci {
			rsrpServing = measReportItem.GetRsrp().GetValue()
		} else {
			CGIString := getCGIFromMeasReportItem(measReportItem)
			rsrpNeighbors[CGIString] = measReportItem.GetRsrp().GetValue()
		}
	}

	return rsrpServing, rsrpNeighbors
}
