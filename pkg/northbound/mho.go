// SPDX-FileCopyrightText: 2022-present Intel Corporation
// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package northbound

import (
	"context"
	"fmt"
	"sync"

	mhoapi "github.com/onosproject/onos-api/go/onos/mho"
	"github.com/onosproject/onos-lib-go/pkg/logging"
	"github.com/onosproject/onos-lib-go/pkg/logging/service"
	"github.com/onosproject/onos-mho/pkg/store"
	"google.golang.org/grpc"
)

var log = logging.GetLogger()

// NewService ...
func NewService(metricStore store.Store) service.Service {
	return &Service{
		metricStore: metricStore,
	}
}

// Service ...
type Service struct {
	service.Service
	metricStore store.Store
}

// Register registers the Service with the gRPC server.
func (s Service) Register(r *grpc.Server) {
	server := &Server{
		metricStore: s.metricStore,
	}
	mhoapi.RegisterMhoServer(r, server)
}

// Server implements the MHO gRPC service for administrative facilities.
type Server struct {
	metricStore store.Store
}

func (s *Server) GetUes(ctx context.Context, request *mhoapi.GetRequest) (*mhoapi.UeList, error) {
	ch := make(chan *store.Entry)
	var wg sync.WaitGroup
	wg.Add(1)
	ues := make([]*mhoapi.UE, 0)
	go func() {
		defer wg.Done()
		for e := range ch {
			v := e.Value.(*store.MetricValue)
			if v.RawUEID.GetGNbUeid() == nil {
				log.Errorf("Currently gNB UE ID supported only: received UE ID: %v", v.RawUEID)
				continue
			}
			tmpUe := &mhoapi.UE{
				UeId:    fmt.Sprintf("%d", v.RawUEID.GetGNbUeid().GetAmfUeNgapId().GetValue()),
				HoState: v.State.String(),
				Cgi:     string(v.TgtCellID),
			}
			ues = append(ues, tmpUe)
		}
	}()
	s.metricStore.Entries(ctx, ch)

	wg.Wait()
	return &mhoapi.UeList{
		Ues: ues,
	}, nil
}

func (s *Server) GetCells(ctx context.Context, request *mhoapi.GetRequest) (*mhoapi.CellList, error) {
	ch := make(chan *store.Entry)
	var wg sync.WaitGroup
	wg.Add(1)
	cells := make([]*mhoapi.Cell, 0)
	result := make(map[string]int, 0)
	go func() {
		defer wg.Done()
		for e := range ch {
			v := e.Value.(*store.MetricValue)
			if v.RawUEID.GetGNbUeid() == nil {
				log.Errorf("Currently gNB UE ID supported only: received UE ID: %v", v.RawUEID)
				continue
			}
			result[v.TgtCellID]++
		}
	}()
	s.metricStore.Entries(ctx, ch)

	wg.Wait()

	for k, v := range result {
		tmpCell := &mhoapi.Cell{
			Cgi:    k,
			NumUes: int64(v),
		}

		cells = append(cells, tmpCell)
	}
	return &mhoapi.CellList{
		Cells: cells,
	}, nil
}
