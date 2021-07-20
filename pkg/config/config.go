// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: LicenseRef-ONF-Member-1.0

package config

import (
	"context"

	configurable "github.com/onosproject/onos-ric-sdk-go/pkg/config/registry"

	"github.com/onosproject/onos-lib-go/pkg/logging"
	app "github.com/onosproject/onos-ric-sdk-go/pkg/config/app/default"
	"github.com/onosproject/onos-ric-sdk-go/pkg/config/event"
	configutils "github.com/onosproject/onos-ric-sdk-go/pkg/config/utils"
)

var log = logging.GetLogger("config")

const defaultConfigPath = "/etc/onos/config/config.json"

const (
	ReportingPeriodConfigPath = "reportingPeriod"
	PeriodicConfigPath = "periodic"
	UponRcvMeasConfigPath = "uponRcvMeasReport"
	UponChangeRrcStatusConfigPath = "uponChangeRrcStatus"
	A3OffsetRangeConfigPath        = "A3OffsetRange"
	HysteresisRangeConfigPath      = "HysteresisRange"
	CellIndividualOffsetConfigPath = "CellIndividualOffset"
	FrequencyOffsetConfigPath      = "FrequencyOffset"
	TimeToTriggerConfigPath        = "TimeToTrigger"
)

// Config xApp configuration interface
type Config interface {
	GetReportingPeriod() (uint64, error)
	Watch(context.Context, chan event.Event) error
}

// NewConfig initialize the xApp config
func NewConfig() (*AppConfig, error) {
	appConfig, err := configurable.RegisterConfigurable(defaultConfigPath, &configurable.RegisterRequest{})
	if err != nil {
		return nil, err
	}

	cfg := &AppConfig{
		appConfig: appConfig.Config.(*app.Config),
	}
	return cfg, nil
}

// AppConfig application configuration
type AppConfig struct {
	appConfig *app.Config
}

// Watch watch config changes
func (c *AppConfig) Watch(ctx context.Context, ch chan event.Event) error {
	err := c.appConfig.Watch(ctx, ch)
	if err != nil {
		return err
	}
	return nil
}

// GetReportingPeriod gets configured reporting period
func (c *AppConfig) GetReportingPeriod() (uint64, error) {
	interval, err := c.appConfig.Get(ReportingPeriodConfigPath)
	if err != nil {
		log.Error(err)
		return 0, err
	}
	val, err := configutils.ToUint64(interval.Value)
	if err != nil {
		log.Error(err)
		return 0, err
	}

	return val, nil
}

// GetPeriodic returns true if periodic trigger is enabled
func (c *AppConfig) GetPeriodic() bool {
	p, err := c.appConfig.Get(PeriodicConfigPath)
	if err != nil {
		log.Error(err)
		return false
	}
	switch p.Value.(type) {
	case bool:
		return p.Value.(bool)
	default:
		return false
	}
}

// GetUponRcvMeas returns true if periodic trigger is enabled
func (c *AppConfig) GetUponRcvMeas() bool {
	p, err := c.appConfig.Get(UponRcvMeasConfigPath)
	if err != nil {
		log.Error(err)
		return false
	}
	switch p.Value.(type) {
	case bool:
		return p.Value.(bool)
	default:
		return false
	}
}

// GetUponChangeRrcStatus returns true if periodic trigger is enabled
func (c *AppConfig) GetUponChangeRrcStatus() bool {
	p, err := c.appConfig.Get(UponChangeRrcStatusConfigPath)
	if err != nil {
		log.Error(err)
		return false
	}
	switch p.Value.(type) {
	case bool:
		return p.Value.(bool)
	default:
		return false
	}
}

var _ Config = &AppConfig{}
