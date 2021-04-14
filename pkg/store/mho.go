// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: LicenseRef-ONF-Member-1.0

package store

import (
	"github.com/onosproject/onos-ric-sdk-go/pkg/e2/indication"
)

// CGI is the ID for each cell
type CGI struct {
	PlmnID  uint32
	Ecid    uint64
	EcidLen uint32
}

type E2NodeIndication struct {
	NodeID string
	IndMsg indication.Indication
}

type UeInfo struct {
	NumConflicts int32
}