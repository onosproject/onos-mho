// SPDX-FileCopyrightText: 2022-present Intel Corporation
// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package store

import (
	"context"
	"sync"

	"github.com/onosproject/onos-lib-go/pkg/logging"

	"github.com/google/uuid"

	"github.com/onosproject/onos-lib-go/pkg/errors"
)

var log = logging.GetLogger()

// Store mho store interface
type Store interface {
	Put(ctx context.Context, key string, value interface{}, state MHOState) (*Entry, error)

	// Get gets a store entry based on a given key
	Get(ctx context.Context, key string) (*Entry, error)

	// Delete deletes an entry based on a given key
	Delete(ctx context.Context, key string) error

	// Entries list all of the store entries
	Entries(ctx context.Context, ch chan<- *Entry) error

	// Watch store changes
	Watch(ctx context.Context, ch chan<- Event) error

	// HasEntry checks if store has entry
	HasEntry(ctx context.Context, key string) bool

	NumWatchers() int
}

type store struct {
	records  map[string]*Entry
	mu       sync.RWMutex
	watchers *Watchers
}

// NewStore creates new store
func NewStore() Store {
	watchers := NewWatchers()
	return &store{
		records:  make(map[string]*Entry),
		watchers: watchers,
	}
}

func (s *store) Entries(ctx context.Context, ch chan<- *Entry) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, entry := range s.records {
		ch <- entry
	}

	close(ch)
	return nil
}

func (s *store) Delete(ctx context.Context, key string) error {
	// TODO check the key and make sure it is not empty
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.records, key)
	return nil

}

func (s *store) Put(ctx context.Context, key string, value interface{}, state MHOState) (*Entry, error) {
	log.Debugf("Put store entry: key: %v, value: %v", key, value)
	s.mu.Lock()
	defer s.mu.Unlock()
	entry := &Entry{
		Key:   key,
		Value: value,
	}
	if _, ok := s.records[key]; !ok {
		s.records[key] = entry
		s.watchers.Send(Event{
			Key:           key,
			Value:         entry,
			Type:          Created,
			EventMHOState: state,
		})
		return entry, nil
	}
	s.records[key] = entry
	s.watchers.Send(Event{
		Key:           key,
		Value:         entry,
		Type:          Updated,
		EventMHOState: state,
	})
	return entry, nil
}

func (s *store) Get(ctx context.Context, key string) (*Entry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if v, ok := s.records[key]; ok {
		return v, nil
	}
	return nil, errors.New(errors.NotFound, "the entry does not exist")
}

func (s *store) HasEntry(ctx context.Context, key string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.records[key]; ok {
		return true
	}
	return false
}

func (s *store) Watch(ctx context.Context, ch chan<- Event) error {
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

func (s *store) NumWatchers() int {
	return len(s.watchers.watchers)
}

var _ Store = &store{}
