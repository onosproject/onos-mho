// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"flag"

	"github.com/onosproject/onos-lib-go/pkg/certs"
	"github.com/onosproject/onos-lib-go/pkg/logging"
	"github.com/onosproject/onos-mho/pkg/manager"
)

var log = logging.GetLogger("main")

func main() {
	caPath := flag.String("caPath", "", "path to CA certificate")
	keyPath := flag.String("keyPath", "", "path to client private key")
	certPath := flag.String("certPath", "", "path to client certificate")
	configPath := flag.String("configPath", "/etc/onos/config/config.json", "path to config.json file")
	e2tEndpoint := flag.String("e2tEndpoint", "onos-e2t:5150", "E2T service endpoint")
	grpcPort := flag.Int("grpcPort", 5150, "grpc Port number")
	smName := flag.String("smName", "oran-e2sm-mho", "Service model name in RAN function description")
	smVersion := flag.String("smVersion", "v2", "Service model version in RAN function description")

	ready := make(chan bool)

	flag.Parse()

	_, err := certs.HandleCertPaths(*caPath, *keyPath, *certPath, true)
	if err != nil {
		log.Fatal(err)
	}

	log.Info("Starting onos-mho")

	cfg := manager.Config{
		CAPath:      *caPath,
		KeyPath:     *keyPath,
		CertPath:    *certPath,
		ConfigPath:  *configPath,
		E2tEndpoint: *e2tEndpoint,
		GRPCPort:    *grpcPort,
		SMName:      *smName,
		SMVersion:   *smVersion,
	}

	mgr := manager.NewManager(cfg)
	mgr.Run()
	<-ready
}
