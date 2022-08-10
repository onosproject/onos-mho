// SPDX-FileCopyrightText: 2022-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package subscription

import (
	e2api "github.com/onosproject/onos-api/go/onos/e2t/e2/v1beta1"
	"github.com/onosproject/onos-e2-sm/servicemodels/e2sm_rc/pdubuilder"
	e2smcommonies "github.com/onosproject/onos-e2-sm/servicemodels/e2sm_rc/v1/e2sm-common-ies"
	e2smrcies "github.com/onosproject/onos-e2-sm/servicemodels/e2sm_rc/v1/e2sm-rc-ies"
	"github.com/onosproject/onos-lib-go/pkg/logging"
	"github.com/onosproject/onos-mho/pkg/definition"
	"google.golang.org/protobuf/proto"
)

var log = logging.GetLogger()

func CreateEventTriggerDefinition() ([]byte, error) {
	eventTriggerUeEventIDItem, err := pdubuilder.CreateEventTriggerUeeventInfoItem(definition.AssociatedUeEventID)
	if err != nil {
		return nil, err
	}

	eventTriggerUeEventInfo := &e2smrcies.EventTriggerUeeventInfo{
		UeEventList: []*e2smrcies.EventTriggerUeeventInfoItem{eventTriggerUeEventIDItem},
	}

	eventTriggerItem, err := pdubuilder.CreateE2SmRcEventTriggerFormat1Item(definition.E2EventTriggerConditionID, &e2smrcies.MessageTypeChoice{
		MessageTypeChoice: &e2smrcies.MessageTypeChoice_MessageTypeChoiceRrc{
			MessageTypeChoiceRrc: &e2smrcies.MessageTypeChoiceRrc{
				RRcMessage: &e2smcommonies.RrcMessageId{
					RrcType: &e2smcommonies.RrcType{
						RrcType: &e2smcommonies.RrcType_Nr{
							Nr: e2smcommonies.RrcclassNr_RRCCLASS_NR_U_L_DCCH,
						},
					},
					MessageId: definition.RrcMessageIDNrUlDcchMeasurementReport,
				},
			},
		},
	}, nil, nil, eventTriggerUeEventInfo, nil)
	if err != nil {
		return nil, err
	}

	itemList := []*e2smrcies.E2SmRcEventTriggerFormat1Item{eventTriggerItem}

	rcEventTriggerDefinitionFormat3, err := pdubuilder.CreateE2SmRcEventTriggerFormat1(itemList)
	if err != nil {
		return nil, err
	}

	err = rcEventTriggerDefinitionFormat3.Validate()
	if err != nil {
		return nil, err
	}

	protoBytes, err := proto.Marshal(rcEventTriggerDefinitionFormat3)
	if err != nil {
		return nil, err
	}

	return protoBytes, nil
}

func CreateSubscriptionAction() ([]e2api.Action, error) {
	actions := make([]e2api.Action, 0)

	ad, err := pdubuilder.CreateE2SmRcActionDefinitionFormat3(definition.RicInsertStyleTypeID, definition.RicInsertIndicationID, []int64{1})
	if err != nil {
		return nil, err
	}

	err = ad.Validate()
	if err != nil {
		return nil, err
	}

	adProto, err := proto.Marshal(ad)
	if err != nil {
		return nil, err
	}

	action := &e2api.Action{
		ID:      definition.RicActionID,
		Type:    e2api.ActionType_ACTION_TYPE_INSERT,
		Payload: adProto,
	}

	actions = append(actions, *action)
	return actions, nil
}
