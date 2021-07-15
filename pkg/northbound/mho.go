// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: LicenseRef-ONF-Member-1.0

package northbound

import (
	service "github.com/onosproject/onos-lib-go/pkg/northbound"
	"google.golang.org/grpc"
)

// NewService returns a new MHO interface service.
func NewService() service.Service {
	return &Service{}
}

// Service is a service implementation for administration.
type Service struct {
	service.Service
}

// Register registers the Service with the gRPC server.
func (s Service) Register(r *grpc.Server) {
	//server := &Server{}
	//mhoapi.RegisterMhoServer(r, server)

}

type Server struct {
}