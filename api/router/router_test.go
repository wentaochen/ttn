// Copyright © 2017 The Things Network
// Use of this source code is governed by the MIT license that can be found in the LICENSE file.

package router

import (
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/TheThingsNetwork/go-utils/log"
	"github.com/TheThingsNetwork/ttn/api/gateway"
	"github.com/htdvisser/grpc-testing/test"
	. "github.com/smartystreets/assertions"
	"google.golang.org/grpc"
)

func TestRouter(t *testing.T) {
	waitTime := 10 * time.Millisecond

	a := New(t)

	testLogger := test.NewLogger()
	log.Set(testLogger)
	defer testLogger.Print(t)

	lis, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatalf("Failed to listen: %v", err)
	}
	s := grpc.NewServer()
	server := NewReferenceRouterServer(10)

	RegisterRouterServer(s, server)
	go s.Serve(lis)

	cli := NewClient(DefaultClientConfig)

	log.Get().Info("Expect err about grpc.WithInsecure()")
	cli.AddServer("invalid-config", lis.Addr().String())

	cli.AddServer("test", lis.Addr().String(), grpc.WithInsecure())
	time.Sleep(waitTime)
	defer func() {
		cli.Close()
		time.Sleep(waitTime)
		s.Stop()
	}()

	gtw := cli.NewGatewayStreams("test", "token")
	time.Sleep(waitTime)
	for i := 0; i < 20; i++ {
		gtw.Uplink(&UplinkMessage{})
		gtw.Status(&gateway.Status{})
		time.Sleep(time.Millisecond)
	}
	time.Sleep(waitTime)

	a.So(server.metrics.uplinkMessages, ShouldEqual, 20)
	a.So(server.metrics.gatewayStatuses, ShouldEqual, 20)

	testLogger.Print(t)

	downlink := gtw.Downlink()
	recvDownlink := []*DownlinkMessage{}
	var downlinkClosed bool
	go func() {
		for msg := range downlink {
			fmt.Println(msg)
			recvDownlink = append(recvDownlink, msg)
		}
		downlinkClosed = true
	}()

	server.downlink["test"].ch <- &DownlinkMessage{}

	time.Sleep(waitTime)
	gtw.Close()
	time.Sleep(waitTime)

	a.So(recvDownlink, ShouldHaveLength, 1)
	a.So(downlinkClosed, ShouldBeTrue)
}
