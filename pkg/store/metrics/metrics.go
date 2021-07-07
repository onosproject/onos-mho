// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: LicenseRef-ONF-Member-1.0

package metrics

import (
	"context"
	"sync"

	e2sm_mho "github.com/onosproject/onos-e2-sm/servicemodels/e2sm_mho/v1/e2sm-mho"

	"github.com/onosproject/onos-lib-go/pkg/logging"

	"github.com/google/uuid"

	"github.com/onosproject/onos-mho/pkg/store/watcher"

	"github.com/onosproject/onos-mho/pkg/store/event"

	"github.com/onosproject/onos-lib-go/pkg/errors"
)

var log = logging.GetLogger("store", "metrics")

// Store kpm metrics store interface
type Store interface {
	Put(ctx context.Context, key Key, value interface{}) (*Entry, error)

	// Get gets a metric store entry based on a given key
	Get(ctx context.Context, key Key) (*Entry, error)

	// Update updates an existing entry in the store
	Update(ctx context.Context, entry *Entry) error

	// Delete deletes an entry based on a given key
	Delete(ctx context.Context, key Key) error

	// Entries list all of the metric store entries
	Entries(ctx context.Context, ch chan<- *Entry) error

	// Watch measurement store changes
	Watch(ctx context.Context, ch chan<- event.Event) error
}

type store struct {
	metrics  map[Key]*Entry
	mu       sync.RWMutex
	watchers *watcher.Watchers
}

// NewStore creates new store
func NewStore() Store {
	watchers := watcher.NewWatchers()
	return &store{
		metrics:  make(map[Key]*Entry),
		watchers: watchers,
	}
}

func (s *store) Entries(ctx context.Context, ch chan<- *Entry) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.metrics) == 0 {
		return errors.New(errors.NotFound, "no measurements entries stored")
	}

	for _, entry := range s.metrics {
		ch <- entry
	}
	close(ch)
	return nil
}

func (s *store) Delete(ctx context.Context, key Key) error {
	// TODO check the key and make sure it is not empty
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.metrics, key)
	return nil

}

func (s *store) Put(ctx context.Context, key Key, value interface{}) (*Entry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	entry := &Entry{
		Key:   key,
		Value: value,
	}
	s.metrics[key] = entry
	s.watchers.Send(event.Event{
		Key:   key,
		Value: entry,
		Type:  Created,
	})
	return entry, nil

}

func (s *store) Get(ctx context.Context, key Key) (*Entry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if v, ok := s.metrics[key]; ok {
		return v, nil
	}
	return nil, errors.New(errors.NotFound, "the measurement entry does not exist")
}

func (s *store) Watch(ctx context.Context, ch chan<- event.Event) error {
	id := uuid.New()
	err := s.watchers.AddWatcher(id, ch)
	if err != nil {
		log.Error(err)
		close(ch)
		return err
	}
	go func() {
		<-ctx.Done()
		err = s.watchers.RemoveWatcher(id)
		if err != nil {
			log.Error(err)
		}
		close(ch)
	}()
	return nil
}

func (s *store) Update(ctx context.Context, entry *Entry) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.metrics[entry.Key]; ok {
		s.metrics[entry.Key] = entry
		s.watchers.Send(event.Event{
			Key:   entry.Key,
			Value: entry,
			Type:  Updated,
		})

		return nil
	}
	return errors.New(errors.NotFound, "the entry does not exist")
}

// NewKey creates a new measurements map key
func NewKey(ueID *e2sm_mho.UeIdentity) Key {
	return Key{
		UeID: ueID,
	}
}

var _ Store = &store{}
