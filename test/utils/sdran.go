// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"context"
	"github.com/onosproject/helmit/pkg/helm"
	"github.com/onosproject/helmit/pkg/input"
	"github.com/onosproject/helmit/pkg/kubernetes"
	"github.com/onosproject/onos-mho/pkg/manager"
	"github.com/onosproject/onos-mho/pkg/mho"
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
		Set("import.onos-mho.enabled", false).
		Set("onos-mho.image.tag", "latest").
		Set("onos-e2t.image.tag", "latest").
		Set("ran-simulator.image.tag", "latest").
		Set("global.image.registry", registry)

	return sdran, nil
}

// VerifyUes ...
func VerifyUes(ctx context.Context, t *testing.T, mgr *manager.Manager) bool {
	store := mgr.GetUeStore()
	numUes := 0

	ticker := time.NewTicker(1 * time.Second)
	tickerCount := 0
	for {
		select {
		case <-ticker.C:
			if tickerCount >= 120 {
				ticker.Stop()
				return false
			}
			numUes = numUesInStore(t, store)
			if numUes == TotalNumUEs {
				ticker.Stop()
				return true
			}
			tickerCount++
		case <-ctx.Done():
			return false
		}
	}
}

// GetRandomUe gets a random UE
func GetRandomUe(ctx context.Context, t *testing.T, mgr *manager.Manager) mho.UeData {
	ueStore := mgr.GetUeStore()
	ch := make(chan *store.Entry)
	go func() {
		err := ueStore.Entries(context.Background(), ch)
		if err != nil {
			t.Log(err)
		}
	}()

	var ueData mho.UeData
	for e := range ch {
		ueID := e.Key
		ueData = e.Value.(mho.UeData)
		t.Logf("ueID: %v, cgi:%v", ueID, ueData.CGIString)
		break
	}

	return ueData
}

// VerifyHO
func VerifyHO(ctx context.Context, t *testing.T, mgr *manager.Manager, ueID string) bool {
	oldCgi := getCgi(ctx, t, mgr, ueID)

	t.Logf("Old serving cell CGI:%v", oldCgi)

	if len(oldCgi) == 0 {
		t.Log("Invalid CGI for old serving cell")
		return false
	}

	ticker := time.NewTicker(1 * time.Second)
	tickerCount := 0
	for {
		select {
		case <-ticker.C:
			if tickerCount >= 60 {
				ticker.Stop()
				return false
			}
			newCgi := getCgi(ctx, t, mgr, ueID)
			if oldCgi != newCgi {
				t.Logf("New serving cell CGI:%v", newCgi)
				return true
			}
			tickerCount++
		case <-ctx.Done():
			ticker.Stop()
			return false
		}
	}
}

func getCgi(ctx context.Context, t *testing.T, mgr *manager.Manager, ueID string) string {
	var cgi string
	ue := GetUe(ctx, t, mgr, ueID)
	if ue != nil {
		cgi = ue.CGIString
	}
	return cgi
}

// GetUe
func GetUe(ctx context.Context, t *testing.T, mgr *manager.Manager, ueID string) *mho.UeData {
	ueStore := mgr.GetUeStore()
	ue, err := ueStore.Get(ctx, ueID)
	if err != nil || ue == nil {
		t.Logf("Failed getting UE %v from store, %v", ueID, err)
		return nil
	}
	u := ue.Value.(mho.UeData)
	ueData := &u
	if ueData.UeID != ueID {
		t.Logf("bad data UE %v", ueID)
		return nil
	}
	return ueData
}

func numUesInStore(t *testing.T, s store.Store) int {
	ch := make(chan *store.Entry)
	go func() {
		err := s.Entries(context.Background(), ch)
		if err != nil {
			t.Log(err)
		}
	}()

	numUes := 0
	for e := range ch {
		ueID := e.Key
		ueData := e.Value.(mho.UeData)
		t.Logf("ueID: %v, cgi:%v", ueID, ueData.CGIString)
		numUes++
	}

	return numUes
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
		Set("pci.modelName", "two-cell-two-node-model").
		Set("global.image.registry", registry)
	err = simulator.Install(true)
	assert.NoError(t, err, "could not install device simulator %v", err)

	return simulator
}
