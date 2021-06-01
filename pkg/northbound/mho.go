// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: LicenseRef-ONF-Member-1.0

package northbound

import (
	service "github.com/onosproject/onos-lib-go/pkg/northbound"
	//mhoapi "github.com/onosproject/onos-api/go/onos/mho"
	"github.com/onosproject/onos-mho/pkg/controller"
	"google.golang.org/grpc"
)

// NewService returns a new MHO interface service.
func NewService(ctrl *controller.MhoCtrl) service.Service {
	return &Service{
		Ctrl: ctrl,
	}
}

// Service is a service implementation for administration.
type Service struct {
	service.Service
	Ctrl *controller.MhoCtrl
}

// Register registers the Service with the gRPC server.
func (s Service) Register(r *grpc.Server) {
	//server := &Server{
	//	Ctrl: s.Ctrl,
	//}
	//mhoapi.RegisterMhoServer(r, server)
}

type Server struct {
	Ctrl *controller.MhoCtrl
}
