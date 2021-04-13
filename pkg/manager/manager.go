// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: LicenseRef-ONF-Member-1.0

package manager

import (
	e2tapi "github.com/onosproject/onos-api/go/onos/e2t/e2"
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

// Manager is a manager for the PCI xAPP service
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

	// Start Southbound client to watch indication messages
	m.Sessions.E2Session.ReportPeriodMs, err = m.getReportPeriod()
	m.Sessions.E2Session.AppConfig = m.Config.AppConfig
	if err != nil {
		log.Errorf("Failed to get report period so period is set to 0ms: %v", err)
	}

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

func (m *Manager) getReportPeriod() (uint64, error) {
	interval, _ := m.Config.AppConfig.Get(ricapie2.ReportPeriodConfigPath)
	val, err := configutils.ToUint64(interval.Value)
	if err != nil {
		log.Error(err)
		return 0, err
	}

	log.Infof("Received period value: %v", val)
	return val, nil
}
