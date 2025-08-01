package main

import (
	"fmt"
	"log"

	"github.com/oran/near-rt-ric-new/api/proto/e2mgr"
	"github.com/oran/near-rt-ric-new/api/proto/submgr"
	"github.com/oran/near-rt-ric-new/api/proto/rtmgr"
)

func main() {
	fmt.Println("Verifying protobuf definitions...")

	// Test E2 Manager protobuf
	e2mgrReq := &e2mgr.GetNodesRequest{
		NodeType: stringPtr("gNB"),
		Limit:    uint32Ptr(10),
	}
	fmt.Printf("E2 Manager GetNodesRequest: %+v\n", e2mgrReq)

	// Test Subscription Manager protobuf
	submgrReq := &submgr.CreateSubscriptionRequest{
		E2NodeId:      "test-node",
		XappId:        "test-xapp",
		RanFunctionId: 1,
	}
	fmt.Printf("Subscription Manager CreateSubscriptionRequest: %+v\n", submgrReq)

	// Test Routing Manager protobuf
	rtmgrReq := &rtmgr.GetRoutesRequest{
		SourceXapp: stringPtr("test-source"),
		TargetXapp: stringPtr("test-target"),
	}
	fmt.Printf("Routing Manager GetRoutesRequest: %+v\n", rtmgrReq)

	fmt.Println("All protobuf definitions verified successfully!")
}

func stringPtr(s string) *string {
	return &s
}

func uint32Ptr(u uint32) *uint32 {
	return &u
}