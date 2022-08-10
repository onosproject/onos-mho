// SPDX-FileCopyrightText: 2022-present Intel Corporation
// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package store

import (
	topoapi "github.com/onosproject/onos-api/go/onos/topo"
	e2smcommonies "github.com/onosproject/onos-e2-sm/servicemodels/e2sm_rc/v1/e2sm-common-ies"
)

// Entry is store entry
type Entry struct {
	Key   string
	Value interface{}
}

// EventType is a store event
type EventType int

const (
	// None none cell event
	None EventType = iota
	// Created created store event
	Created
	// Updated updated store event
	Updated
	// Deleted deleted store event
	Deleted
)

func (e EventType) String() string {
	return [...]string{"None", "Created", "Updated", "Deleted"}[e]
}

// MetricValue is the value for Metric store
type MetricValue struct {
	RawUEID       *e2smcommonies.Ueid
	TgtCellID     string
	State         MHOState
	CallProcessID []byte
	E2NodeID      topoapi.ID
}

// MHOState is HO approval
type MHOState int

const (
	// StateCreated is an MHO created state
	StateCreated MHOState = iota
	// Waiting is an MHO waiting state
	Waiting
	// Approved is an MHO approved state
	Approved
	// Denied is an MHO denied state
	Denied
	// Done is an MHO done state
	Done
)

func (h MHOState) String() string {
	return [...]string{"StateCreated", "Waiting", "Approved", "Denied", "Done"}[h]
}

// MHO metric entry state machine
//  (a)   +---------------+                    (f)
// ------>| State Created |<------------------------------------------+
//        +---------------+                                           |
//                 |   ^                                              |
//             (b) |   |       (f)                                    |
//                 |   +-----------------+                            |
//                 V                     |                            |
//             +---------+   (g)    +--------+                        |
//             | Waiting |--------->| Denied |                        |
//             +---------+          +--------+                        |
//                |   ^               |  |                     (e)    |
//                |   | (h)           |  | (i)                +---+   |
//            (c) |   +---------------+  |                    |   |   |
//                |                      V                    V   |   |
//                |               +----------+              +------+  |
//                +-------------->| Approved |------------->| Done |--+
//                                +----------+     (d)      +------+
// (a) an indication message for a new UE arrives
// (b) MHO controller starts making handover decision
// (c) MHO approves handover
// (d) MHO control message is successfully sent
// (e) MHO control message is successfully sent but indication message for this UE arrives with the same target cell ID
// (f) indication message for this UE arrives with the target cell ID changed
// (g) MHO denies handover
// (h) indication message for this UE arrives with the same target cell ID
