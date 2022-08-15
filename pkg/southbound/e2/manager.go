// SPDX-FileCopyrightText: 2022-present Intel Corporation
// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package e2

import (
	"context"
	"github.com/onosproject/onos-mho/pkg/store"
	"github.com/onosproject/onos-mho/pkg/utils/control"
	"github.com/onosproject/onos-mho/pkg/utils/subscription"
	"strings"

	"github.com/onosproject/onos-mho/pkg/monitoring"

	prototypes "github.com/gogo/protobuf/types"
	"github.com/onosproject/onos-lib-go/pkg/errors"

	e2api "github.com/onosproject/onos-api/go/onos/e2t/e2/v1beta1"

	"github.com/onosproject/onos-mho/pkg/broker"

	appConfig "github.com/onosproject/onos-mho/pkg/config"

	topoapi "github.com/onosproject/onos-api/go/onos/topo"
	"github.com/onosproject/onos-lib-go/pkg/logging"
	"github.com/onosproject/onos-mho/pkg/rnib"
	e2client "github.com/onosproject/onos-ric-sdk-go/pkg/e2/v1beta1"
)

var log = logging.GetLogger()

const (
	oid = "1.3.6.1.4.1.53148.1.1.2.3"
)

// Node e2 manager interface
type Node interface {
	Start() error
	Stop() error
}

// Manager subscription manager
type Manager struct {
	e2client     e2client.Client
	rnibClient   rnib.Client
	serviceModel ServiceModelOptions
	appConfig    appConfig.Config
	streams      broker.Broker
	metricStore  store.Store
}

// NewManager creates a new subscription manager
func NewManager(opts ...Option) (Manager, error) {
	log.Info("Init E2Manager")
	options := Options{}

	for _, opt := range opts {
		opt.apply(&options)
	}

	serviceModelName := e2client.ServiceModelName(options.ServiceModel.Name)
	serviceModelVersion := e2client.ServiceModelVersion(options.ServiceModel.Version)
	appID := e2client.AppID(options.App.AppID)
	e2Client := e2client.NewClient(
		e2client.WithServiceModel(serviceModelName, serviceModelVersion),
		e2client.WithAppID(appID),
		e2client.WithE2TAddress(options.E2TService.Host, options.E2TService.Port))

	rnibClient, err := rnib.NewClient()
	if err != nil {
		return Manager{}, err
	}

	return Manager{
		e2client:   e2Client,
		rnibClient: rnibClient,
		serviceModel: ServiceModelOptions{
			Name:    options.ServiceModel.Name,
			Version: options.ServiceModel.Version,
		},
		appConfig:   options.App.Config,
		streams:     options.App.Broker,
		metricStore: options.App.MetricStore,
	}, nil

}

// Start starts subscription manager
func (m *Manager) Start() error {
	log.Info("Start E2Manager")
	go func() {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		err := m.watchE2Connections(ctx)
		if err != nil {
			return
		}
	}()

	return nil
}

func (m *Manager) sendIndicationOnStream(streamID broker.StreamID, ch chan e2api.Indication) {
	streamWriter, err := m.streams.GetWriter(streamID)
	if err != nil {
		log.Error(err)
		return
	}

	for msg := range ch {
		err := streamWriter.Send(msg)
		if err != nil {
			log.Warn(err)
			return
		}
	}
}

func (m *Manager) getRanFunction(serviceModelsInfo map[string]*topoapi.ServiceModelInfo) (*topoapi.RCRanFunction, error) {
	for _, sm := range serviceModelsInfo {
		smName := strings.ToLower(sm.Name)
		if smName == string(m.serviceModel.Name) && sm.OID == oid {
			rcRanFunction := &topoapi.RCRanFunction{}
			for _, ranFunction := range sm.RanFunctions {
				if ranFunction.TypeUrl == ranFunction.GetTypeUrl() {
					err := prototypes.UnmarshalAny(ranFunction, rcRanFunction)
					if err != nil {
						return nil, err
					}
					return rcRanFunction, nil
				}
			}
		}
	}
	return nil, errors.New(errors.NotFound, "cannot retrieve ran functions")

}

func (m *Manager) createSubscription(ctx context.Context, e2nodeID topoapi.ID) error {
	log.Infof("Creating subscription for E2 node with ID: %v", e2nodeID)

	eventTriggerData, err := subscription.CreateEventTriggerDefinition()
	if err != nil {
		log.Error(err)
		return err
	}

	aspects, err := m.rnibClient.GetE2NodeAspects(ctx, e2nodeID)
	if err != nil {
		log.Warn(err)
		return err
	}

	_, err = m.getRanFunction(aspects.ServiceModels)
	if err != nil {
		log.Warn(err)
		return err
	}

	actions, err := subscription.CreateSubscriptionAction()
	if err != nil {
		log.Warn(err)
	}

	ch := make(chan e2api.Indication)
	node := m.e2client.Node(e2client.NodeID(e2nodeID))
	subName := "onos-mho-subscription"
	subSpec := e2api.SubscriptionSpec{
		Actions: actions,
		EventTrigger: e2api.EventTrigger{
			Payload: eventTriggerData,
		},
	}
	channelID, err := node.Subscribe(ctx, subName, subSpec, ch)
	if err != nil {
		log.Warn(err)
		return err
	}
	log.Debugf("Channel ID: %s", channelID)
	streamReader, err := m.streams.OpenReader(ctx, node, subName, channelID, subSpec)
	if err != nil {
		return err
	}

	go m.sendIndicationOnStream(streamReader.StreamID(), ch)
	monitor := monitoring.NewMonitor(monitoring.WithAppConfig(m.appConfig),
		monitoring.WithNode(node),
		monitoring.WithStreamReader(streamReader),
		monitoring.WithNodeID(e2nodeID),
		monitoring.WithRNIBClient(m.rnibClient),
		monitoring.WithMetricStore(m.metricStore))

	err = monitor.Start(ctx)
	if err != nil {
		log.Warn(err)
	}

	return nil

}

func (m *Manager) newSubscription(ctx context.Context, e2NodeID topoapi.ID) error {
	err := m.createSubscription(ctx, e2NodeID)
	return err
}

func (m *Manager) watchE2Connections(ctx context.Context) error {
	log.Info("Start monitoring E2 connections")
	ch := make(chan topoapi.Event)
	err := m.rnibClient.WatchE2Connections(ctx, ch)
	if err != nil {
		log.Warn(err)
		return err
	}

	for topoEvent := range ch {
		if topoEvent.Type == topoapi.EventType_ADDED || topoEvent.Type == topoapi.EventType_NONE {
			log.Infof("New E2 connection detected")
			relation := topoEvent.Object.Obj.(*topoapi.Object_Relation)
			e2NodeID := relation.Relation.TgtEntityID

			if !m.rnibClient.HasRcRANFunction(ctx, e2NodeID, oid) {
				log.Debugf("Received topo event does not have RC RAN function for MHO - %v", topoEvent)
				continue
			}

			go func() {
				log.Debugf("start creating subscription %v", topoEvent)
				err := m.newSubscription(ctx, e2NodeID)
				if err != nil {
					log.Warn(err)
				}
			}()

			go m.watchMHOChanges(ctx, e2NodeID)
		} else if topoEvent.Type == topoapi.EventType_REMOVED {
			// TODO - Handle E2 node disconnect
			relation := topoEvent.Object.Obj.(*topoapi.Object_Relation)
			e2NodeID := relation.Relation.TgtEntityID
			if !m.rnibClient.HasRcRANFunction(ctx, e2NodeID, oid) {
				log.Debugf("Received topo event does not have RC RAN function for MHO - %v", topoEvent)
				continue
			}
			cellIDs, err := m.rnibClient.GetCells(ctx, e2NodeID)
			if err != nil {
				return err
			}
			for _, cellID := range cellIDs {
				log.Debugf("cell removed, e2NodeID:%v, cellID:%v", e2NodeID, cellID.CellGlobalID.GetValue())
			}
		}
	}
	return nil
}

func (m *Manager) watchMHOChanges(ctx context.Context, e2nodeID topoapi.ID) {
	ch := make(chan store.Event)
	err := m.metricStore.Watch(ctx, ch)
	if err != nil {
		log.Error(err)
		return
	}

	for e := range ch {
		if e.Type == store.Updated {
			key := e.Key.(string)
			v, err := m.metricStore.Get(ctx, key)
			if err != nil {
				log.Error(err)
			}
			nv := v.Value.(*store.MetricValue)
			if e.EventMHOState.(store.MHOState) == store.Approved && nv.E2NodeID == e2nodeID {
				rawUEID := nv.RawUEID
				tgtCellID := nv.TgtCellID
				header, err := control.CreateRcControlHeader(rawUEID)
				if err != nil {
					log.Error(err)
				}
				log.Debugf("send control message for key: %v, value: %v", key, nv)
				payload, err := control.CreateRcControlMessage(tgtCellID)
				if err != nil {
					log.Error(err)
				}

				node := m.e2client.Node(e2client.NodeID(e2nodeID))
				outcome, err := node.Control(ctx, &e2api.ControlMessage{
					Header:  header,
					Payload: payload,
				}, nv.CallProcessID)

				if err != nil {
					log.Warn(err)
				}

				log.Debugf("Outcome: %v", outcome)

				log.Debugf("State changed for %v from %v to %v", key, nv.State.String(), store.Done)
				nv.State = store.Done
				_, err = m.metricStore.Put(ctx, key, nv, store.Done)
				if err != nil {
					log.Error(err)
				}
			}
		}
	}
}

// Stop stops the subscription manager
func (m *Manager) Stop() error {
	panic("implement me")
}

var _ Node = &Manager{}
