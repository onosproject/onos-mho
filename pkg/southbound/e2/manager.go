// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: LicenseRef-ONF-Member-1.0

package e2

import (
	"context"
	"fmt"
	"github.com/onosproject/onos-mho/pkg/mho"
	"google.golang.org/protobuf/proto"
	"strings"

	"github.com/onosproject/onos-mho/pkg/monitoring"

	prototypes "github.com/gogo/protobuf/types"
	"github.com/onosproject/onos-lib-go/pkg/errors"

	e2api "github.com/onosproject/onos-api/go/onos/e2t/e2/v1beta1"

	"github.com/onosproject/onos-mho/pkg/broker"

	appConfig "github.com/onosproject/onos-mho/pkg/config"

	topoapi "github.com/onosproject/onos-api/go/onos/topo"
	"github.com/onosproject/onos-e2-sm/servicemodels/e2sm_mho/pdubuilder"
	e2sm_mho "github.com/onosproject/onos-e2-sm/servicemodels/e2sm_mho/v1/e2sm-mho"
	"github.com/onosproject/onos-lib-go/pkg/logging"
	"github.com/onosproject/onos-mho/pkg/rnib"
	e2client "github.com/onosproject/onos-ric-sdk-go/pkg/e2/v1beta1"
)

var log = logging.GetLogger("e2", "subscription", "manager")

const (
	oid = "1.3.6.1.4.1.53148.1.1.2.101"
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
	indChan      chan *mho.E2NodeIndication
	CtrlReqChs map[string]chan *e2api.ControlMessage

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
		indChan: options.App.IndCh,
		CtrlReqChs: options.App.CtrlReqChs,
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

func (m *Manager) getRanFunction(serviceModelsInfo map[string]*topoapi.ServiceModelInfo) (*topoapi.MHORanFunction, error) {
	for _, sm := range serviceModelsInfo {
		smName := strings.ToLower(sm.Name)
		if smName == string(m.serviceModel.Name) && sm.OID == oid {
			mhoRanFunction := &topoapi.MHORanFunction{}
			for _, ranFunction := range sm.RanFunctions {
				if ranFunction.TypeUrl == ranFunction.GetTypeUrl() {
					err := prototypes.UnmarshalAny(ranFunction, mhoRanFunction)
					if err != nil {
						return nil, err
					}
					return mhoRanFunction, nil
				}
			}
		}
	}
	return nil, errors.New(errors.NotFound, "cannot retrieve ran functions")

}

func (m *Manager) createSubscription(ctx context.Context, e2nodeID topoapi.ID, triggerType e2sm_mho.MhoTriggerType) error {

	eventTriggerData, err := m.createEventTrigger(triggerType)
	if err != nil {
		log.Error(err)
		//log.Warn(err)
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

	actions := m.createSubscriptionActions()

	ch := make(chan e2api.Indication)
	node := m.e2client.Node(e2client.NodeID(e2nodeID))
	subName := fmt.Sprintf("onos-mho-subscription-%s", triggerType)
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
		monitoring.WithIndChan(m.indChan),
		monitoring.WithTriggerType(triggerType))

	err = monitor.Start(ctx)
	if err != nil {
		log.Warn(err)
	}

	return nil

}

func (m *Manager) watchE2Connections(ctx context.Context) error {
	log.Info("Start monitoring E2 connections")
	ch := make(chan topoapi.Event)
	err := m.rnibClient.WatchE2Connections(ctx, ch)
	if err != nil {
		log.Warn(err)
		return err
	}

	// creates a new subscription whenever there is a new E2 node connected and supports MHO service model
	for topoEvent := range ch {
		if topoEvent.Type == topoapi.EventType_ADDED || topoEvent.Type == topoapi.EventType_NONE {
			relation := topoEvent.Object.Obj.(*topoapi.Object_Relation)
			e2NodeID := relation.Relation.TgtEntityID
			m.CtrlReqChs[string(e2NodeID)] = make(chan *e2api.ControlMessage)
			triggers := make (map[e2sm_mho.MhoTriggerType]bool)
			triggers[e2sm_mho.MhoTriggerType_MHO_TRIGGER_TYPE_PERIODIC] = m.appConfig.GetPeriodic()
			triggers[e2sm_mho.MhoTriggerType_MHO_TRIGGER_TYPE_UPON_RCV_MEAS_REPORT] = m.appConfig.GetUponRcvMeas()
			triggers[e2sm_mho.MhoTriggerType_MHO_TRIGGER_TYPE_UPON_CHANGE_RRC_STATUS] = m.appConfig.GetUponChangeRrcStatus()
			for triggerType, enabled := range triggers {
				if enabled {
					go func(triggerType e2sm_mho.MhoTriggerType) {
						log.Infof("tx subscription, e2NodeID:%v, triggerType:%v", e2NodeID, e2sm_mho.MhoTriggerType_name[int32(triggerType)])
						err := m.createSubscription(ctx, e2NodeID, triggerType)
						if err != nil {
							log.Warn(err)
						}
					}(triggerType)
				}
			}
			go m.watchMHOChanges(ctx, e2NodeID)
		} else if topoEvent.Type == topoapi.EventType_REMOVED {
			// TODO - Handle E2 node disconnect
			relation := topoEvent.Object.Obj.(*topoapi.Object_Relation)
			e2NodeID := relation.Relation.TgtEntityID
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

	for ctrlReqMsg := range m.CtrlReqChs[string(e2nodeID)] {
		go func(ctrlReqMsg *e2api.ControlMessage) {
			node := m.e2client.Node(e2client.NodeID(e2nodeID))
			ctrlRespMsg, err := node.Control(ctx, ctrlReqMsg)
			if err != nil {
				//TODO - Ignore, MHO does not implement control response message
				//log.Warnf("Error sending control message - %v", err)
			} else if ctrlRespMsg == nil {
				//TODO - Ignore, MHO does not implement control response message
				log.Debugf("Control response message is nil")
			}
		}(ctrlReqMsg)
	}
}

func (m *Manager) createEventTrigger(triggerType e2sm_mho.MhoTriggerType) ([]byte, error) {
	var reportPeriodMs int32
	reportingPeriod, err := m.appConfig.GetReportingPeriod()
	if err != nil {
		return []byte{}, err
	}
	if triggerType == e2sm_mho.MhoTriggerType_MHO_TRIGGER_TYPE_PERIODIC {
		reportPeriodMs = int32(reportingPeriod)
	} else {
		reportPeriodMs = 0
	}
	e2smRcEventTriggerDefinition, err := pdubuilder.CreateE2SmMhoEventTriggerDefinition(triggerType, reportPeriodMs)
	if err != nil {
		return []byte{}, err
	}

	err = e2smRcEventTriggerDefinition.Validate()
	if err != nil {
		return []byte{}, err
	}

	protoBytes, err := proto.Marshal(e2smRcEventTriggerDefinition)
	if err != nil {
		return []byte{}, err
	}

	return protoBytes, err
}

func (m *Manager) createSubscriptionActions() []e2api.Action {
	actions := make([]e2api.Action, 0)
	action := &e2api.Action{
		ID:   int32(0),
		Type: e2api.ActionType_ACTION_TYPE_REPORT,
		SubsequentAction: &e2api.SubsequentAction{
			Type:       e2api.SubsequentActionType_SUBSEQUENT_ACTION_TYPE_CONTINUE,
			TimeToWait: e2api.TimeToWait_TIME_TO_WAIT_ZERO,
		},
	}
	actions = append(actions, *action)
	return actions

}

// Stop stops the subscription manager
func (m *Manager) Stop() error {
	panic("implement me")
}

var _ Node = &Manager{}
