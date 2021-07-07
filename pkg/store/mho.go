// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: LicenseRef-ONF-Member-1.0

package store

import (
	"github.com/onosproject/onos-ric-sdk-go/pkg/e2/indication"
)

type E2NodeIndication struct {
	NodeID string
	IndMsg indication.Indication
}
