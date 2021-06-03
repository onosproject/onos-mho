// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: LicenseRef-ONF-Member-1.0

package controller

const (
	A3OffsetRangeConfigPath        = "/hoParameters/A3OffsetRange"
	HysteresisRangeConfigPath      = "/hoParameters/HysteresisRange"
	CellIndividualOffsetConfigPath = "/hoParameters/CellIndividualOffset"
	FrequencyOffsetConfigPath      = "hoParameters/FrequencyOffset"
	TimeToTriggerConfigPath        = "hoParameters/TimeToTrigger"
)

// HoParms is the handover parameters
type HandOver struct {
	A3OffsetRange        uint64
	HysteresisRange      uint64
	CellIndividualOffset string
	FrequencyOffset      string
	TimeToTrigger        string
}