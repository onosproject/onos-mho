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
	"github.com/onosproject/onos-mho/pkg/controller"
	"github.com/onosproject/onos-mho/pkg/store"
)

const (
	// UENIBAddress has UENIB endpoint
	UENIBAddress = "onos-uenib:5150"
)

var log = logging.GetLogger("uenib")

// NewUENIBClient returns new UENIBClient object
func NewUENIBClient(ctx context.Context, certPath string, keyPath string, store store.Store) Client {
	conn, err := southbound.Connect(ctx, UENIBAddress, certPath, keyPath)

	if err != nil {
		log.Error(err)
	}
	return &client{
		client:           uenib.NewUEServiceClient(conn),
		store: store,
	}
}

// Client is an interface for UENIB client
type Client interface {
	// Run runs UENIBClient
	Run(ctx context.Context)

	// UpdateMhoResult updates MHO results to UENIB
	UpdateMhoResult(ctx context.Context, measEntry *store.Entry)

	// WatchMhoStore watches store entries
	WatchMhoStore(ctx context.Context, ch chan store.Event)
}

type client struct {
	client           uenib.UEServiceClient
	store store.Store
}

func (c *client) WatchMhoStore(ctx context.Context, ch chan store.Event) {
	for e := range ch {
		measEntry := e.Value.(*store.Entry)
		c.UpdateMhoResult(ctx, measEntry)
	}
}

func (c *client) Run(ctx context.Context) {
	ch := make(chan store.Event)
	err := c.store.Watch(ctx, ch)
	if err != nil {
		log.Warn(err)
	}

	go c.WatchMhoStore(ctx, ch)
}

func (c *client) UpdateMhoResult(ctx context.Context, measEntry *store.Entry) {
	req := c.createUENIBUpdateReq(measEntry)
	log.Debugf("UpdateReq msg: %v", req)
	resp, err := c.client.UpdateUE(ctx, req)
	if err != nil {
		log.Warn(err)
	}

	log.Debugf("resp: %v", resp)
}

func (c *client) createUENIBUpdateReq(measEntry *store.Entry) *uenib.UpdateUERequest {
	keyID := measEntry.Key.UeID
	ueData := measEntry.Value.(controller.UeData)
	//log.Debugf("Key ID to be stored in UENIB: %v", keyID)
	//log.Debugf("UeData to be stored in UENIB: %v", ueData)

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
