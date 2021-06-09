// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: LicenseRef-ONF-Member-1.0

package controller

import (
	"github.com/onosproject/rrm-son-lib/pkg/handover"
	measurement2 "github.com/onosproject/rrm-son-lib/pkg/measurement"
	"github.com/onosproject/rrm-son-lib/pkg/model/device"
)

const (
	A3OffsetRangeConfigPath        = "/hoParameters/A3OffsetRange"
	HysteresisRangeConfigPath      = "/hoParameters/HysteresisRange"
	CellIndividualOffsetConfigPath = "/hoParameters/CellIndividualOffset"
	FrequencyOffsetConfigPath      = "/hoParameters/FrequencyOffset"
	TimeToTriggerConfigPath        = "/hoParameters/TimeToTrigger"
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

func NewHandOverController() *HandOverController {
	return &HandOverController{
		UeChan:          make(chan device.UE),
		A3Handler:       measurement2.NewMeasEventA3Handler(),
		HandoverHandler: handover.NewA3HandoverHandler(),
	}
}

func (h *HandOverController) Run() {

	go h.A3Handler.Run()

	go h.HandoverHandler.Run()

	// forward event a3 measurement only to handover handler
	for ue := range h.A3Handler.Chans.OutputChan {
		h.HandoverHandler.Chans.InputChan <- ue
	}

}
