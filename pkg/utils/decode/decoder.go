// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: LicenseRef-ONF-Member-1.0

package decode

import (
	"fmt"
	"github.com/onosproject/onos-api/go/onos/ransim/types"
	"github.com/onosproject/onos-mho/pkg/store"
)

//DecodePlmnIdToUint32 decodes PLMN ID from byte array to uint32
func PlmnIdToUint32(plmnBytes []byte) uint32 {
	return uint32(plmnBytes[0]) | uint32(plmnBytes[1])<<8 | uint32(plmnBytes[2])<<16
}

// CgiToString concatenates PLMN ID and ECI as a string
func CgiToString(cgi *store.CGI) string {
	if cgi != nil {
		return fmt.Sprintf("%d", types.ToECGI(types.PlmnID(cgi.PlmnID), types.ECI(cgi.Ecid)))
	}
	return ""
}
