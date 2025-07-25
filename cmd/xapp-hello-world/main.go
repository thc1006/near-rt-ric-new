/*
SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"context"
	"fmt"

	e2api "github.com/onosproject/onos-api/go/onos/e2t/e2/v1beta1"
	topoapi "github.com/onosproject/onos-api/go/onos/topo"
	"github.com/onosproject/onos-lib-go/pkg/logging"
	"github.com/onosproject/onos-ric-sdk-go/pkg/e2/creds"
	sdk "github.com/onosproject/onos-ric-sdk-go/pkg/e2/v1beta1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

var log = logging.GetLogger("xapp-hello-world")

// Manager is a sample xApp manager
type Manager struct {
	e2client sdk.Client
	topoHost string
	topoPort int
}

// NewManager creates a new xApp manager
func NewManager() *Manager {
	return &Manager{
		e2client: sdk.NewClient(),
		topoHost: "onos-topo",
		topoPort: 5150,
	}
}

// Run starts the xApp manager
func (m *Manager) Run() {
	log.Info("Starting xApp-hello-world Manager")
	go m.watchE2Nodes()
	<-context.Background().Done()
}

func (m *Manager) watchE2Nodes() {
	log.Info("Starting to watch E2 nodes")
	clientCreds, err := creds.GetClientCredentials()
	if err != nil {
		log.Fatal(err)
	}
	conn, err := grpc.Dial(fmt.Sprintf("%s:%d", m.topoHost, m.topoPort),
		grpc.WithTransportCredentials(credentials.NewTLS(clientCreds)))
	if err != nil {
		log.Fatal(err)
	}
	client := topoapi.NewTopoClient(conn)

	stream, err := client.Watch(context.Background(), &topoapi.WatchRequest{
		Filters: &topoapi.Filters{
			KindFilter: &topoapi.Filter{
				Filter: &topoapi.Filter_Equal_{
					Equal_: &topoapi.EqualFilter{
						Value: "e2node",
					},
				},
			},
		},
	})
	if err != nil {
		log.Fatal(err)
	}

	for {
		resp, err := stream.Recv()
		if err != nil {
			log.Warn(err)
			continue
		}
		event := resp.Event
		if event.Type == topoapi.EventType_ADDED || event.Type == topoapi.EventType_NONE {
			log.Infof("Received E2 node event: %v", event)
			go m.handleNode(event.Object)
		}
	}
}

func (m *Manager) handleNode(e2Node topoapi.Object) {
	log.Infof("Handling E2 node: %s", e2Node.ID)
	node := m.e2client.Node(sdk.NodeID(e2Node.ID))

	// Create a subscription
	subName := fmt.Sprintf("hello-world-sub-%s", e2Node.ID)
	subSpec := e2api.SubscriptionSpec{
		EventTrigger: e2api.EventTrigger{
			Payload: []byte{}, // Simplified for this example
		},
		Actions: []e2api.Action{
			{
				ID:   1,
				Type: e2api.ActionType_ACTION_TYPE_REPORT,
				SubsequentAction: &e2api.SubsequentAction{
					Type:       e2api.SubsequentActionType_SUBSEQUENT_ACTION_TYPE_CONTINUE,
					TimeToWait: e2api.TimeToWait_TIME_TO_WAIT_ZERO,
				},
			},
		},
	}

	indCh := make(chan e2api.Indication)
	channelID, err := node.Subscribe(context.Background(), subName, subSpec, indCh)
	if err != nil {
		log.Errorf("Failed to subscribe to node %s: %v", e2Node.ID, err)
		return
	}

	log.Infof("Successfully subscribed to node %s with subscription %s on channel %s", e2Node.ID, subName, channelID)

	// Process indications
	for indication := range indCh {
		log.Infof("Received indication from node %s: %v", e2Node.ID, indication)
	}
}

// Close closes the xApp manager
func (m *Manager) Close() {
	log.Info("Closing xApp-hello-world Manager")
}

func main() {
	// Setup logging
	logging.SetLevel(logging.InfoLevel)

	// Create and run the xApp manager
	manager := NewManager()
	manager.Run()
}
