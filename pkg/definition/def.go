// SPDX-FileCopyrightText: 2022-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package definition

const (
	// TargetPrimaryCellIDRANParameterID Target Primary Cell ID RAN parameter ID
	TargetPrimaryCellIDRANParameterID = 1
	// TargetCellRANParameterID Choice of Target Cell RAN parameter ID
	TargetCellRANParameterID = 2
	// NRCellRANParameterID NR Cell RAN parameter ID
	NRCellRANParameterID = 3
	// NRCGIRANParameterID NR CGI RAN parameter ID
	NRCGIRANParameterID = 4
	// EUTRACellRANParameterID E-UTRA Cell RAN parameter ID
	EUTRACellRANParameterID = 5
	// EUTRACGIRANParameterID E-UTRA CGI RAN parameter ID
	EUTRACGIRANParameterID = 6
)

const (
	E2EventTriggerConditionID             = 1
	RrcMessageIDNrUlDcchMeasurementReport = 0
	AssociatedUeEventID                   = 2
	RicInsertStyleTypeID                  = 3 // connected mode mobility request
	RicInsertIndicationID                 = 1 // handover control request
	RicActionID                           = 3
	ControlActionID                       = 1
)
