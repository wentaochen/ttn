// Copyright © 2017 The Things Network
// Use of this source code is governed by the MIT license that can be found in the LICENSE file.

package util

import (
	"github.com/TheThingsNetwork/go-account-lib/scope"
	ttnlog "github.com/TheThingsNetwork/go-utils/log"
	"github.com/TheThingsNetwork/ttn/api"
	"github.com/TheThingsNetwork/ttn/api/discovery"
	"github.com/TheThingsNetwork/ttn/api/handler"
	"github.com/TheThingsNetwork/ttn/api/protocol/lorawan"
	"github.com/TheThingsNetwork/ttn/core/types"
	"github.com/TheThingsNetwork/ttn/utils/errors"
	"github.com/spf13/viper"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

type HandlerClient interface {
	handler.SimplifiedApplicationManagerClient
	GetDevAddr(ctx context.Context, constraints []string, opts ...grpc.CallOption) (*types.DevAddr, error)
}

type handlerClient struct {
	handler.SimplifiedApplicationManagerClient
	devAddrManagerClient lorawan.DevAddrManagerClient
}

func (c *handlerClient) GetDevAddr(ctx context.Context, constraints []string, opts ...grpc.CallOption) (*types.DevAddr, error) {
	addr, err := c.devAddrManagerClient.GetDevAddr(ctx, &lorawan.DevAddrRequest{Usage: constraints})
	if err != nil {
		return nil, err
	}
	return addr.DevAddr, nil
}

// GetHandlerManager gets a new HandlerManager for ttnctl
func GetHandlerManager(ctx ttnlog.Interface, appID string) (*grpc.ClientConn, HandlerClient) {
	ctx.WithField("Handler", viper.GetString("handler-id")).Info("Discovering Handler...")
	dscConn, client := GetDiscovery(ctx)
	defer dscConn.Close()
	handlerAnnouncement, err := client.Get(GetContext(ctx), &discovery.GetRequest{
		ServiceName: "handler",
		Id:          viper.GetString("handler-id"),
	})
	if err != nil {
		ctx.WithError(errors.FromGRPCError(err)).Fatal("Could not find Handler")
	}

	token := TokenForScope(ctx, scope.App(appID))

	ctx.WithField("Handler", handlerAnnouncement.NetAddress).Info("Connecting with Handler...")
	hdlConn, err := handlerAnnouncement.Dial()
	if err != nil {
		ctx.WithError(err).Fatal("Could not connect to Handler")
	}

	return hdlConn, &handlerClient{
		SimplifiedApplicationManagerClient: handler.NewSimplifiedApplicationManagerClient(hdlConn, func(ctx context.Context) context.Context {
			return api.ContextWithToken(ctx, token)
		}),
		devAddrManagerClient: lorawan.NewDevAddrManagerClient(hdlConn),
	}
}
