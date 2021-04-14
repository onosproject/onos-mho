// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: LicenseRef-ONF-Member-1.0

package mho

import (
	"github.com/onosproject/helmit/pkg/test"
	"github.com/onosproject/onos-mho/pkg/test/utils"
)

// TestSuite is the primary onos-mho test suite
type TestSuite struct {
	test.Suite
}

// SetupTestSuite sets up the onos-mho test suite
func (s *TestSuite) SetupTestSuite() error {
	sdran, err := utils.CreateSdranRelease()
	if err != nil {
		return err
	}
	return sdran.Install(true)
}
