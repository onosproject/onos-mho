// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package northbound

import (
	"context"

	mhoapi "github.com/onosproject/onos-api/go/onos/mho"
	"github.com/onosproject/onos-lib-go/pkg/logging"
	"github.com/onosproject/onos-lib-go/pkg/logging/service"
	"github.com/onosproject/onos-mho/pkg/mho"
	"github.com/onosproject/onos-mho/pkg/store"
	"google.golang.org/grpc"
)

var log = logging.GetLogger()

// NewService ...
func NewService(ueStore store.Store, cellStore store.Store) service.Service {
	return &Service{
		ueStore:   ueStore,
		cellStore: cellStore,
	}
}

// Service ...
type Service struct {
	service.Service
	ueStore   store.Store
	cellStore store.Store
}

// Register registers the Service with the gRPC server.
func (s Service) Register(r *grpc.Server) {
	server := &Server{
		ueStore:   s.ueStore,
		cellStore: s.cellStore,
	}
	mhoapi.RegisterMhoServer(r, server)
}

// Server implements the MHO gRPC service for administrative facilities.
type Server struct {
	ueStore   store.Store
	cellStore store.Store
}

func (s *Server) GetMhoParams(ctx context.Context, request *mhoapi.GetMhoParamRequest) (*mhoapi.GetMhoParamResponse, error) {
	return nil, nil
}

func (s *Server) SetMhoParams(ctx context.Context, request *mhoapi.SetMhoParamRequest) (*mhoapi.SetMhoParamResponse, error) {
	return nil, nil
}

func (s *Server) GetUes(ctx context.Context, request *mhoapi.GetRequest) (*mhoapi.UeList, error) {
	ch := make(chan *store.Entry)
	go func() {
		err := s.ueStore.Entries(context.Background(), ch)
		if err != nil {
			log.Error(err)
		}
	}()

	ueList := mhoapi.UeList{}

	for e := range ch {
		ueData := e.Value.(mho.UeData)
		ue := mhoapi.UE{
			UeId:     ueData.UeID,
			Cgi:      ueData.CGIString,
			RrcState: ueData.RrcState,
		}
		ueList.Ues = append(ueList.Ues, &ue)
	}
	return &ueList, nil
}

func (s *Server) GetCells(ctx context.Context, request *mhoapi.GetRequest) (*mhoapi.CellList, error) {
	ch := make(chan *store.Entry)
	go func() {
		err := s.cellStore.Entries(context.Background(), ch)
		if err != nil {
			log.Error(err)
		}
	}()

	cellList := mhoapi.CellList{}

	for e := range ch {
		cellData := e.Value.(mho.CellData)
		cell := mhoapi.Cell{
			Cgi:                    cellData.CGIString,
			NumUes:                 int64(len(cellData.Ues)),
			CumulativeHandoversIn:  int64(cellData.CumulativeHandoversIn),
			CumulativeHandoversOut: int64(cellData.CumulativeHandoversOut),
		}
		cellList.Cells = append(cellList.Cells, &cell)
	}
	return &cellList, nil

}
