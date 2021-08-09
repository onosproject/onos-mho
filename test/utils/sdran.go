// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: LicenseRef-ONF-Member-1.0

package utils

import (
	"context"
	"fmt"
	"github.com/onosproject/helmit/pkg/helm"
	"github.com/onosproject/helmit/pkg/input"
	"github.com/onosproject/helmit/pkg/kubernetes"
	"github.com/onosproject/onos-mho/pkg/manager"
	"github.com/onosproject/onos-mho/pkg/store"
	"github.com/onosproject/onos-test/pkg/onostest"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func getCredentials() (string, string, error) {
	kubClient, err := kubernetes.New()
	if err != nil {
		return "", "", err
	}
	secrets, err := kubClient.CoreV1().Secrets().Get(context.Background(), onostest.SecretsName)
	if err != nil {
		return "", "", err
	}
	username := string(secrets.Object.Data["sd-ran-username"])
	password := string(secrets.Object.Data["sd-ran-password"])

	return username, password, nil
}

// CreateSdranRelease creates a helm release for an sd-ran instance
func CreateSdranRelease(c *input.Context) (*helm.HelmRelease, error) {
	username, password, err := getCredentials()
	registry := c.GetArg("registry").String("")

	if err != nil {
		return nil, err
	}

	sdran := helm.Chart("sd-ran", onostest.SdranChartRepo).
		Release("sd-ran").
		SetUsername(username).
		SetPassword(password).
		Set("import.onos-config.enabled", false).
		Set("import.onos-topo.enabled", true).
		Set("import.ran-simulator.enabled", true).
		Set("import.onos-pci.enabled", false).
		Set("import.onos-kpimon.enabled", false).
		Set("import.onos-mho.enabled", false).
		Set("onos-uenib.image.tag", "latest").
		Set("global.image.registry", registry)

	return sdran, nil
}

// VerifyNumUesInStore...
func VerifyNumUesInStore(ctx context.Context, t *testing.T, mgr *manager.Manager) error {
	store := mgr.GetUeStore()
	numUes := 0
	var err error

	select {
	case <-ctx.Done():
		numUes, err = numUesInStore(t, store)
	case <-time.After(TestInterval):
		numUes, err = numUesInStore(t, store)
	}
	if err != nil {
		return err
	}
	if numUes >= TotalNumUEs {
		return nil
	}
	return fmt.Errorf("%s", "Test failed - the number of UEs does not matched")
}

func numUesInStore(t *testing.T, s store.Store) (int, error) {
	// TODO
	ch := make(chan *store.Entry)
	go func() {
		err := s.Entries(context.Background(), ch)
		if err != nil {
			t.Log(err)
		}
	}()

	numUes := 0
	for e := range ch {
		t.Logf("ueID: %v", e.Key)
		numUes++
	}

	return numUes, nil
}

// CreateRanSimulatorWithNameOrDie creates a simulator and fails the test if the creation returned an error
func CreateRanSimulatorWithNameOrDie(t *testing.T, c *input.Context, simName string) *helm.HelmRelease {
	sim := CreateRanSimulatorWithName(t, c, simName)
	assert.NotNil(t, sim)
	return sim
}

// CreateRanSimulatorWithName creates a ran simulator
func CreateRanSimulatorWithName(t *testing.T, c *input.Context, name string) *helm.HelmRelease {
	username, password, err := getCredentials()
	assert.NoError(t, err)

	registry := c.GetArg("registry").String("")

	simulator := helm.
		Chart("ran-simulator", onostest.SdranChartRepo).
		Release(name).
		SetUsername(username).
		SetPassword(password).
		Set("image.tag", "latest").
		Set("fullnameOverride", "").
		Set("global.image.registry", registry)
	err = simulator.Install(true)
	assert.NoError(t, err, "could not install device simulator %v", err)

	return simulator
}
