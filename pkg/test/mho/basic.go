// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: LicenseRef-ONF-Member-1.0

package mho

import (
	e2tapi "github.com/onosproject/onos-api/go/onos/e2t/e2"
	"github.com/onosproject/onos-mho/pkg/controller"
	"github.com/onosproject/onos-mho/pkg/store"
	"github.com/onosproject/onos-mho/pkg/test/utils"
	"github.com/onosproject/onos-ric-sdk-go/pkg/e2/indication"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func (s *TestSuite) TestSubscription(t *testing.T) {
	indCh := make(chan indication.Indication)

	e2IndCh := make(chan *store.E2NodeIndication)
	ctrlReqChMap := make(map[string]chan *e2tapi.ControlRequest)
	mhoCtrl := controller.NewMhoController(e2IndCh, ctrlReqChMap)

	go mhoCtrl.Run()

	sub, err := utils.CreateMhoSubscriptionSingle(indCh, ctrlReqChMap)
	assert.NoError(t, err)

	var nodeIDs []string
	for k := range ctrlReqChMap {
		nodeIDs = append(nodeIDs, k)
	}

	assert.Equal(t, 1, len(nodeIDs))

	numIndMsg := 0

	// Indication message block
	go func() {
		for {
			t.Logf("Num received indication messages: %d\n", numIndMsg)
			if numIndMsg >= 3 {
				t.Log("Received three indication messages - Succeed so far")
				break
			}
			select {
			case indMsg := <- indCh:
				e2IndCh <- &store.E2NodeIndication{
					NodeID: nodeIDs[0],
					IndMsg: indMsg,
				}
				numIndMsg++
			case <- time.After(60 * time.Second):
				t.Error("Indication message did not arrive before timer was expired")
			}
		}
	}()

	numCtrlMsg := 0

	// Control message block
	for {
		t.Logf("Num sent control messages: %d\n", numCtrlMsg)
		if numCtrlMsg >= 2 {
			t.Log("Received three control messages - Succeed so far")
			break
		}
		select {
		case ctrlMsg := <- ctrlReqChMap[nodeIDs[0]]:
			t.Logf("Received control message: %v", ctrlMsg)
			numCtrlMsg++
		case <- time.After(60 * time.Second):
			t.Fatal("Control message did not arrive before timer was expired")
		}
	}

	err = sub.Close()
	assert.NoError(t, err)

}
