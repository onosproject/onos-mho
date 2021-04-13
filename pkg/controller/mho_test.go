// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: LicenseRef-ONF-Member-1.0

package controller

import (
	"fmt"
	e2tapi "github.com/onosproject/onos-api/go/onos/e2t/e2"
	"github.com/onosproject/onos-mho/pkg/store"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNewPciController(t *testing.T) {
	sampleIndChan := make(chan *store.E2NodeIndication)
	sampleCtrlReqChans := make(map[string]chan *e2tapi.ControlRequest)
	samplePciController := MhoCtrl{
		IndChan: sampleIndChan,
		CtrlReqChans: sampleCtrlReqChans,
	}
	targetPciController := NewMhoController(sampleIndChan, sampleCtrlReqChans)
	fmt.Printf("samplePciController: %v\n", &samplePciController)
	fmt.Printf("targetPciController: %v\n", targetPciController)

	assert.Equal(t, &samplePciController, targetPciController)
}
