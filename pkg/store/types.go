// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: LicenseRef-ONF-Member-1.0

package store

// Key is the key of monitoring result metric store
type Key struct {
	UeID       string
}

// Entry measurement store entry
type Entry struct {
	Key   Key
	Value interface{}
}

// MeasurementEvent a measurement event
type MeasurementEvent int

const (
	// None none cell event
	None MeasurementEvent = iota
	// Created created measurement event
	Created
	// Updated updated measurement event
	Updated
	// Deleted deleted measurement event
	Deleted
)

func (e MeasurementEvent) String() string {
	return [...]string{"None", "Created", "Updated", "Deleted"}[e]
}
