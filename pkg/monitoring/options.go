// SPDX-FileCopyrightText: 2022-present Intel Corporation
// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package monitoring

import (
	topoapi "github.com/onosproject/onos-api/go/onos/topo"
	"github.com/onosproject/onos-mho/pkg/broker"
	appConfig "github.com/onosproject/onos-mho/pkg/config"
	"github.com/onosproject/onos-mho/pkg/rnib"
	"github.com/onosproject/onos-mho/pkg/store"

	e2client "github.com/onosproject/onos-ric-sdk-go/pkg/e2/v1beta1"
)

// Options monitor options
type Options struct {
	App AppOptions

	Monitor MonitorOptions
}

// AppOptions application options
type AppOptions struct {
	Config appConfig.Config

	RNIBClient rnib.Client

	MetricStore store.Store
}

// MonitorOptions monitoring options
type MonitorOptions struct {
	Node         e2client.Node
	NodeID       topoapi.ID
	StreamReader broker.StreamReader
}

// Option option interface
type Option interface {
	apply(*Options)
}

type funcOption struct {
	f func(*Options)
}

func (f funcOption) apply(options *Options) {
	f.f(options)
}

func newOption(f func(*Options)) Option {
	return funcOption{
		f: f,
	}
}

// WithNodeID sets node ID
func WithNodeID(nodeID topoapi.ID) Option {
	return newOption(func(options *Options) {
		options.Monitor.NodeID = nodeID
	})
}

// WithStreamReader sets stream reader
func WithStreamReader(streamReader broker.StreamReader) Option {
	return newOption(func(options *Options) {
		options.Monitor.StreamReader = streamReader
	})
}

// WithNode sets e2 node interface
func WithNode(node e2client.Node) Option {
	return newOption(func(options *Options) {
		options.Monitor.Node = node
	})
}

// WithAppConfig sets app config
func WithAppConfig(config appConfig.Config) Option {
	return newOption(func(options *Options) {
		options.App.Config = config
	})
}

// WithRNIBClient sets RNIB client
func WithRNIBClient(rnibClient rnib.Client) Option {
	return newOption(func(options *Options) {
		options.App.RNIBClient = rnibClient
	})
}

// WithMetricStore sets metric store
func WithMetricStore(metricStore store.Store) Option {
	return newOption(func(options *Options) {
		options.App.MetricStore = metricStore
	})
}
