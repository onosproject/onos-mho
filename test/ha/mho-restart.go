// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: LicenseRef-ONF-Member-1.0

package ha

import (
	"context"
	"github.com/onosproject/helmit/pkg/helm"
	"github.com/onosproject/helmit/pkg/kubernetes"
	"github.com/onosproject/helmit/pkg/kubernetes/core/v1"
	"github.com/onosproject/onos-api/go/onos/mho"
	"github.com/onosproject/onos-mho/test/utils"
	"github.com/stretchr/testify/assert"
	"strings"
	"testing"
	"time"

	//"time"
)

const (
	onosComponentName = "sd-ran"
)

// GetPodListOrFail gets the list of pods active in the onos-config release. The test is failed if getting the list returns
// an error.
func GetPodListOrFail(t *testing.T) []*v1.Pod {
	release := helm.Chart(onosComponentName).Release(onosComponentName)
	client := kubernetes.NewForReleaseOrDie(release)
	podList, err := client.
		CoreV1().
		Pods().
		List(context.Background())
	assert.NoError(t, err)
	return podList
}

// CrashPodOrFail deletes the given pod and fails the test if there is an error
func CrashPodOrFail(t *testing.T, pod *v1.Pod) {
	err := pod.Delete(context.Background())
	assert.NoError(t, err)
}

// FindPodWithPrefix looks for the first pod whose name matches the given prefix string. The test is failed
// if no matching pod is found.
func FindPodWithPrefix(t *testing.T, prefix string) *v1.Pod {
	podList := GetPodListOrFail(t)
	for _, p := range podList {
		if strings.HasPrefix(p.Name, prefix) {
			return p
		}
	}
	assert.Failf(t, "No pod found matching %s", prefix)
	return nil
}

// GetUesOrFail queries measurement data from onos-mho
func GetUesOrFail(t *testing.T) *mho.UeList {
	var (
		resp *mho.UeList
		err  error
	)
	client := utils.GetMhoClient(t)
	assert.NotNil(t, client)

	req := &mho.GetRequest{}

	maxAttempts := 30
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		resp, err = client.GetUes(context.Background(), req)
		if err == nil && len(resp.Ues) > 0 {
			return resp
		}
		time.Sleep(4 * time.Second)
	}

	return nil
}

// TestMhoRestart tests that onos-mho recovers from crashes
func (s *TestSuite) TestMhoRestart(t *testing.T) {
	sim := utils.CreateRanSimulatorWithNameOrDie(t, s.c, "test-mho-restart")
	assert.NotNil(t, sim)

	// First make sure that MHO came up properly
	resp := GetUesOrFail(t)
	assert.NotNil(t, resp)

	for i := 1; i <= 5; i++ {
		// Crash onos-mho
		e2tPod := FindPodWithPrefix(t, "onos-mho")
		CrashPodOrFail(t, e2tPod)

		//TODO
		resp = GetUesOrFail(t)
		assert.NotNil(t, resp)
	}

	t.Log("MHO restart test passed")
}
