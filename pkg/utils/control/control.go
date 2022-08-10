// SPDX-FileCopyrightText: 2022-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package control

import (
	"github.com/onosproject/onos-e2-sm/servicemodels/e2sm_rc/pdubuilder"
	e2smcommonies "github.com/onosproject/onos-e2-sm/servicemodels/e2sm_rc/v1/e2sm-common-ies"
	e2smrcies "github.com/onosproject/onos-e2-sm/servicemodels/e2sm_rc/v1/e2sm-rc-ies"
	"github.com/onosproject/onos-mho/pkg/definition"
	"google.golang.org/protobuf/proto"
)

func CreateRcControlHeader(ueID *e2smcommonies.Ueid) ([]byte, error) {
	ctrlHdrFormat1, err := pdubuilder.CreateE2SmRcControlHeaderFormat1(ueID, definition.RicInsertStyleTypeID, definition.ControlActionID)
	if err != nil {
		return nil, err
	}

	err = ctrlHdrFormat1.Validate()
	if err != nil {
		return nil, err
	}

	protoByte, err := proto.Marshal(ctrlHdrFormat1)
	if err != nil {
		return nil, err
	}

	return protoByte, nil
}

func CreateRcControlMessage(tgtCellID string) ([]byte, error) {
	nrcgiRanParamValuePrint, err := pdubuilder.CreateRanparameterValuePrintableString(tgtCellID)
	if err != nil {
		return nil, err
	}
	nrCgiRanParamValue, err := pdubuilder.CreateRanparameterValueTypeChoiceElementFalse(nrcgiRanParamValuePrint)
	if err != nil {
		return nil, err
	}
	nrCgiRanParamValueItem, err := pdubuilder.CreateRanparameterStructureItem(definition.NRCGIRANParameterID, nrCgiRanParamValue)
	if err != nil {
		return nil, err
	}
	nrCellRanParamValue, err := pdubuilder.CreateRanParameterStructure([]*e2smrcies.RanparameterStructureItem{nrCgiRanParamValueItem})
	if err != nil {
		return nil, err
	}
	nrCellRanParamValueType, err := pdubuilder.CreateRanparameterValueTypeChoiceStructure(nrCellRanParamValue)
	if err != nil {
		return nil, err
	}
	nrCellRanParamValueItem, err := pdubuilder.CreateRanparameterStructureItem(definition.NRCellRANParameterID, nrCellRanParamValueType)
	if err != nil {
		return nil, err
	}
	targetCellRanParamValue, err := pdubuilder.CreateRanParameterStructure([]*e2smrcies.RanparameterStructureItem{nrCellRanParamValueItem})
	if err != nil {
		return nil, err
	}
	targetCellRanParamValueType, err := pdubuilder.CreateRanparameterValueTypeChoiceStructure(targetCellRanParamValue)
	if err != nil {
		return nil, err
	}
	targetCellRanParamValueItem, err := pdubuilder.CreateRanparameterStructureItem(definition.TargetCellRANParameterID, targetCellRanParamValueType)
	if err != nil {
		return nil, err
	}
	targetPrimaryCellIDRanParamValue, err := pdubuilder.CreateRanParameterStructure([]*e2smrcies.RanparameterStructureItem{targetCellRanParamValueItem})
	if err != nil {
		return nil, err
	}
	targetPrimaryCellIDRanParamValueType, err := pdubuilder.CreateRanparameterValueTypeChoiceStructure(targetPrimaryCellIDRanParamValue)
	if err != nil {
		return nil, err
	}
	targetPrimaryCellIDRanParamValueItem, err := pdubuilder.CreateE2SmRcControlMessageFormat1Item(definition.TargetPrimaryCellIDRANParameterID, targetPrimaryCellIDRanParamValueType)
	if err != nil {
		return nil, err
	}

	rpl := []*e2smrcies.E2SmRcControlMessageFormat1Item{targetPrimaryCellIDRanParamValueItem}

	e2smRcControlMessage, err := pdubuilder.CreateE2SmRcControlMessageFormat1(rpl)
	if err != nil {
		return nil, err
	}

	err = e2smRcControlMessage.Validate()
	if err != nil {
		return nil, err
	}

	protoBytes, err := proto.Marshal(e2smRcControlMessage)
	if err != nil {
		return nil, err
	}

	return protoBytes, nil
}
