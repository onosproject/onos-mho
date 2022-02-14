// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

// Copied over onos-e2t/test/utils/bitstring.go

package mho

import (
	"fmt"
	e2sm_v2_ies "github.com/onosproject/onos-e2-sm/servicemodels/e2sm_mho_go/v2/e2sm-v2-ies"
)

// Uint64ToBitString converts uint64 to a bit string byte array
func Uint64ToBitString(value uint64, bitCount int) []byte {
	result := make([]byte, bitCount/8+1)
	if bitCount%8 > 0 {
		value = value << (8 - bitCount%8)
	}

	for i := 0; i <= (bitCount / 8); i++ {
		result[i] = byte(value >> (((bitCount / 8) - i) * 8) & 0xFF)
	}

	return result
}

// BitStringToUint64 converts bit string to uint 64
func BitStringToUint64(bitString []byte, bitCount int) uint64 {
	var result uint64
	for i, b := range bitString {
		result += uint64(b) << ((len(bitString) - i - 1) * 8)
	}
	if bitCount%8 != 0 {
		return result >> (8 - bitCount%8)
	}
	return result
}

// ToDo - assuming that UeID is represented with an integer. Change it once treatment of UeID in SD-RAN is defined
func GetUeID(ueID *e2sm_v2_ies.Ueid) (int64, error) {

	switch ue := ueID.Ueid.(type) {
	case *e2sm_v2_ies.Ueid_GNbUeid:
		return ue.GNbUeid.GetAmfUeNgapId().GetValue(), nil
	case *e2sm_v2_ies.Ueid_ENbUeid:
		return ue.ENbUeid.GetMMeUeS1ApId().GetValue(), nil
	case *e2sm_v2_ies.Ueid_EnGNbUeid:
		return int64(ue.EnGNbUeid.GetMENbUeX2ApId().GetValue()), nil
	case *e2sm_v2_ies.Ueid_NgENbUeid:
		return ue.NgENbUeid.GetAmfUeNgapId().GetValue(), nil
	default:
		return -1, fmt.Errorf("GetUeID() couldn't extract UeID - obtained unexpected type %v", ue)
	}
}
