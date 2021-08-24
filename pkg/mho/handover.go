// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: LicenseRef-ONF-Member-1.0

package mho

import (
	"context"
	e2sm_mho "github.com/onosproject/onos-e2-sm/servicemodels/e2sm_mho/v1/e2sm-mho"
	appConfig "github.com/onosproject/onos-mho/pkg/config"
	"github.com/onosproject/rrm-son-lib/pkg/handover"
	measurement2 "github.com/onosproject/rrm-son-lib/pkg/measurement"
	"github.com/onosproject/rrm-son-lib/pkg/model/device"
	rrmid "github.com/onosproject/rrm-son-lib/pkg/model/id"
	"github.com/onosproject/rrm-son-lib/pkg/model/measurement"
	meastype "github.com/onosproject/rrm-son-lib/pkg/model/measurement/type"
	"strconv"
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
	ueID := message.GetUeId().GetValue()

	ecgiSCell := rrmid.NewECGI(header.GetCgi().GetNrCgi().GetNRcellIdentity().GetValue().GetValue())
	scell := device.NewCell(
		ecgiSCell,
		meastype.A3OffsetRange(h.A3OffsetRange),
		meastype.HysteresisRange(h.HysteresisRange),
		meastype.QOffsetRange(h.CellIndividualOffset),
		meastype.QOffsetRange(h.FrequencyOffset),
		meastype.TimeToTriggerRange(h.TimeToTrigger))
	cscellList := make([]device.Cell, 0)

	ueidInt, err := strconv.Atoi(ueID)
	if err != nil {
		log.Error(err)
		return
	}
	ue := device.NewUE(rrmid.NewUEID(uint64(ueidInt), uint32(0), header.GetCgi().GetNrCgi().GetNRcellIdentity().GetValue().GetValue()), scell, nil)

	for _, measReport := range message.MeasReport {
		if measReport.GetCgi().GetNrCgi().GetNRcellIdentity().GetValue().GetValue() == header.GetCgi().GetNrCgi().GetNRcellIdentity().GetValue().GetValue() {
			ue.GetMeasurements()[ecgiSCell.String()] = measurement.NewMeasEventA3(ecgiSCell, measurement.RSRP(measReport.GetRsrp().GetValue()))
		} else {
			ecgiCSCell := rrmid.NewECGI(measReport.GetCgi().GetNrCgi().GetNRcellIdentity().GetValue().GetValue())
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
