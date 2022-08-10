// SPDX-FileCopyrightText: 2022-present Intel Corporation
// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package monitoring

import (
	"context"
	e2smrcies "github.com/onosproject/onos-e2-sm/servicemodels/e2sm_rc/v1/e2sm-rc-ies"
	"github.com/onosproject/onos-lib-go/pkg/errors"
	"github.com/onosproject/onos-mho/pkg/store"
	idutil "github.com/onosproject/onos-mho/pkg/utils/id"
	"google.golang.org/protobuf/proto"

	"github.com/onosproject/onos-mho/pkg/rnib"

	e2api "github.com/onosproject/onos-api/go/onos/e2t/e2/v1beta1"

	topoapi "github.com/onosproject/onos-api/go/onos/topo"

	appConfig "github.com/onosproject/onos-mho/pkg/config"

	"github.com/onosproject/onos-mho/pkg/broker"

	"github.com/onosproject/onos-lib-go/pkg/logging"
)

var log = logging.GetLogger()

// NewMonitor creates a new indication monitor
func NewMonitor(opts ...Option) *Monitor {
	options := Options{}

	for _, opt := range opts {
		opt.apply(&options)
	}
	return &Monitor{
		streamReader: options.Monitor.StreamReader,
		appConfig:    options.App.Config,
		metricStore:  options.App.MetricStore,
		nodeID:       options.Monitor.NodeID,
		rnibClient:   options.App.RNIBClient,
	}
}

// Monitor indication monitor
type Monitor struct {
	streamReader broker.StreamReader
	appConfig    appConfig.Config
	metricStore  store.Store
	nodeID       topoapi.ID
	rnibClient   rnib.Client
}

func (m *Monitor) processInsertIndicationHeader2Message5(ctx context.Context, indication e2api.Indication, nodeID topoapi.ID) error {
	header := e2smrcies.E2SmRcIndicationHeader{}
	err := proto.Unmarshal(indication.Header, &header)
	if err != nil {
		return err
	}

	message := e2smrcies.E2SmRcIndicationMessage{}
	err = proto.Unmarshal(indication.Payload, &message)
	if err != nil {
		return err
	}

	headerFormat2 := header.GetRicIndicationHeaderFormats().GetIndicationHeaderFormat2()
	messageFormat5 := message.GetRicIndicationMessageFormats().GetIndicationMessageFormat5()

	log.Debugf("Indication Header: %v", headerFormat2)
	log.Debugf("Indication Message: %v", messageFormat5)

	callProcessID := indication.GetCallProcessId()

	ueID := headerFormat2.GetUeId()
	var tgtCellID string
	if ueID.GetGNbUeid() != nil {
		key := idutil.GenerateGnbUeIDString(ueID.GetGNbUeid())
		// TODO make it better - implement function to recursively retrieve appropriate RAN parameter
		tgtCellID = messageFormat5.GetRanPRequestedList()[0].GetRanParameterValueType().GetRanPChoiceStructure().GetRanParameterStructure().GetSequenceOfRanParameters()[0].
			GetRanParameterValueType().GetRanPChoiceStructure().GetRanParameterStructure().GetSequenceOfRanParameters()[0].
			GetRanParameterValueType().GetRanPChoiceStructure().GetRanParameterStructure().GetSequenceOfRanParameters()[0].
			GetRanParameterValueType().GetRanPChoiceElementFalse().GetRanParameterValue().GetValuePrintableString()

		//for _, m := range messageFormat5.GetRanPRequestedList() {
		//	if m.GetRanParameterId().Value == definition.NRCGIRANParameterID {
		//		tgtCellID = m.GetRanParameterValueType().GetRanPChoiceElementFalse().GetRanParameterValue().GetValuePrintableString()
		//	} else {
		//		v, err := ranparam.GetRanParameterValue(m.GetRanParameterValueType(), definition.NRCGIRANParameterID)
		//		if err != nil {
		//			return err
		//		}
		//		tgtCellID = v.GetValuePrintableString()
		//	}
		//}
		log.Debugf("Received UEID: %v", *ueID)
		log.Debugf("Received TargetCellID: %v", tgtCellID)
		log.Debugf("Received CallProcessID: %v", callProcessID)
		log.Debugf("Received Store Key: %v", key)

		if m.metricStore.HasEntry(ctx, key) {
			v, err := m.metricStore.Get(ctx, key)
			if err != nil {
				return err
			}
			nv := v.Value.(*store.MetricValue)
			if nv.TgtCellID != tgtCellID {
				log.Debugf("State changed for %v from %v to %v", key, nv.State.String(), store.StateCreated)
				metricValue := &store.MetricValue{
					RawUEID:       ueID,
					TgtCellID:     tgtCellID,
					State:         store.StateCreated,
					CallProcessID: callProcessID,
					E2NodeID:      nodeID,
				}
				_, err := m.metricStore.Put(ctx, key, metricValue, store.StateCreated)
				if err != nil {
					return err
				}
			} else if nv.State == store.Denied {
				// update with the same value to trigger control
				log.Debugf("State changed for %v from % to %v", key, nv.State.String(), store.Denied)
				_, err := m.metricStore.Put(ctx, key, nv, store.Denied)
				if err != nil {
					return err
				}
			} else {
				log.Debugf("Current state for %v is %v", key, nv.State.String())
			}
		} else {
			log.Debugf("State created for %v", key)
			metricValue := &store.MetricValue{
				RawUEID:       ueID,
				TgtCellID:     tgtCellID,
				State:         store.StateCreated,
				CallProcessID: callProcessID,
				E2NodeID:      nodeID,
			}
			_, err := m.metricStore.Put(ctx, key, metricValue, store.StateCreated)
			if err != nil {
				return err
			}
		}
	} else {
		return errors.NewNotSupported("supported type GnbUeid only; received %v", ueID)
	}
	return nil
}

func (m *Monitor) processIndication(ctx context.Context, indication e2api.Indication, nodeID topoapi.ID) error {
	log.Debugf("processIndication, nodeID: %v, indication: %v ", nodeID, indication)
	return m.processInsertIndicationHeader2Message5(ctx, indication, nodeID)
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
