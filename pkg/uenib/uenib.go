// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package uenib

import (
	"context"
	"fmt"
	"github.com/gogo/protobuf/types"
	"github.com/onosproject/onos-api/go/onos/uenib"
	"github.com/onosproject/onos-lib-go/pkg/logging"
	"github.com/onosproject/onos-lib-go/pkg/southbound"
	"github.com/onosproject/onos-mho/pkg/mho"
	"github.com/onosproject/onos-mho/pkg/store"
	"strconv"
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
		client: uenib.NewUEServiceClient(conn),
		store:  store,
	}
}

// Client is an interface for UENIB client
type Client interface {
	// Run runs UENIBClient
	Run(ctx context.Context)

	// UpdateMhoResult updates MHO results to UENIB
	UpdateMhoResult(ctx context.Context, entry *store.Entry)

	// WatchMhoStore watches store entries
	WatchMhoStore(ctx context.Context, ch chan store.Event)
}

type client struct {
	client uenib.UEServiceClient
	store  store.Store
}

func (c *client) WatchMhoStore(ctx context.Context, ch chan store.Event) {
	for e := range ch {
		entry := e.Value.(*store.Entry)
		go c.UpdateMhoResult(ctx, entry)
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

func (c *client) UpdateMhoResult(ctx context.Context, entry *store.Entry) {
	req := c.createUENIBUpdateReq(entry)
	log.Debugf("UpdateReq msg: %v", req)
	resp, err := c.client.UpdateUE(ctx, req)
	if err != nil {
		log.Warn(err)
	}

	log.Debugf("resp: %v", resp)
}

func (c *client) createUENIBUpdateReq(entry *store.Entry) *uenib.UpdateUERequest {
	ueID := entry.Key
	ueData := entry.Value.(mho.UeData)
	//log.Debugf("Key ID to be stored in UENIB: %v", keyID)
	//log.Debugf("UeData to be stored in UENIB: %v", ueData)

	uenibObj := uenib.UE{
		ID:      uenib.ID(ueID),
		Aspects: make(map[string]*types.Any),
	}

	uenibObj.Aspects["RRCState"] = &types.Any{
		TypeUrl: "RRCState",
		// TODO
		Value: []byte(ueData.RrcState),
	}

	uenibObj.Aspects["CGI"] = &types.Any{
		TypeUrl: "CGI",
		// TODO
		Value: []byte(ueData.CGIString),
	}

	uenibObj.Aspects["RSRP-Serving"] = &types.Any{
		TypeUrl: "RSRP-Serving",
		// TODO
		Value: []byte(strconv.Itoa(int(ueData.RsrpServing))),
	}

	uenibObj.Aspects["5QI"] = &types.Any{
		TypeUrl: "5QI",
		// TODO
		Value: []byte(strconv.Itoa(int(ueData.FiveQI))),
	}

	var str string
	for cgi, rsrp := range ueData.RsrpNeighbors {
		str = str + fmt.Sprintf("%s:%s ", cgi, strconv.Itoa(int(rsrp)))
	}

	uenibObj.Aspects["RSRP-Neighbors"] = &types.Any{
		TypeUrl: "RSRP-Neighbors",
		// TODO
		Value: []byte(str),
	}

	return &uenib.UpdateUERequest{
		UE: uenibObj,
	}
}
