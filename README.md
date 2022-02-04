<!--
SPDX-FileCopyrightText: 2019-present Open Networking Foundation <info@opennetworking.org>

SPDX-License-Identifier: Apache-2.0
-->

# onos-mho
Mobile Hand Over (MHO) xApplication for µONOS RIC

## Overview 
[µONOS RIC](https://docs.sd-ran.org/master/index.html)  supports collection of per-UE mobility data from E2 nodes for the purpose of enabling development of sophisticated mobility management xApplications. The collected UE data is available to xApplications via the SDKs. `onos-mho` is a sample xApplication that implements a simple A3 event based handover function to demonstrate the mobility management capabilities of µONOS RIC platform.

The E2 Service Model for Mobile HandOver (E2SM-MHO) specifies procedures over the E2 interface to subscribe to and receive indications of UE mobility information and trigger handovers through control messages. The service model defines two types of indications - 1) UE measurement reports and 2) UE RRC state changes. The UE measurement reports can be triggered by A3 events or periodic. The complete E2SM-MHO ASN.1 specification is available in the
[onos-e2-sm](https://github.com/onosproject/onos-e2-sm/blob/master/servicemodels/e2sm_mho_go/v2/e2sm-mho-go.asn1) repo.

The onos-mho xApp interfaces with µONOS RIC, using the Golang SDK, to subscribe for A3 measurement reports from E2 nodes that support the E2SM-MHO service model.  In addition to the A3 Events, onos-mho can be configured to subscribe to RRC state changes and periodic UE measurement reports as well. onos-mho processes the measurement event based on its configured A3 Event parameters. If a handover decision is made, onos-mho sends a handover control message to the E2 node to trigger a handover. In addition, onos-mho also updates `UE-NIB` with UE related information such as RRC state, signal strengths for serving and neighbor cells.

onos-mho is shipped as a [Docker](https://www.docker.com/) image and deployed with [Helm](https://helm.sh/). To build the Docker image, run `make images`.

## Getting Started
The E2SM-MHO service model is currently only supported by [RANSim](https://github.com/onosproject/ran-simulator) and not by real CU/DU/gNB. onos-mho can be deployed, alongwith the other µONOS micro-services and `RANSim`, using the [sdran-helm-charts]. Alternatively, it can also be deployed using `SDRAN-in-a-box (RIAB)`.

### Deploy using Helm charts

Refer to the [µONOS](https://docs.onosproject.org/developers/deploy_with_helm/) and SD-RAN documentation on how to deploy µONOS SD-RAN micro-services with the sd-ran umbrella helm chart. onos-mho is not enabled by default in the sd-ran umbrella chart. To deploy onos-mho with the other µONOS SD-RAN micro-services, either enable it in the sd-ran helm chart or on the command line as follows:
```bash
helm install --set import.ran-simulator.enabled=true --set import.onos-mho.enabled=true sd-ran sd-ran
```
By default, RANSim uses the `model.yaml` model file. To use a different model, e.g. the two-cell-two-node-model, specify the model as follows:
```bash
helm install --set import.ran-simulator.enabled=true --set import.onos-mho.enabled=true --set ran-simulator.pci.modelName=two-cell-two-node-model sd-ran sd-ran
```

### Deploy using RiaB

Refer to the [SDRAN-in-a-box](https://docs.sd-ran.org/master/riab_install_index.html) (RIAB) documentation on how to deploy onos-mho within RIAB. For example, the following command deploys `latest` version of onos-mho and µONOS SD-RAN micro-services:
```bash
make riab OPT=mho VER=latest
```

### Command Line Interface
The following commands are available in [onos-cli](https://github.com/onosproject/onos-cli) for viewing onos-mho related information:
```bash
$ onos mho get cells
$ onos mho get ues
$ onos uenib get ues [-v]
$ onos uenib get ue <ueID> [-v]
```
The information from the above commands on UE handovers and RRC states can be compared to information provided by RANSim using the following CLI commands:
```bash
$ onos ransim get cells
$ onos ransim get ues
$ onos ransim get ue <ueID>
```

### RANSim models
The generic **model.yaml** model, which simulates UEs moving on randomly generated routes, can be used with onos-mho to test handovers. Alternatively, the **two-cell-two-node-model.yaml** model can be used to test onos-mho handover functionality in a more controlled and deterministic manner. Refer to documentation on [RANSim models](https://github.com/onosproject/ran-simulator/blob/master/docs/model.md) for further information.

