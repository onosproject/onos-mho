// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: LicenseRef-ONF-Member-1.0

package manager

import (
	e2tapi "github.com/onosproject/onos-api/go/onos/e2t/e2"
	e2sm_mho "github.com/onosproject/onos-e2-sm/servicemodels/e2sm_mho/v1/e2sm-mho"
	"github.com/onosproject/onos-lib-go/pkg/logging"
	"github.com/onosproject/onos-lib-go/pkg/northbound"
	"github.com/onosproject/onos-mho/pkg/controller"
	nbi "github.com/onosproject/onos-mho/pkg/northbound"
	"github.com/onosproject/onos-mho/pkg/southbound/admin"
	"github.com/onosproject/onos-mho/pkg/southbound/ricapie2"
	"github.com/onosproject/onos-mho/pkg/store"
	app "github.com/onosproject/onos-ric-sdk-go/pkg/config/app/default"
	configurable "github.com/onosproject/onos-ric-sdk-go/pkg/config/registry"
	configutils "github.com/onosproject/onos-ric-sdk-go/pkg/config/utils"
)

var log = logging.GetLogger("manager")

// Config is a manager configuration
type Config struct {
	CAPath        string
	KeyPath       string
	CertPath      string
	E2tEndpoint   string
	E2SubEndpoint string
	GRPCPort      int
	AppConfig     *app.Config
	RicActionID   int32
}

// NewManager creates a new manager
func NewManager(config Config) *Manager {
	log.Info("Creating Manager")
	indCh := make(chan *store.E2NodeIndication)
	ctrlReqChs := make(map[string]chan *e2tapi.ControlRequest)
	return &Manager{
		Config: config,
		Sessions: SBSessions{
			AdminSession: admin.NewSession(config.E2tEndpoint),
			E2Session:    ricapie2.NewSession(config.E2tEndpoint, config.E2SubEndpoint, config.RicActionID, 0),
		},
		Chans: Channels{
			IndCh:      indCh,
			CtrlReqChs: ctrlReqChs,
		},
		Ctrls: Controllers{
			MhoCtrl: controller.NewMhoController(indCh, ctrlReqChs),
		},
	}
}

// Manager is a manager for the MHO xAPP service
type Manager struct {
	Config   Config
	Sessions SBSessions
	Chans    Channels
	Ctrls    Controllers
}

// SBSessions is a set of Southbound sessions
type SBSessions struct {
	AdminSession *admin.E2AdminSession
	E2Session    *ricapie2.E2Session
}

// Channels is a set of channels
type Channels struct {
	IndCh      chan *store.E2NodeIndication
	CtrlReqChs map[string]chan *e2tapi.ControlRequest
}

// Controllers is a set of controllers
type Controllers struct {
	MhoCtrl *controller.MhoCtrl
}

// Run starts the manager and the associated services
func (m *Manager) Run() {
	log.Info("Running Manager")
	if err := m.Start(); err != nil {
		log.Fatal("Unable to run Manager", err)
	}
}

// Start starts the manager
func (m *Manager) Start() error {
	// Start Northbound server
	err := m.startNorthboundServer()
	if err != nil {
		return err
	}

	// Register the xApp as configurable entity
	err = m.registerConfigurable()
	if err != nil {
		log.Error("Failed to register the app as a configurable entity", err)
		return err
	}

	// Fetch config
	if err = m.getConfig(); err != nil {
		log.Errorf("Failed to get config: %v", err)
		return err
	}

	m.Sessions.E2Session.AppConfig = m.Config.AppConfig

	go m.Sessions.E2Session.Run(m.Chans.IndCh, m.Chans.CtrlReqChs, m.Sessions.AdminSession)
	go m.Ctrls.MhoCtrl.Run()
	return nil
}

// Close kills the channels and manager related objects
func (m *Manager) Close() {
	log.Info("Closing Manager")
}

// registerConfigurable registers the xApp as a configurable entity
func (m *Manager) registerConfigurable() error {
	appConfig, err := configurable.RegisterConfigurable(&configurable.RegisterRequest{})
	if err != nil {
		return err
	}
	m.Config.AppConfig = appConfig.Config.(*app.Config)
	return nil
}

func (m *Manager) startNorthboundServer() error {
	s := northbound.NewServer(northbound.NewServerCfg(
		m.Config.CAPath,
		m.Config.KeyPath,
		m.Config.CertPath,
		int16(m.Config.GRPCPort),
		true,
		northbound.SecurityConfig{}))

	s.AddService(nbi.NewService(m.Ctrls.MhoCtrl))

	doneCh := make(chan error)
	go func() {
		err := s.Serve(func(started string) {
			log.Info("Started NBI on ", started)
			close(doneCh)
		})
		if err != nil {
			doneCh <- err
		}
	}()
	return <-doneCh
}

func (m *Manager) getConfig() error {
	if periodicEnabled, err := m.Config.AppConfig.Get(ricapie2.PeriodicEnabledConfigPath); err == nil {
		m.Sessions.E2Session.Trigger[e2sm_mho.MhoTriggerType_MHO_TRIGGER_TYPE_PERIODIC] = periodicEnabled.Value.(bool)
		if m.Sessions.E2Session.Trigger[e2sm_mho.MhoTriggerType_MHO_TRIGGER_TYPE_PERIODIC] {
			log.Infof("Periodic trigger is enabled")
			if interval, err := m.Config.AppConfig.Get(ricapie2.ReportPeriodConfigPath); err == nil {
				if val, err := configutils.ToUint64(interval.Value); err == nil {
					m.Sessions.E2Session.ReportPeriodMs = val
					log.Infof("ReportPeriodMs: %v", m.Sessions.E2Session.ReportPeriodMs)
				}
			} else {
				return err
			}
		} else {
			m.Sessions.E2Session.ReportPeriodMs = 0
		}
	} else {
		return err
	}

	if uponRcvMeasReportEnabled, err := m.Config.AppConfig.Get(ricapie2.UponRcvMeasReportEnabledConfigPath); err == nil {
		m.Sessions.E2Session.Trigger[e2sm_mho.MhoTriggerType_MHO_TRIGGER_TYPE_UPON_RCV_MEAS_REPORT] = uponRcvMeasReportEnabled.Value.(bool)
		log.Info("UponRcvMeasReport trigger is enabled")
	} else {
		return err
	}

	if uponChangeRrcStatusEnabled, err := m.Config.AppConfig.Get(ricapie2.UponChangeRrcStatusEnabledConfigPath); err == nil {
		m.Sessions.E2Session.Trigger[e2sm_mho.MhoTriggerType_MHO_TRIGGER_TYPE_UPON_CHANGE_RRC_STATUS] = uponChangeRrcStatusEnabled.Value.(bool)
		log.Info("UponChangeRrcStatus trigger is enabled")
	} else {
		return err
	}

	if a3OffsetRange, err := m.Config.AppConfig.Get(controller.A3OffsetRangeConfigPath); err == nil {
		if m.Ctrls.MhoCtrl.HoParms.A3OffsetRange, err = configutils.ToUint64(a3OffsetRange.Value); err != nil {
			return err
		}
		log.Infof("A3OffsetRange: %v", m.Ctrls.MhoCtrl.HoParms.A3OffsetRange)
	} else {
		return err
	}

	if hysteresisRange, err := m.Config.AppConfig.Get(controller.HysteresisRangeConfigPath); err == nil {
		if m.Ctrls.MhoCtrl.HoParms.HysteresisRange, err = configutils.ToUint64(hysteresisRange.Value); err != nil {
			return err
		}
		log.Infof("HysteresisRange: %v", m.Ctrls.MhoCtrl.HoParms.HysteresisRange)
	} else {
		return err
	}

	if cellIndividualOffset, err := m.Config.AppConfig.Get(controller.CellIndividualOffsetConfigPath); err == nil {
		if m.Ctrls.MhoCtrl.HoParms.CellIndividualOffset, err = configutils.ToString(cellIndividualOffset.Value); err != nil {
			return err
		}
		log.Infof("CellIndividualOffset: %v", m.Ctrls.MhoCtrl.HoParms.CellIndividualOffset)
	} else {
		return err
	}

	if frequencyOffset, err := m.Config.AppConfig.Get(controller.FrequencyOffsetConfigPath); err == nil {
		if m.Ctrls.MhoCtrl.HoParms.FrequencyOffset, err = configutils.ToString(frequencyOffset.Value); err != nil {
			return err
		}
		log.Infof("FrequencyOffset: %v", m.Ctrls.MhoCtrl.HoParms.FrequencyOffset)
	} else {
		return err
	}

	if timeToTrigger, err := m.Config.AppConfig.Get(controller.TimeToTriggerConfigPath); err == nil {
		if m.Ctrls.MhoCtrl.HoParms.TimeToTrigger, err = configutils.ToString(timeToTrigger.Value); err != nil {
			return err
		}
		log.Infof("TimeToTrigger: %v", m.Ctrls.MhoCtrl.HoParms.TimeToTrigger)
	} else {
		return err
	}

	return nil

}