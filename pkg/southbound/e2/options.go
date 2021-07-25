// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: LicenseRef-ONF-Member-1.0

package e2

import (
	e2api "github.com/onosproject/onos-api/go/onos/e2t/e2/v1beta1"
	"github.com/onosproject/onos-mho/pkg/broker"
	appConfig "github.com/onosproject/onos-mho/pkg/config"
	"github.com/onosproject/onos-mho/pkg/controller"
	"github.com/onosproject/onos-mho/pkg/store"
)

// Options E2 client options
type Options struct {
	E2TService E2TServiceOptions

	E2SubService E2SubServiceOptions

	ServiceModel ServiceModelOptions

	App AppOptions
}

// AppOptions application options
type AppOptions struct {
	AppID string

	Config appConfig.Config

	Broker broker.Broker

	IndCh chan *controller.E2NodeIndication

	CtrlReqChs map[string]chan *e2api.ControlMessage

	MeasurementStore store.Store
}

// E2TServiceOptions are the options for a E2T service
type E2TServiceOptions struct {
	// Host is the service host
	Host string
	// Port is the service port
	Port int
}

// E2SubServiceOptions are the options for E2sub service
type E2SubServiceOptions struct {
	// Host is the service host
	Host string
	// Port is the service port
	Port int
}

// ServiceModelName is a service model identifier
type ServiceModelName string

// ServiceModelVersion string
type ServiceModelVersion string

// ServiceModelOptions is options for defining a service model
type ServiceModelOptions struct {
	// Name is the service model identifier
	Name ServiceModelName

	// Version is the service model version
	Version ServiceModelVersion
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

// WithE2TAddress sets the address for the E2T service
func WithE2TAddress(host string, port int) Option {
	return newOption(func(options *Options) {
		options.E2TService.Host = host
		options.E2TService.Port = port
	})
}

// WithE2THost sets the host for the e2t service
func WithE2THost(host string) Option {
	return newOption(func(options *Options) {
		options.E2TService.Host = host
	})
}

// WithE2TPort sets the port for the e2t service
func WithE2TPort(port int) Option {
	return newOption(func(options *Options) {
		options.E2TService.Port = port
	})
}

// WithE2SubAddress sets the address for the E2Sub service
func WithE2SubAddress(host string, port int) Option {
	return newOption(func(options *Options) {
		options.E2SubService.Host = host
		options.E2SubService.Port = port
	})
}

// WithServiceModel sets the client service model
func WithServiceModel(name ServiceModelName, version ServiceModelVersion) Option {
	return newOption(func(options *Options) {
		options.ServiceModel = ServiceModelOptions{
			Name:    name,
			Version: version,
		}
	})
}

// WithAppID sets application ID
func WithAppID(appID string) Option {
	return newOption(func(options *Options) {
		options.App.AppID = appID
	})
}

// WithAppConfig sets the app config interface
func WithAppConfig(appConfig appConfig.Config) Option {
	return newOption(func(options *Options) {
		options.App.Config = appConfig
	})
}

// WithBroker sets subscription broker
func WithBroker(broker broker.Broker) Option {
	return newOption(func(options *Options) {
		options.App.Broker = broker
	})
}

// WithIndChan ...
func WithIndChan(indCh chan *controller.E2NodeIndication) Option {
	return newOption(func(options *Options) {
		options.App.IndCh = indCh
	})
}

// WithCtrlReqChs ...
func WithCtrlReqChs(ctrlReqChs map[string]chan *e2api.ControlMessage) Option {
	return newOption(func(options *Options) {
		options.App.CtrlReqChs = ctrlReqChs
	})
}

// WithMeasurementStore sets measurement store
func WithMeasurementStore(measurementStore store.Store) Option {
	return newOption(func(options *Options) {
		options.App.MeasurementStore = measurementStore
	})
}
