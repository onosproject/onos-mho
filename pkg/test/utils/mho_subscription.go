// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: LicenseRef-ONF-Member-1.0

package utils

import (
	"context"
	"github.com/onosproject/onos-api/go/onos/e2sub/subscription"
	e2tapi "github.com/onosproject/onos-api/go/onos/e2t/e2"
	"github.com/onosproject/onos-e2-sm/servicemodels/e2sm_rc_pre/pdubuilder"
	e2client "github.com/onosproject/onos-ric-sdk-go/pkg/e2"
	"github.com/onosproject/onos-ric-sdk-go/pkg/e2/indication"
	e2subscription "github.com/onosproject/onos-ric-sdk-go/pkg/e2/subscription"
	"google.golang.org/protobuf/proto"
	"time"
)

func CreateMhoSubscriptionSingle(indCh chan indication.Indication, ctrlReqMap map[string]chan *e2tapi.ControlRequest) (e2subscription.Context, error) {
	clientConfig := e2client.Config{
		AppID: "e2mho-test",
		SubscriptionService: e2client.ServiceConfig{
			Host: SubscriptionServiceHost,
			Port: SubscriptionServicePort,
		},
	}

	client, err := e2client.NewClient(clientConfig)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	time.Sleep(10 * time.Second)

	nodeIDs, err := GetNodeIDs()
	if err != nil {
		return nil, err
	}

	ctrlReqMap[nodeIDs[0]] = make(chan *e2tapi.ControlRequest)

	subReq, err := CreateMhoSubscriptionDetails(nodeIDs[0])
	if err != nil {
		return nil, err
	}

	sub, err := client.Subscribe(ctx, subReq, indCh)
	if err != nil {
		return nil, err
	}

	return sub, nil
}

func CreateMhoSubscriptionDetails(nodeID string) (subscription.SubscriptionDetails, error) {
	data, err := CreateMhoEventTrigger()
	if err != nil {
		return subscription.SubscriptionDetails{}, err
	}
	return subscription.SubscriptionDetails{
		E2NodeID: subscription.E2NodeID(nodeID),
		ServiceModel: subscription.ServiceModel{
			Name:    RcServiceModelName,
			Version: RcServiceModelVersion,
		},
		EventTrigger: subscription.EventTrigger{
			Payload: subscription.Payload{
				Encoding: subscription.Encoding_ENCODING_PROTO,
				Data:     data,
			},
		},
		Actions: []subscription.Action{
			{
				ID:   int32(10),
				Type: subscription.ActionType_ACTION_TYPE_REPORT,
				SubsequentAction: &subscription.SubsequentAction{
					Type:       subscription.SubsequentActionType_SUBSEQUENT_ACTION_TYPE_CONTINUE,
					TimeToWait: subscription.TimeToWait_TIME_TO_WAIT_ZERO,
				},
			},
		},
	}, nil
}

func CreateMhoEventTrigger() ([]byte, error) {
	e2smRcEventTriggerDefinition, err := pdubuilder.CreateE2SmRcPreEventTriggerDefinitionUponChange()
	if err != nil {
		return []byte{}, err
	}

	err = e2smRcEventTriggerDefinition.Validate()
	if err != nil {
		return []byte{}, err
	}

	protoBytes, err := proto.Marshal(e2smRcEventTriggerDefinition)
	if err != nil {
		return []byte{}, err
	}

	return protoBytes, nil
}
