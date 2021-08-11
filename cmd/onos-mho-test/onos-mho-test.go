// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: LicenseRef-ONF-Member-1.0

package main

import (
	"github.com/onosproject/helmit/pkg/registry"
	"github.com/onosproject/helmit/pkg/test"
	"github.com/onosproject/onos-mho/test/ha"
	"github.com/onosproject/onos-mho/test/mho"
)

func main() {
	registry.RegisterTestSuite("mho", &mho.TestSuite{})
	registry.RegisterTestSuite("ha", &ha.TestSuite{})
	test.Main()
}
