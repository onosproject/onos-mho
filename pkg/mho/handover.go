// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package mho

import (
	"bytes"
	"context"
	e2sm_mho "github.com/onosproject/onos-e2-sm/servicemodels/e2sm_mho_go/v2/e2sm-mho-go"
	appConfig "github.com/onosproject/onos-mho/pkg/config"
	"github.com/onosproject/rrm-son-lib/pkg/handover"
	measurement2 "github.com/onosproject/rrm-son-lib/pkg/measurement"
	"github.com/onosproject/rrm-son-lib/pkg/model/device"
	rrmid "github.com/onosproject/rrm-son-lib/pkg/model/id"
	"github.com/onosproject/rrm-son-lib/pkg/model/measurement"
	meastype "github.com/onosproject/rrm-son-lib/pkg/model/measurement/type"
)

// HandOverController is the handover controller
type HandOverController struct {
	A3OffsetRange        uint64
	HysteresisRange      uint64
	CellIndividualOffset uint64
	FrequencyOffset      uint64
	TimeToTrigger        uint64
	UeChan               chan device.UE
	A3Handler            *measurement2.MeasEventA3Handler
	HandoverHandler      *handover.A3HandoverHandler
}

func NewHandOverController(cfg appConfig.Config) *HandOverController {
	log.Info("Init HandOverController")
	return &HandOverController{
		A3OffsetRange:        cfg.GetA3OffsetRange(),
		HysteresisRange:      cfg.GetHysteresisRange(),
		CellIndividualOffset: cfg.GetCellIndividualOffset(),
		FrequencyOffset:      cfg.GetFrequencyOffset(),
		TimeToTrigger:        cfg.GetTimeToTrigger(),
		UeChan:               make(chan device.UE),
		A3Handler:            measurement2.NewMeasEventA3Handler(),
		HandoverHandler:      handover.NewA3HandoverHandler(),
	}
}

func (h *HandOverController) Run() {
	log.Info("Start A3Handler")
	go h.A3Handler.Run()

	log.Info("Start HandoverHandler")
	go h.HandoverHandler.Run()

	log.Info("Start forwarding A3Handler events to HandoverHandler")
	for ue := range h.A3Handler.Chans.OutputChan {
		h.HandoverHandler.Chans.InputChan <- ue
	}

}

func (h *HandOverController) Input(ctx context.Context, header *e2sm_mho.E2SmMhoIndicationHeaderFormat1, message *e2sm_mho.E2SmMhoIndicationMessageFormat1) {
	ueID, err := GetUeID(message.GetUeId())
	if err != nil {
		log.Errorf("handlePeriodicReport() couldn't extract UeID: %v", err)
	}

	ecgiSCell := rrmid.NewECGI(BitStringToUint64(header.GetCgi().GetNRCgi().GetNRcellIdentity().GetValue().GetValue(), int(header.GetCgi().GetNRCgi().GetNRcellIdentity().GetValue().GetLen())))
	scell := device.NewCell(
		ecgiSCell,
		meastype.A3OffsetRange(h.A3OffsetRange),
		meastype.HysteresisRange(h.HysteresisRange),
		meastype.QOffsetRange(h.CellIndividualOffset),
		meastype.QOffsetRange(h.FrequencyOffset),
		meastype.TimeToTriggerRange(h.TimeToTrigger))
	cscellList := make([]device.Cell, 0)

	ue := device.NewUE(rrmid.NewUEID(uint64(ueID), uint32(0),
		BitStringToUint64(header.GetCgi().GetNRCgi().GetNRcellIdentity().GetValue().GetValue(), int(header.GetCgi().GetNRCgi().GetNRcellIdentity().GetValue().GetLen()))),
		scell, nil)

	for _, measReport := range message.MeasReport {
		if bytes.Equal(measReport.GetCgi().GetNRCgi().GetNRcellIdentity().GetValue().GetValue(), header.GetCgi().GetNRCgi().GetNRcellIdentity().GetValue().GetValue()) {
			ue.GetMeasurements()[ecgiSCell.String()] = measurement.NewMeasEventA3(ecgiSCell, measurement.RSRP(measReport.GetRsrp().GetValue()))
		} else {
			ecgiCSCell := rrmid.NewECGI(BitStringToUint64(measReport.GetCgi().GetNRCgi().GetNRcellIdentity().GetValue().GetValue(), int(measReport.GetCgi().GetNRCgi().GetNRcellIdentity().GetValue().GetLen())))
			cscell := device.NewCell(
				ecgiCSCell,
				meastype.A3OffsetRange(h.A3OffsetRange),
				meastype.HysteresisRange(h.HysteresisRange),
				meastype.QOffsetRange(h.CellIndividualOffset),
				meastype.QOffsetRange(h.FrequencyOffset),
				meastype.TimeToTriggerRange(h.TimeToTrigger))
			cscellList = append(cscellList, cscell)
			ue.GetMeasurements()[ecgiCSCell.String()] = measurement.NewMeasEventA3(ecgiCSCell, measurement.RSRP(measReport.GetRsrp().GetValue()))
		}
	}
	ue.SetCSCells(cscellList)

	h.A3Handler.Chans.InputChan <- ue
}
