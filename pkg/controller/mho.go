// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: LicenseRef-ONF-Member-1.0

package controller

import (
	e2tapi "github.com/onosproject/onos-api/go/onos/e2t/e2"
	e2sm_mho "github.com/onosproject/onos-e2-sm/servicemodels/e2sm_mho/v1/e2sm-mho"
	"github.com/onosproject/onos-lib-go/pkg/logging"
	"github.com/onosproject/onos-mho/pkg/southbound/ricapie2"
	"github.com/onosproject/onos-mho/pkg/store"
	"google.golang.org/protobuf/proto"
)

const (
	ControlPriority                = 10
)

var log = logging.GetLogger("controller", "mho")

// MhoCtrl is the controller for MHO
type MhoCtrl struct {
	IndChan      chan *store.E2NodeIndication
	CtrlReqChans map[string]chan *e2tapi.ControlRequest
	HoCtrl      *HandOverController
}

// NewMhoController returns the struct for MHO logic
func NewMhoController(indChan chan *store.E2NodeIndication, ctrlReqChs map[string]chan *e2tapi.ControlRequest) *MhoCtrl {
	log.Info("Start onos-mho Application Controller")
	hoCtrl := NewHandOverController()
	return &MhoCtrl{
		IndChan:      indChan,
		CtrlReqChans: ctrlReqChs,
		HoCtrl:      hoCtrl,
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
		e2NodeID := indMsg.NodeID

		indHeader := e2sm_mho.E2SmMhoIndicationHeader{}
		if err = proto.Unmarshal(indHeaderByte, &indHeader); err == nil {
			indMessage := e2sm_mho.E2SmMhoIndicationMessage{}
			if err = proto.Unmarshal(indMessageByte, &indMessage); err == nil {
				log.Debugf("MHO indication header: %v", indHeader.GetIndicationHeaderFormat1())
				log.Debugf("MHO indication message: %v", indMessage.GetIndicationMessageFormat1())
				switch x := indMessage.E2SmMhoIndicationMessage.(type) {
				case *e2sm_mho.E2SmMhoIndicationMessage_IndicationMessageFormat1:
					c.handleIndMsgFormat1(indHeader.GetIndicationHeaderFormat1(), indMessage.GetIndicationMessageFormat1(), e2NodeID)
				case *e2sm_mho.E2SmMhoIndicationMessage_IndicationMessageFormat2:
					c.handleIndMsgFormat2(indHeader.GetIndicationHeaderFormat1(), indMessage.GetIndicationMessageFormat1(), e2NodeID)
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

//*E2SmMhoIndicationMessageFormat1
func (c *MhoCtrl) handleIndMsgFormat1(header *e2sm_mho.E2SmMhoIndicationHeaderFormat1, message *e2sm_mho.E2SmMhoIndicationMessageFormat1, e2NodeID string) {
	// TODO
	if err := c.control(header, message, e2NodeID); err != nil {
		log.Error(err)
	}
}

func (c *MhoCtrl) handleIndMsgFormat2(header *e2sm_mho.E2SmMhoIndicationHeaderFormat1, message *e2sm_mho.E2SmMhoIndicationMessageFormat1, e2NodeID string) {
	// TODO
	log.Info("Ignore indication message format 2")
}

func (c *MhoCtrl) control(header *e2sm_mho.E2SmMhoIndicationHeaderFormat1, message *e2sm_mho.E2SmMhoIndicationMessageFormat1, e2NodeID string) error {
	var err error

	// send control message to the E2Node
	e2smMhoControlHandler := &E2SmMhoControlHandler{
		NodeID:              e2NodeID,
		EncodingType:        e2tapi.EncodingType_PROTO,
		ServiceModelName:    ricapie2.ServiceModelName,
		ServiceModelVersion: ricapie2.ServiceModelVersion,
		ControlAckRequest:   e2tapi.ControlAckRequest_NO_ACK,
	}

	cellID := header.GetCgi().GetNrCgi().GetNRcellIdentity().GetValue().GetValue()
	cellIDLen := header.GetCgi().GetNrCgi().GetNRcellIdentity().GetValue().GetLen()
	plmnID := header.GetCgi().GetNrCgi().GetPLmnIdentity().GetValue()
	servingCGI := header.GetCgi()
	ueID := message.GetUeId()

	targetCGI := message.GetMeasReport()[0].GetCgi()

	log.Debugf("cellIdLen:%d, plmnID:%v, len:%d", cellIDLen, plmnID, len(plmnID))

	if e2smMhoControlHandler.ControlHeader, err = e2smMhoControlHandler.CreateMhoControlHeader(cellID, cellIDLen, int32(ControlPriority), plmnID); err == nil {
		if e2smMhoControlHandler.ControlMessage, err = e2smMhoControlHandler.CreateMhoControlMessage(servingCGI, ueID, targetCGI); err == nil {
			if controlRequest, err := e2smMhoControlHandler.CreateMhoControlRequest(); err == nil {
				log.Debugf("Control Request message for e2NodeID %s: %v", e2NodeID, controlRequest)
				c.CtrlReqChans[e2NodeID] <- controlRequest
			}
		}
	}

	return err

}
