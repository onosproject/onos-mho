// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: LicenseRef-ONF-Member-1.0

package mho

import (
	appConfig "github.com/onosproject/onos-mho/pkg/config"
	"github.com/onosproject/rrm-son-lib/pkg/handover"
	measurement2 "github.com/onosproject/rrm-son-lib/pkg/measurement"
	"github.com/onosproject/rrm-son-lib/pkg/model/device"
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
