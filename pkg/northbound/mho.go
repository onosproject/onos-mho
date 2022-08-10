// SPDX-FileCopyrightText: 2022-present Intel Corporation
// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package northbound

import (
	"context"

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
	//TODO implement me
	log.Error("Not implemented yet")
	return nil, nil
}

func (s *Server) GetCells(ctx context.Context, request *mhoapi.GetRequest) (*mhoapi.CellList, error) {
	//TODO implement me
	log.Error("Not implemented yet")
	return nil, nil
}

func (s *Server) GetMhoParams(ctx context.Context, request *mhoapi.GetMhoParamRequest) (*mhoapi.GetMhoParamResponse, error) {
	return nil, nil
}

func (s *Server) SetMhoParams(ctx context.Context, request *mhoapi.SetMhoParamRequest) (*mhoapi.SetMhoParamResponse, error) {
	return nil, nil
}
