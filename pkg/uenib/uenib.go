// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: LicenseRef-ONF-Member-1.0

package uenib

import (
	"context"
	"crypto/tls"
	"github.com/onosproject/onos-api/go/onos/uenib"
	"github.com/onosproject/onos-lib-go/pkg/certs"
	"github.com/onosproject/onos-lib-go/pkg/logging"
	"github.com/onosproject/onos-lib-go/pkg/southbound"
	"github.com/onosproject/onos-mho/pkg/rnib"
	"github.com/onosproject/onos-mho/pkg/store/event"
	"github.com/onosproject/onos-mho/pkg/store/metrics"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"time"
)

const (
	// UENIBAddress has UENIB endpoint
	UENIBAddress = "onos-uenib:5150"
)

var log = logging.GetLogger("uenib")

func NewUENIBDialOpt(certPath string, keyPath string) ([]grpc.DialOption, error) {
	dialOpts := []grpc.DialOption{
		grpc.WithStreamInterceptor(southbound.RetryingStreamClientInterceptor(100 * time.Millisecond)),
		grpc.WithUnaryInterceptor(southbound.RetryingUnaryClientInterceptor()),
	}
	if certPath != "" && keyPath != "" {
		cert, err := tls.LoadX509KeyPair(certPath, keyPath)
		if err != nil {
			return nil, err
		}
		dialOpts = append(dialOpts, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
			Certificates:       []tls.Certificate{cert},
			InsecureSkipVerify: true,
		})))
	} else {
		// Load default Certificates
		cert, err := tls.X509KeyPair([]byte(certs.DefaultClientCrt), []byte(certs.DefaultClientKey))
		if err != nil {
			log.Errorf("failed to make tls key pair: %v", err)
			return nil, err
		}
		dialOpts = append(dialOpts, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
			Certificates:       []tls.Certificate{cert},
			InsecureSkipVerify: true,
		})))
	}

	return dialOpts, nil
}

func NewUENIBClient(ctx context.Context, store metrics.Store, certPath string, keyPath string) Client {
	dialOpts, err := NewUENIBDialOpt(certPath, keyPath)
	if err != nil {
		log.Error(err)
	}
	conn, err := grpc.Dial(UENIBAddress, dialOpts...)
	if err != nil {
		log.Error(err)
	}
	rnibClient, err := rnib.NewClient()
	if err != nil {
		log.Error(err)
	}
	return Client{
		uenibClient:  uenib.NewUEServiceClient(conn),
		rnibClient:   rnibClient,
		metricsStore: store,
	}
}

type Client struct {
	uenibClient  uenib.UEServiceClient
	rnibClient   rnib.Client
	metricsStore metrics.Store
}

func (c *Client) Run(ctx context.Context) {
	go c.watchMetricStore(ctx)
}

func (c *Client) watchMetricStore(ctx context.Context) {
	ch := make(chan event.Event)
	err := c.metricsStore.Watch(ctx, ch)
	if err != nil {
		log.Error(err)
	}
	for e := range ch {
		// new indication message arrives
		log.Debugf("new indication message: %v", e)
		//if e.Type == metrics.Created {
		//	err := c.storeNeighborCellList(ctx, *e.Value.(*metrics.Entry))
		//	if err != nil {
		//		log.Errorf("Error happened when storing neighbors to UENIB: %v", err)
		//	}
		//}
	}
}
