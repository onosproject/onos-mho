// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: LicenseRef-ONF-Member-Only-1.0

package mho

import (
	e2tapi "github.com/onosproject/onos-api/go/onos/e2t/e2"
	e2api "github.com/onosproject/onos-api/go/onos/e2t/e2/v1beta1"
	"github.com/onosproject/onos-e2-sm/servicemodels/e2sm_mho_go/pdubuilder"
	e2sm_mho "github.com/onosproject/onos-e2-sm/servicemodels/e2sm_mho_go/v2/e2sm-mho-go"
	"github.com/onosproject/onos-lib-go/api/asn1/v1/asn1"
	"github.com/onosproject/rrm-son-lib/pkg/handover"
	"google.golang.org/protobuf/proto"
	"strconv"
)

type E2SmMhoControlHandler struct {
	NodeID            string
	ControlMessage    []byte
	ControlHeader     []byte
	ControlAckRequest e2tapi.ControlAckRequest
}

func (c *E2SmMhoControlHandler) CreateMhoControlRequest() (*e2api.ControlMessage, error) {
	return &e2api.ControlMessage{
		Header:  c.ControlHeader,
		Payload: c.ControlMessage,
	}, nil
}

func (c *E2SmMhoControlHandler) CreateMhoControlHeader(cellID []byte, cellIDLen uint32, priority int32, plmnID []byte) ([]byte, error) {
	eci := &asn1.BitString{
		Value: cellID,
		Len:   cellIDLen,
	}
	cgi, err := pdubuilder.CreateCellGlobalIDNrCGI(plmnID, eci)
	log.Debugf("eci: %v", eci)
	log.Debugf("cgi: %v", cgi)
	if err != nil {
		return []byte{}, err
	}

	newE2SmMhoPdu, err := pdubuilder.CreateE2SmMhoControlHeader(priority)

	log.Debugf("newE2SmMhoPdu: %v", newE2SmMhoPdu)
	if err != nil {
		return []byte{}, err
	}

	err = newE2SmMhoPdu.Validate()
	if err != nil {
		return []byte{}, err
	}

	protoBytes, err := proto.Marshal(newE2SmMhoPdu)
	if err != nil {
		return []byte{}, err
	}

	return protoBytes, nil
}

func (c *E2SmMhoControlHandler) CreateMhoControlMessage(servingCgi *e2sm_mho.CellGlobalId, uedID *e2sm_mho.UeIdentity, targetCgi *e2sm_mho.CellGlobalId) ([]byte, error) {

	var err error

	if newE2SmMhoPdu, err := pdubuilder.CreateE2SmMhoControlMessage(servingCgi, uedID, targetCgi); err == nil {
		if err = newE2SmMhoPdu.Validate(); err == nil {
			if protoBytes, err := proto.Marshal(newE2SmMhoPdu); err == nil {
				return protoBytes, nil
			}
		}
	}

	return []byte{}, err

}

func SendHORequest(ueData *UeData, ho handover.A3HandoverDecision, ctrlReqChan chan *e2api.ControlMessage) {
	e2NodeID := ueData.E2NodeID
	servingCGI := ueData.CGI
	servingPlmnIDBytes := servingCGI.GetNrCgi().GetPLmnIdentity().GetValue()
	servingNCI := servingCGI.GetNrCgi().GetNRcellIdentity().GetValue().GetValue()
	servingNCILen := servingCGI.GetNrCgi().GetNRcellIdentity().GetValue().GetLen()
	targetPlmnIDBytes := servingPlmnIDBytes
	targetNCI, err := strconv.Atoi(ho.TargetCell.GetID().String())
	if err != nil {
		panic("bad data")
	}
	targetNCILen := 36

	e2smMhoControlHandler := &E2SmMhoControlHandler{
		NodeID:            e2NodeID,
		ControlAckRequest: e2tapi.ControlAckRequest_NO_ACK,
	}

	targetCGI := &e2sm_mho.CellGlobalId{
		CellGlobalId: &e2sm_mho.CellGlobalId_NrCgi{
			NrCgi: &e2sm_mho.Nrcgi{
				PLmnIdentity: &e2sm_mho.PlmnIdentity{
					Value: targetPlmnIDBytes,
				},
				NRcellIdentity: &e2sm_mho.NrcellIdentity{
					Value: &asn1.BitString{
						Value: Uint64ToBitString(uint64(targetNCI), targetNCILen),
						Len:   uint32(targetNCILen),
					},
				},
			},
		},
	}

	ueIdentity := e2sm_mho.UeIdentity{
		Value: []byte(ueData.UeID),
	}

	go func() {
		if e2smMhoControlHandler.ControlHeader, err = e2smMhoControlHandler.CreateMhoControlHeader(servingNCI, servingNCILen, int32(ControlPriority), servingPlmnIDBytes); err == nil {
			if e2smMhoControlHandler.ControlMessage, err = e2smMhoControlHandler.CreateMhoControlMessage(servingCGI, &ueIdentity, targetCGI); err == nil {
				if controlRequest, err := e2smMhoControlHandler.CreateMhoControlRequest(); err == nil {
					ctrlReqChan <- controlRequest
					log.Infof("tx control, e2NodeID:%v, ueID:%v", e2NodeID, ueData.UeID)
				}
			}
		}
	}()

}
