// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: LicenseRef-ONF-Member-1.0

package metrics

import (
	e2sm_mho "github.com/onosproject/onos-e2-sm/servicemodels/e2sm_mho/v1/e2sm-mho"
)

// Key metric key
type Key struct {
	UeID		*e2sm_mho.UeIdentity
}

// Entry entry of metrics store
type Entry struct {
	Key   Key
	Value interface{}
}

// MetricEvent a metric event
type MetricEvent int

const (
	// None none cell event
	None MetricEvent = iota
	// Created created measurement event
	Created
	// Updated updated measurement event
	Updated
	// Deleted deleted measurement event
	Deleted

)

func (e MetricEvent) String() string {
	return [...]string{"None", "Created", "Updated", "Deleted"}[e]
}
