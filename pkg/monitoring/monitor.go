// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: LicenseRef-ONF-Member-1.0

package monitoring

import (
	"context"
	"github.com/onosproject/onos-mho/pkg/controller"
	"github.com/onosproject/onos-mho/pkg/rnib"

	e2api "github.com/onosproject/onos-api/go/onos/e2t/e2/v1beta1"

	topoapi "github.com/onosproject/onos-api/go/onos/topo"

	appConfig "github.com/onosproject/onos-mho/pkg/config"

	"github.com/onosproject/onos-mho/pkg/broker"

	"github.com/onosproject/onos-lib-go/pkg/logging"

	e2sm_mho "github.com/onosproject/onos-e2-sm/servicemodels/e2sm_mho/v1/e2sm-mho"

	e2ind "github.com/onosproject/onos-ric-sdk-go/pkg/e2/indication"

)

var log = logging.GetLogger("monitoring")

// NewMonitor creates a new indication monitor
func NewMonitor(opts ...Option) *Monitor {
	options := Options{}

	for _, opt := range opts {
		opt.apply(&options)
	}
	return &Monitor{
		streamReader: options.Monitor.StreamReader,
		appConfig:    options.App.Config,
		nodeID:       options.Monitor.NodeID,
		rnibClient:   options.App.RNIBClient,
		indChan: options.App.IndCh,
		triggerType: options.App.TriggerType,
	}
}

// Monitor indication monitor
type Monitor struct {
	streamReader broker.StreamReader
	appConfig    appConfig.Config
	nodeID       topoapi.ID
	rnibClient   rnib.Client
	indChan      chan *controller.E2NodeIndication
	triggerType e2sm_mho.MhoTriggerType
}

func (m *Monitor) processIndication(ctx context.Context, indication e2api.Indication, nodeID topoapi.ID) error {
	log.Debugf("processIndication, nodeID: %v, indication: %v ", nodeID, indication)

	m.indChan <- &controller.E2NodeIndication{
		NodeID: string(nodeID),
		TriggerType: m.triggerType,
		IndMsg: e2ind.Indication{
			Payload: e2ind.Payload{
				Header:  indication.Header,
				Message: indication.Payload,
			},
		},
	}

	return nil
}

// Start start monitoring of indication messages for a given subscription ID
func (m *Monitor) Start(ctx context.Context) error {
	errCh := make(chan error)
	go func() {
		for {
			indMsg, err := m.streamReader.Recv(ctx)
			if err != nil {
				log.Errorf("Error reading indication stream, chanID:%v, streamID:%v, err:%v", m.streamReader.ChannelID(), m.streamReader.StreamID(), err)
				errCh <- err
			}
			err = m.processIndication(ctx, indMsg, m.nodeID)
			if err != nil {
				log.Errorf("Error processing indication, err:%v", err)
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
