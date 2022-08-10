// SPDX-FileCopyrightText: 2022-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package id

import (
	"fmt"
	e2smcommonies "github.com/onosproject/onos-e2-sm/servicemodels/e2sm_rc/v1/e2sm-common-ies"
)

func GenerateGnbUeIDString(ueid *e2smcommonies.UeidGnb) string {
	plmnID := bytesToUint64(ueid.GetGuami().GetPLmnidentity().GetValue())
	amfRegionID := bytesToUint64(ueid.GetGuami().GetAMfregionId().GetValue().GetValue())
	amfSetID := bytesToUint64(ueid.GetGuami().GetAMfsetId().GetValue().GetValue())
	amfPointer := bytesToUint64(ueid.GetGuami().GetAMfpointer().GetValue().GetValue())

	amfUeNgapID := ueid.GetAmfUeNgapId().GetValue()
	return fmt.Sprintf("plmnID:%x/amfRegionID:%x/amfSetID:%x/amfPointer:%x/amfUeNgapID:%x",
		plmnID, amfRegionID, amfSetID, amfPointer, amfUeNgapID)
}

func bytesToUint64(b []byte) uint64 {
	result := uint64(0)
	for _, e := range b {
		result = (result << 8) + uint64(e)
	}
	return result
}
