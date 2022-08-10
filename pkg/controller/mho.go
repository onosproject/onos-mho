// SPDX-FileCopyrightText: 2022-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"context"
	"github.com/onosproject/onos-lib-go/pkg/logging"
	"github.com/onosproject/onos-mho/pkg/store"
)

var log = logging.GetLogger()

func NewMHOController(metricStore store.Store) *MHOController {
	return &MHOController{
		metricStore: metricStore,
	}
}

type MHOController struct {
	metricStore store.Store
}

func (m *MHOController) Run(ctx context.Context) {
	log.Info("MHO controller started")
	go m.controlHandover(ctx)
}

func (m *MHOController) controlHandover(ctx context.Context) {
	ch := make(chan store.Event)
	err := m.metricStore.Watch(ctx, ch)
	if err != nil {
		log.Error(err)
	}

	for e := range ch {
		if e.Type == store.Created || e.Type == store.Updated {
			key := e.Key.(string)
			v, err := m.metricStore.Get(ctx, key)
			if err != nil {
				log.Error(err)
			}
			nv := v.Value.(*store.MetricValue)
			log.Debugf("new event indication message - key: %v, value: %v, event type: %v", e.Key, nv, e.Type)
			if e.EventMHOState.(store.MHOState) == store.StateCreated || e.EventMHOState.(store.MHOState) == store.Denied {
				// TODO add advanced MHO logic here
				// uncomment below if Waiting state is necessary
				//log.Infof("State changed for %v from %v to %v", key, nv.State.String(), store.Waiting)
				//nv.State = store.Waiting
				//_, err = m.metricStore.Put(ctx, key, nv)
				//if err != nil {
				//	log.Error(err)
				//}

				log.Debugf("State changed for %v from %v to %v", key, nv.State.String(), store.Approved)
				nv.State = store.Approved
				_, err = m.metricStore.Put(ctx, key, nv, store.Approved)
				if err != nil {
					log.Error(err)
				}
			}
		}
	}
}
