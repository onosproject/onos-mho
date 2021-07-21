// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: LicenseRef-ONF-Member-1.0

package uenib

import (
	"context"
	"github.com/gogo/protobuf/types"
	"github.com/onosproject/onos-api/go/onos/uenib"
	"github.com/onosproject/onos-lib-go/pkg/logging"
	"github.com/onosproject/onos-lib-go/pkg/southbound"
	"github.com/onosproject/onos-mho/pkg/store/event"
	"github.com/onosproject/onos-mho/pkg/controller"
	measurementStore "github.com/onosproject/onos-mho/pkg/store/measurements"
)

const (
	// UENIBAddress has UENIB endpoint
	UENIBAddress = "onos-uenib:5150"
)

var log = logging.GetLogger("uenib")

// NewUENIBClient returns new UENIBClient object
func NewUENIBClient(ctx context.Context, certPath string, keyPath string, store measurementStore.Store) Client {
	conn, err := southbound.Connect(ctx, UENIBAddress, certPath, keyPath)

	if err != nil {
		log.Error(err)
	}
	return &client{
		client:           uenib.NewUEServiceClient(conn),
		measurementStore: store,
	}
}

// Client is an interface for UENIB client
type Client interface {
	// Run runs UENIBClient
	Run(ctx context.Context)

	// UpdateMHOResult updates MHO results to UENIB
	UpdateMHOResult(ctx context.Context, measEntry *measurementStore.Entry)

	// WatchMeasStore watches measurement entries
	WatchMeasStore(ctx context.Context, ch chan event.Event)
}

type client struct {
	client           uenib.UEServiceClient
	measurementStore measurementStore.Store
}

func (c *client) WatchMeasStore(ctx context.Context, ch chan event.Event) {
	for e := range ch {
		measEntry := e.Value.(*measurementStore.Entry)
		c.UpdateMHOResult(ctx, measEntry)
	}
}

func (c *client) Run(ctx context.Context) {
	ch := make(chan event.Event)
	err := c.measurementStore.Watch(ctx, ch)
	if err != nil {
		log.Warn(err)
	}

	go c.WatchMeasStore(ctx, ch)
}

func (c *client) UpdateMHOResult(ctx context.Context, measEntry *measurementStore.Entry) {
	req := c.createUENIBUpdateReq(measEntry)
	log.Debugf("UpdateReq msg: %v", req)
	resp, err := c.client.UpdateUE(ctx, req)
	if err != nil {
		log.Warn(err)
	}

	log.Debugf("resp: %v", resp)
}

func (c *client) createUENIBUpdateReq(measEntry *measurementStore.Entry) *uenib.UpdateUERequest {
	keyID := measEntry.Key.UeID
	ueData := measEntry.Value.(controller.UeData)
	log.Infof("Key ID to be stored in UENIB: %v", keyID)
	log.Infof("Meas Items to be stored in UENIB: %v", ueData)

	uenibObj := uenib.UE{
		ID:      uenib.ID(keyID),
		Aspects: make(map[string]*types.Any),
	}

	uenibObj.Aspects["RRCState"] = &types.Any{
		TypeUrl: "RRCState",
		// TODO
		Value:   []byte(ueData.RrcState),
	}

	return &uenib.UpdateUERequest{
		UE: uenibObj,
	}
}
