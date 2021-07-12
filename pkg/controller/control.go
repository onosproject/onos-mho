// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: LicenseRef-ONF-Member-1.0

package controller

import (
	e2tapi "github.com/onosproject/onos-api/go/onos/e2t/e2"
	e2api "github.com/onosproject/onos-api/go/onos/e2t/e2/v1beta1"
	"github.com/onosproject/onos-e2-sm/servicemodels/e2sm_mho/pdubuilder"
	e2sm_mho "github.com/onosproject/onos-e2-sm/servicemodels/e2sm_mho/v1/e2sm-mho"
	"google.golang.org/protobuf/proto"
)

type E2SmMhoControlHandler struct {
	NodeID              string
	ServiceModelName    e2tapi.ServiceModelName
	ServiceModelVersion e2tapi.ServiceModelVersion
	ControlMessage      []byte
	ControlHeader       []byte
	ControlAckRequest   e2tapi.ControlAckRequest
	EncodingType        e2tapi.EncodingType
}

func (c *E2SmMhoControlHandler) CreateMhoControlRequest() (*e2api.ControlMessage, error) {
	return &e2api.ControlMessage{
		Header: c.ControlHeader,
		Payload: c.ControlMessage,
		}, nil
}

func (c *E2SmMhoControlHandler) CreateMhoControlHeader(cellID uint64, cellIDLen uint32, priority int32, plmnID []byte) ([]byte, error) {
	eci := &e2sm_mho.BitString{
		Value: cellID,
		Len:   cellIDLen,
	}
	cgi, err := pdubuilder.CreateCellGlobalIDNrCgi(plmnID, eci)
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
