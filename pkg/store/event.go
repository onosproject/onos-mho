// SPDX-FileCopyrightText: 2022-present Intel Corporation
// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package store

// Event store event data structure
type Event struct {
	Key           interface{}
	Value         interface{}
	Type          interface{}
	EventMHOState interface{}
}
