// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: LicenseRef-ONF-Member-Only-1.0

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

const (
	ReportingPeriodConfigPath      = "reportingPeriod"
	PeriodicConfigPath             = "periodic"
	UponRcvMeasConfigPath          = "uponRcvMeasReport"
	UponChangeRrcStatusConfigPath  = "uponChangeRrcStatus"
	A3OffsetRangeConfigPath        = "A3OffsetRange"
	HysteresisRangeConfigPath      = "HysteresisRange"
	CellIndividualOffsetConfigPath = "CellIndividualOffset"
	FrequencyOffsetConfigPath      = "FrequencyOffset"
	TimeToTriggerConfigPath        = "TimeToTrigger"
)

// Config xApp configuration interface
type Config interface {
	GetReportingPeriod() (uint64, error)
	GetPeriodic() bool
	GetUponRcvMeas() bool
	GetUponChangeRrcStatus() bool
	GetA3OffsetRange() uint64
	GetHysteresisRange() uint64
	GetCellIndividualOffset() uint64
	GetFrequencyOffset() uint64
	GetTimeToTrigger() uint64
	Watch(context.Context, chan event.Event) error
}

// NewConfig initialize the xApp config
func NewConfig(configPath string) (Config, error) {
	appConfig, err := configurable.RegisterConfigurable(configPath, &configurable.RegisterRequest{})
	if err != nil {
		return nil, err
	}

	cfg := &mhoConfig{
		appConfig: appConfig.Config.(*app.Config),
	}
	return cfg, nil
}

// mhoConfig application configuration
type mhoConfig struct {
	appConfig *app.Config
}

// Watch watch config changes
func (c *mhoConfig) Watch(ctx context.Context, ch chan event.Event) error {
	err := c.appConfig.Watch(ctx, ch)
	if err != nil {
		return err
	}
	return nil
}

// GetReportingPeriod gets configured reporting period
func (c *mhoConfig) GetReportingPeriod() (uint64, error) {
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
func (c *mhoConfig) GetPeriodic() bool {
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
func (c *mhoConfig) GetUponRcvMeas() bool {
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
func (c *mhoConfig) GetUponChangeRrcStatus() bool {
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

// GetA3OffsetRange gets configured reporting period
func (c *mhoConfig) GetA3OffsetRange() uint64 {
	x, err := c.appConfig.Get(A3OffsetRangeConfigPath)
	if err != nil {
		log.Error(err)
		return 0
	}
	val, err := configutils.ToUint64(x.Value)
	if err != nil {
		log.Error(err)
		return 0
	}

	return val
}

// GetHysteresisRange gets configured reporting period
func (c *mhoConfig) GetHysteresisRange() uint64 {
	x, err := c.appConfig.Get(HysteresisRangeConfigPath)
	if err != nil {
		log.Error(err)
		return 0
	}
	val, err := configutils.ToUint64(x.Value)
	if err != nil {
		log.Error(err)
		return 0
	}

	return val
}

// GetCellIndividualOffset gets configured reporting period
func (c *mhoConfig) GetCellIndividualOffset() uint64 {
	x, err := c.appConfig.Get(CellIndividualOffsetConfigPath)
	if err != nil {
		log.Error(err)
		return 0
	}
	val, err := configutils.ToUint64(x.Value)
	if err != nil {
		log.Error(err)
		return 0
	}

	return val
}

// GetFrequencyOffset gets configured reporting period
func (c *mhoConfig) GetFrequencyOffset() uint64 {
	x, err := c.appConfig.Get(FrequencyOffsetConfigPath)
	if err != nil {
		log.Error(err)
		return 0
	}
	val, err := configutils.ToUint64(x.Value)
	if err != nil {
		log.Error(err)
		return 0
	}

	return val
}

// GetTimeToTrigger gets configured reporting period
func (c *mhoConfig) GetTimeToTrigger() uint64 {
	x, err := c.appConfig.Get(TimeToTriggerConfigPath)
	if err != nil {
		log.Error(err)
		return 0
	}
	val, err := configutils.ToUint64(x.Value)
	if err != nil {
		log.Error(err)
		return 0
	}

	return val
}

var _ Config = &mhoConfig{}
