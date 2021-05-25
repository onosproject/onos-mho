// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: LicenseRef-ONF-Member-1.0

package controller

import (
	e2tapi "github.com/onosproject/onos-api/go/onos/e2t/e2"
	e2sm_mho "github.com/onosproject/onos-e2-sm/servicemodels/e2sm_mho/v1/e2sm-mho"
	"github.com/onosproject/onos-lib-go/pkg/logging"
	"github.com/onosproject/onos-mho/pkg/store"
	"google.golang.org/protobuf/proto"
)

var log = logging.GetLogger("controller", "mho")

// MhoCtrl is the controller for the KPI monitoring
type MhoCtrl struct {
	IndChan           chan *store.E2NodeIndication
	CtrlReqChans      map[string]chan *e2tapi.ControlRequest
}

// NewPciController returns the struct for PCI logic
func NewMhoController(indChan chan *store.E2NodeIndication, ctrlReqChs map[string]chan *e2tapi.ControlRequest) *MhoCtrl {
	log.Info("Start onos-mho Application Controller")
	return &MhoCtrl{
		IndChan:         indChan,
		CtrlReqChans:    ctrlReqChs,
	}
}

// Run starts to listen Indication message and then save the result to its struct
func (c *MhoCtrl) Run() {
	c.listenIndChan()
}

func (c *MhoCtrl) listenIndChan() {
	var err error
	for indMsg := range c.IndChan {
		log.Debugf("Raw message: %v", indMsg)

		indHeaderByte := indMsg.IndMsg.Payload.Header
		indMessageByte := indMsg.IndMsg.Payload.Message

		indHeader := e2sm_mho.E2SmMhoIndicationHeader{}
		err = proto.Unmarshal(indHeaderByte, &indHeader)
		if err != nil {
			log.Errorf("Error - Unmarshalling header protobytes to struct: %v", err)
		}

		indMessage := e2sm_mho.E2SmMhoIndicationMessage{}
		err = proto.Unmarshal(indMessageByte, &indMessage)
		if err != nil {
			log.Errorf("Error - Unmarshalling message protobytes to struct: %v", err)
		}

		c.handleIndMsg(indHeader.GetIndicationHeaderFormat1(), indMessage.GetIndicationMessageFormat1())

	}
}

func (c *MhoCtrl) handleIndMsg(header *e2sm_mho.E2SmMhoIndicationHeaderFormat1, message *e2sm_mho.E2SmMhoIndicationMessageFormat1) {
	// TODO
	log.Debugf("MHO indication header: %v", header)
	log.Debugf("MHO indication message: %v", message)
}
