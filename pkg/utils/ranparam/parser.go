// SPDX-FileCopyrightText: 2022-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package ranparam

import (
	e2smrcies "github.com/onosproject/onos-e2-sm/servicemodels/e2sm_rc/v1/e2sm-rc-ies"
	"github.com/onosproject/onos-lib-go/pkg/errors"
)

func GetRanParameterValue(item *e2smrcies.RanparameterValueType, ranParamID int64) (*e2smrcies.RanparameterValue, error) {
	if item.GetRanPChoiceStructure() == nil {
		return nil, errors.NewNotFound("ran parameter ID %v not found", ranParamID)
	}

	for _, i := range item.GetRanPChoiceStructure().GetRanParameterStructure().GetSequenceOfRanParameters() {
		if i.RanParameterId.Value == ranParamID {
			return i.GetRanParameterValueType().GetRanPChoiceElementFalse().GetRanParameterValue(), nil
		} else {
			v, err := GetRanParameterValue(i.GetRanParameterValueType(), ranParamID)
			if err == nil {
				return v, nil
			}
		}
	}

	return nil, errors.NewNotFound("ran parameter ID %v not found", ranParamID)
}
