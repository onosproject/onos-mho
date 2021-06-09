// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: LicenseRef-ONF-Member-1.0

package controller

import (
	"fmt"
	e2tapi "github.com/onosproject/onos-api/go/onos/e2t/e2"
	"github.com/onosproject/onos-mho/pkg/store"
	"testing"
)

func TestNewMhoController(t *testing.T) {
	sampleIndChan := make(chan *store.E2NodeIndication)
	sampleCtrlReqChans := make(map[string]chan *e2tapi.ControlRequest)
	hoCtrl := NewHandOverController()
	sampleMhoController := MhoCtrl{
		IndChan: sampleIndChan,
		CtrlReqChans: sampleCtrlReqChans,
		HoCtrl: hoCtrl,
	}
	targetMhoController := NewMhoController(sampleIndChan, sampleCtrlReqChans)
	fmt.Printf("sampleMhoController: %v\n", &sampleMhoController)
	fmt.Printf("targetMhoController: %v\n", targetMhoController)

	// TODO - fix test
	//assert.Equal(t, &sampleMhoController, targetMhoController)
}
