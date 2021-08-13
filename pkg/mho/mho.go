// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: LicenseRef-ONF-Member-1.0

package mho

import (
	"context"
	e2api "github.com/onosproject/onos-api/go/onos/e2t/e2/v1beta1"
	e2sm_mho "github.com/onosproject/onos-e2-sm/servicemodels/e2sm_mho/v1/e2sm-mho"
	"github.com/onosproject/onos-lib-go/pkg/logging"
	appConfig "github.com/onosproject/onos-mho/pkg/config"
	"github.com/onosproject/onos-mho/pkg/store"
	"github.com/onosproject/onos-ric-sdk-go/pkg/e2/indication"
	"github.com/onosproject/rrm-son-lib/pkg/handover"
	rrmid "github.com/onosproject/rrm-son-lib/pkg/model/id"
	"google.golang.org/protobuf/proto"
	"strconv"
	"sync"
)

const (
	ControlPriority = 10
)

var log = logging.GetLogger("mho")

type UeData struct {
	UeID          string
	E2NodeID      string
	CGI           *e2sm_mho.CellGlobalId
	CGIString     string
	RrcState      string
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
	IndMsg      indication.Indication
}

// MhoCtrl is the controller for MHO
type MhoCtrl struct {
	IndChan      chan *E2NodeIndication
	CtrlReqChans map[string]chan *e2api.ControlMessage
	HoCtrl       *HandOverController
	ueStore      store.Store
	cellStore    store.Store
	mu           sync.RWMutex
	cells        map[string]*CellData
}

const a3Report = "a3"
const periodicReport = "periodic"

// NewMhoController returns the struct for MHO logic
func NewMhoController(cfg appConfig.Config, indChan chan *E2NodeIndication, ctrlReqChs map[string]chan *e2api.ControlMessage, ueStore store.Store, cellStore store.Store) *MhoCtrl {
	log.Info("Init MhoController")
	return &MhoCtrl{
		IndChan:      indChan,
		CtrlReqChans: ctrlReqChs,
		HoCtrl:       NewHandOverController(cfg),
		ueStore:      ueStore,
		cellStore:    cellStore,
		cells:        make(map[string]*CellData),
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
						go c.handleMeasReport(ctx, indHeader.GetIndicationHeaderFormat1(), indMessage.GetIndicationMessageFormat1(), e2NodeID, a3Report)
					} else if indMsg.TriggerType == e2sm_mho.MhoTriggerType_MHO_TRIGGER_TYPE_PERIODIC {
						go c.handleMeasReport(ctx, indHeader.GetIndicationHeaderFormat1(), indMessage.GetIndicationMessageFormat1(), e2NodeID, periodicReport)
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

func (c *MhoCtrl) listenHandOver(ctx context.Context) {
	for hoDecision := range c.HoCtrl.HandoverHandler.Chans.OutputChan {
		go func(hoDecision handover.A3HandoverDecision) {
			if err := c.control(ctx, hoDecision); err != nil {
				log.Error(err)
			}
		}(hoDecision)
	}
}

func (c *MhoCtrl) handleMeasReport(ctx context.Context, header *e2sm_mho.E2SmMhoIndicationHeaderFormat1, message *e2sm_mho.E2SmMhoIndicationMessageFormat1, e2NodeID string, reportType string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	ueID := message.GetUeId().GetValue()
	cgi := getCGIFromIndicationHeader(header)
	log.Infof("rx %v ueID:%v cgi:%v", reportType, ueID, cgi)

	// get ue from store (create if it does not exist)
	var ueData *UeData
	ueData = c.getUe(ctx, ueID)
	if ueData == nil {
		ueData = c.createUe(ctx, ueID)
		c.attachUe(ctx, ueData, cgi)
	} else if ueData.CGIString != cgi {
		return
	}


	// update rsrp
	ueData.RsrpServing, ueData.RsrpNeighbors = getRsrpFromMeasReport(getNciFromCellGlobalId(header.GetCgi()), message.MeasReport)

	// update store
	c.setUe(ctx, ueData)

	// If A3 report, then send for HO processing
	if reportType == a3Report {
		// update info needed by control() later
		ueData.CGI = header.GetCgi()
		ueData.E2NodeID = e2NodeID
		c.HoCtrl.Input(ctx, header, message)
	}

}

func (c *MhoCtrl) handleRrcState(ctx context.Context, header *e2sm_mho.E2SmMhoIndicationHeaderFormat1, message *e2sm_mho.E2SmMhoIndicationMessageFormat2) {
	c.mu.Lock()
	defer c.mu.Unlock()
	ueID := message.GetUeId().GetValue()
	cgi := getCGIFromIndicationHeader(header)
	log.Infof("rx rrc ueID:%v cgi:%v", ueID, cgi)

	// get ue from store (create if it does not exist)
	var ueData *UeData
	ueData = c.getUe(ctx, ueID)
	if ueData == nil {
		ueData = c.createUe(ctx, ueID)
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

func (c *MhoCtrl) control(ctx context.Context, ho handover.A3HandoverDecision) error {
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

func (c *MhoCtrl) doHandover(ctx context.Context, ueData *UeData, targetCgi string) {
	servingCgi := ueData.CGIString
	c.attachUe(ctx, ueData, targetCgi)

	targetCell := c.getCell(ctx, targetCgi)
	targetCell.CumulativeHandoversOut++
	c.setCell(ctx, targetCell)

	servingCell := c.getCell(ctx, servingCgi)
	servingCell.CumulativeHandoversIn++
	c.setCell(ctx, servingCell)
}

func (c *MhoCtrl) createUe(ctx context.Context, ueID string) *UeData {
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

func (c *MhoCtrl) getUe(ctx context.Context, ueID string) *UeData {
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

func (c *MhoCtrl) setUe(ctx context.Context, ueData *UeData) {
	_, err := c.ueStore.Put(ctx, ueData.UeID, *ueData)
	if err != nil {
		panic("bad data")
	}
}

func (c *MhoCtrl) attachUe(ctx context.Context, ueData *UeData, cgi string) {
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

func (c *MhoCtrl) detachUe(ctx context.Context, ueData *UeData) {
	for _, cell := range c.cells {
		delete(cell.Ues, ueData.UeID)
	}
}

func (c *MhoCtrl) setUeRrcState(ctx context.Context, ueData *UeData, newRrcState string, cgi string) {
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

func (c *MhoCtrl) createCell(ctx context.Context, cgi string) *CellData {
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

func (c *MhoCtrl) getCell(ctx context.Context, cgi string) *CellData {
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

func (c *MhoCtrl) setCell(ctx context.Context, cellData *CellData) {
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

func getCgiFromHO(ueData *UeData, ho handover.A3HandoverDecision) string {
	servingCGI := ueData.CGI
	servingPlmnIDBytes := servingCGI.GetNrCgi().GetPLmnIdentity().GetValue()
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
		if getNciFromCellGlobalId(measReportItem.GetCgi()) == servingNci {
			rsrpServing = measReportItem.GetRsrp().GetValue()
		} else {
			CGIString := getCGIFromMeasReportItem(measReportItem)
			rsrpNeighbors[CGIString] = measReportItem.GetRsrp().GetValue()
		}
	}
	return rsrpServing, rsrpNeighbors
}
