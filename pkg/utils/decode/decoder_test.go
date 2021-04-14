// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: LicenseRef-ONF-Member-1.0

package decode

import (
	"fmt"
	"github.com/onosproject/onos-mho/pkg/store"
	"github.com/stretchr/testify/assert"
	"testing"
)


func TestPlmnIdToUint32(t *testing.T) {
	samplePlmnIDBytes := []byte{38, 132, 19}
	samplePlmnIDUint32 := PlmnIdToUint32(samplePlmnIDBytes)

	assert.Equal(t, samplePlmnIDUint32, uint32(1279014))
	assert.Equal(t, fmt.Sprintf("%x", samplePlmnIDUint32), "138426")
}

func TestCgiToString(t *testing.T) {
	sampleCGI := &store.CGI{
		PlmnID: uint32(1279014),
		Ecid: uint64(82530),
		EcidLen: uint32(28),
	}

	assert.Equal(t, CgiToString(sampleCGI), "343332706402914")
}
