// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: LicenseRef-ONF-Member-1.0

package decode

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPlmnIdToUint32(t *testing.T) {
	samplePlmnIDBytes := []byte{38, 132, 19}
	samplePlmnIDUint32 := PlmnIdToUint32(samplePlmnIDBytes)

	assert.Equal(t, samplePlmnIDUint32, uint32(1279014))
	assert.Equal(t, fmt.Sprintf("%x", samplePlmnIDUint32), "138426")
}
