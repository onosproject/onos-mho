// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: LicenseRef-ONF-Member-1.0

package monitoring

import (
	"context"
	"github.com/onosproject/onos-mho/pkg/rnib"

	"github.com/onosproject/onos-mho/pkg/store/metrics"

	e2api "github.com/onosproject/onos-api/go/onos/e2t/e2/v1beta1"

	topoapi "github.com/onosproject/onos-api/go/onos/topo"

	appConfig "github.com/onosproject/onos-mho/pkg/config"

	"github.com/onosproject/onos-mho/pkg/broker"
)

//var log = logging.GetLogger("monitoring")

// NewMonitor creates a new indication monitor
func NewMonitor(opts ...Option) *Monitor {
	options := Options{}

	for _, opt := range opts {
		opt.apply(&options)
	}
	return &Monitor{
		streamReader: options.Monitor.StreamReader,
		appConfig:    options.App.AppConfig,
		metricStore:  options.App.MetricStore,
		nodeID:       options.Monitor.NodeID,
		rnibClient:   options.App.RNIBClient,
	}
}

// Monitor indication monitor
type Monitor struct {
	streamReader broker.StreamReader
	appConfig    *appConfig.AppConfig
	metricStore  metrics.Store
	nodeID       topoapi.ID
	rnibClient   rnib.Client
}

func (m *Monitor) processIndication(ctx context.Context, indication e2api.Indication, nodeID topoapi.ID) error {
	//err := m.processIndicationFormat1(ctx, indication, nodeID)
	//if err != nil {
	//	log.Warn(err)
	//	return err
	//}

	return nil
}

// Start start monitoring of indication messages for a given subscription ID
func (m *Monitor) Start(ctx context.Context) error {
	errCh := make(chan error)
	go func() {
		for {
			indMsg, err := m.streamReader.Recv(ctx)
			if err != nil {
				errCh <- err
			}
			err = m.processIndication(ctx, indMsg, m.nodeID)
			if err != nil {
				errCh <- err
			}
		}
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}
