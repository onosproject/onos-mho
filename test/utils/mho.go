// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: LicenseRef-ONF-Member-1.0

package utils

import (
	"context"
	"testing"

	"github.com/onosproject/onos-api/go/onos/mho"
	"github.com/onosproject/onos-ric-sdk-go/pkg/utils/creds"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)


// MHOServiceAddress defines the address and port for connections to the MHO service
const MHOServiceAddress = "onos-mho:5150"

// ConnectMHOServiceHost connects to the onos MHO service
func ConnectMHOServiceHost() (*grpc.ClientConn, error) {
	tlsConfig, err := creds.GetClientCredentials()
	if err != nil {
		return nil, err
	}
	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig)),
	}

	return grpc.DialContext(context.Background(), MHOServiceAddress, opts...)
}

// GetMHOClient returns an SDK subscription client
func GetMhoClient(t *testing.T) mho.MhoClient {
	conn, err := ConnectMHOServiceHost()
	assert.NoError(t, err)
	assert.NotNil(t, conn)

	return mho.NewMhoClient(conn)
}
