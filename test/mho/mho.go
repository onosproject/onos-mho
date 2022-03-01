// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package mho

import (
	"context"
	"github.com/onosproject/onos-lib-go/pkg/certs"
	"github.com/onosproject/onos-mho/pkg/manager"
	"github.com/onosproject/onos-mho/test/utils"
	"github.com/stretchr/testify/assert"
	"testing"
)

// TestMhoSm is the function for Helmit-based integration test
func (s *TestSuite) TestMhoSm(t *testing.T) {
	sim := utils.CreateRanSimulatorWithNameOrDie(t, s.c, "test-mho-sm")
	assert.NotNil(t, sim)
	cfg := manager.Config{
		CAPath:      "/tmp/tls.cacrt",
		KeyPath:     "/tmp/tls.key",
		CertPath:    "/tmp/tls.crt",
		ConfigPath:  "/tmp/config.json",
		E2tEndpoint: "onos-e2t:5150",
		GRPCPort:    5150,
		SMName:      utils.MhoServiceModelName,
		SMVersion:   utils.MhoServiceModelVersion,
	}

	_, err := certs.HandleCertPaths(cfg.CAPath, cfg.KeyPath, cfg.CertPath, true)
	assert.NoError(t, err)

	mgr := manager.NewManager(cfg)
	mgr.Run()

	ctx, cancel := context.WithTimeout(context.Background(), utils.TestTimeout)
	defer cancel()

	ok := utils.VerifyUes(ctx, t, mgr)
	assert.True(t, ok)

	ueData := utils.GetRandomUe(ctx, t, mgr)
	assert.NotNil(t, ueData)

	ok = utils.VerifyHO(ctx, t, mgr, ueData.UeID)
	assert.True(t, ok)

	assert.NoError(t, sim.Uninstall())
	t.Log("MHO suite test passed")
}
