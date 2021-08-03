// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: LicenseRef-ONF-Member-1.0

package manager

import (
	"context"
	e2api "github.com/onosproject/onos-api/go/onos/e2t/e2/v1beta1"
	"github.com/onosproject/onos-lib-go/pkg/logging"
	"github.com/onosproject/onos-lib-go/pkg/northbound"
	"github.com/onosproject/onos-mho/pkg/broker"
	appConfig "github.com/onosproject/onos-mho/pkg/config"
	"github.com/onosproject/onos-mho/pkg/mho"
	nbi "github.com/onosproject/onos-mho/pkg/northbound"
	"github.com/onosproject/onos-mho/pkg/southbound/e2"
	"github.com/onosproject/onos-mho/pkg/store"
	"github.com/onosproject/onos-mho/pkg/uenib"
	app "github.com/onosproject/onos-ric-sdk-go/pkg/config/app/default"
)

var log = logging.GetLogger("manager")

// Config is a manager configuration
type Config struct {
	CAPath      string
	KeyPath     string
	CertPath    string
	ConfigPath  string
	E2tEndpoint string
	GRPCPort    int
	AppConfig   *app.Config
	SMName      string
	SMVersion   string
}

// NewManager creates a new manager
func NewManager(config Config) *Manager {
	appCfg, err := appConfig.NewConfig(config.ConfigPath)
	if err != nil {
		log.Warn(err)
	}
	subscriptionBroker := broker.NewBroker()
	ueStore := store.NewStore()
	cellStore := store.NewStore()

	indCh := make(chan *mho.E2NodeIndication)
	ctrlReqChs := make(map[string]chan *e2api.ControlMessage)

	e2Manager, err := e2.NewManager(
		e2.WithE2TAddress("onos-e2t", 5150),
		e2.WithServiceModel(e2.ServiceModelName(config.SMName),
			e2.ServiceModelVersion(config.SMVersion)),
		e2.WithAppConfig(appCfg),
		e2.WithAppID("onos-mho"),
		e2.WithBroker(subscriptionBroker),
		e2.WithIndChan(indCh),
		e2.WithCtrlReqChs(ctrlReqChs),
		e2.WithUeStore(ueStore),
		e2.WithCellStore(cellStore))

	if err != nil {
		log.Warn(err)
	}

	manager := &Manager{
		appConfig:   appCfg,
		config:      config,
		e2Manager:   e2Manager,
		mhoCtrl:     mho.NewMhoController(appCfg, indCh, ctrlReqChs, ueStore, cellStore),
		ueStore: ueStore,
		cellStore: cellStore,
	}
	return manager
}

// Manager is a manager for the MHO xAPP service
type Manager struct {
	appConfig   appConfig.Config
	config      Config
	e2Manager   e2.Manager
	mhoCtrl     *mho.MhoCtrl
	ueStore store.Store
	cellStore store.Store
}

// Run starts the manager and the associated services
func (m *Manager) Run() {
	log.Info("Running Manager")
	if err := m.start(); err != nil {
		log.Fatal("Unable to run Manager", err)
	}
}

func (m *Manager) start() error {
	// Start Northbound server
	err := m.startNorthboundServer()
	go m.startUENIBUpdater()
	if err != nil {
		return err
	}

	err = m.e2Manager.Start()
	if err != nil {
		log.Warn(err)
		return err
	}

	go m.mhoCtrl.Run(context.Background())

	return nil
}

// Close kills the channels and manager related objects
func (m *Manager) Close() {
	log.Info("Closing Manager")
}

func (m *Manager) startNorthboundServer() error {
	s := northbound.NewServer(northbound.NewServerCfg(
		m.config.CAPath,
		m.config.KeyPath,
		m.config.CertPath,
		int16(m.config.GRPCPort),
		true,
		northbound.SecurityConfig{}))

	//TODO - MHO northbound service
	s.AddService(nbi.NewService(m.ueStore, m.cellStore))

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

func (m *Manager) startUENIBUpdater() {
	ctx := context.Background()
	s := uenib.NewUENIBClient(ctx, m.config.CertPath, m.config.KeyPath, m.ueStore)
	s.Run(ctx)
}

// GetMhoStore returns measurement store
func (m *Manager) GetUeStore() store.Store {
	return m.ueStore
}
