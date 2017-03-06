// Copyright © 2017 The Things Network
// Use of this source code is governed by the MIT license that can be found in the LICENSE file.

package router

import (
	"context"
	"io"
	"sync"

	"github.com/TheThingsNetwork/go-utils/grpc/restartstream"
	"github.com/TheThingsNetwork/go-utils/log"
	"github.com/TheThingsNetwork/ttn/api"
	"github.com/TheThingsNetwork/ttn/api/gateway"
	"github.com/golang/protobuf/ptypes/empty"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

// GenericStream is used for sending to and receiving from the router.
type GenericStream interface {
	Uplink(*UplinkMessage)
	Status(*gateway.Status)
	Downlink() <-chan *DownlinkMessage
	Close()
}

// ClientConfig for router Client
type ClientConfig struct {
	BufferSize int
}

// DefaultClientConfig for router Client
var DefaultClientConfig = ClientConfig{
	BufferSize: 10,
}

// NewClient creates a new Client with the given configuration
func NewClient(config ClientConfig) *Client {
	ctx, cancel := context.WithCancel(context.Background())

	return &Client{
		log:    log.Get(),
		ctx:    ctx,
		cancel: cancel,

		config: config,
	}
}

// Client for router
type Client struct {
	log    log.Interface
	ctx    context.Context
	cancel context.CancelFunc

	config      ClientConfig
	serverConns []*serverConn
}

var defaultDialOptions = []grpc.DialOption{
	grpc.WithBlock(),
	grpc.FailOnNonTempDialError(false),
	grpc.WithStreamInterceptor(restartstream.Interceptor(restartstream.DefaultSettings)),
}

// AddServer adds a new router server
func (c *Client) AddServer(name, address string, opts ...grpc.DialOption) {
	log := c.log.WithFields(log.Fields{"Router": name, "Address": address})
	log.Info("Adding Router server")

	s := &serverConn{
		ctx:   log,
		name:  name,
		ready: make(chan struct{}),
	}
	c.serverConns = append(c.serverConns, s)

	go func() {
		conn, err := grpc.DialContext(
			c.ctx,
			address,
			append(defaultDialOptions, opts...)...,
		)
		if err != nil {
			log.WithError(err).Error("Could not connect to Router server")
			close(s.ready)
			return
		}
		s.conn = conn
		close(s.ready)
	}()
}

// AddConn adds a router conn
func (c *Client) AddConn(name string, conn *grpc.ClientConn) {
	log := c.log.WithField("Router", name)
	log.Info("Adding Router server")
	s := &serverConn{
		ctx:  log,
		name: name,
		conn: conn,
	}
	c.serverConns = append(c.serverConns, s)
}

// Close the client and all its connections
func (c *Client) Close() {
	c.cancel()
	for _, server := range c.serverConns {
		server.Close()
	}
}

type serverConn struct {
	ctx  log.Interface
	name string

	ready chan struct{}
	conn  *grpc.ClientConn
}

func (c *serverConn) Close() {
	if c.ready != nil {
		<-c.ready
	}
	if c.conn != nil {
		c.conn.Close()
	}
}

type gatewayStreams struct {
	log    log.Interface
	ctx    context.Context
	cancel context.CancelFunc

	mu       sync.RWMutex
	uplink   map[string]chan *UplinkMessage
	downlink map[string]chan *DownlinkMessage
	status   map[string]chan *gateway.Status
}

func (s *gatewayStreams) Uplink(msg *UplinkMessage) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	s.log.Debug("Sending UplinkMessage to router")
	for serverName, ch := range s.uplink {
		select {
		case ch <- msg:
		default:
			s.log.WithField("Router", serverName).Warn("UplinkMessage buffer full")
		}
	}
}

func (s *gatewayStreams) Status(msg *gateway.Status) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	s.log.Debug("Sending Status to router")
	for serverName, ch := range s.status {
		select {
		case ch <- msg:
		default:
			s.log.WithField("Router", serverName).Warn("GatewayStatus buffer full")
		}
	}
}

func (s *gatewayStreams) Downlink() <-chan *DownlinkMessage {
	s.mu.RLock()
	defer s.mu.RUnlock()
	s.log.Debug("Subscribing to DownlinkMessage from router")
	down := make(chan *DownlinkMessage)
	var wg sync.WaitGroup
	for serverName, ch := range s.downlink {
		wg.Add(1)
		go func(serverName string, ch chan *DownlinkMessage) {
			for msg := range ch {
				s.log.WithField("Router", serverName).Debug("Receiving DownlinkMessage from router")
				down <- msg
			}
			s.log.WithField("Router", serverName).Debug("DownlinkMessage subscription from router ended")
			wg.Done()
		}(serverName, ch)
	}
	go func() {
		wg.Wait()
		close(down)
	}()
	return down
}

func (s *gatewayStreams) Close() {
	s.cancel()
}

// NewGatewayStreams returns new streams using the given gateway ID and token
func (c *Client) NewGatewayStreams(id string, token string) GenericStream {
	log := c.log.WithField("GatewayID", id)
	ctx, cancel := context.WithCancel(c.ctx)
	ctx = api.ContextWithID(ctx, id)
	ctx = api.ContextWithToken(ctx, token)
	s := &gatewayStreams{
		log:    log,
		ctx:    ctx,
		cancel: cancel,

		uplink:   make(map[string]chan *UplinkMessage),
		downlink: make(map[string]chan *DownlinkMessage),
		status:   make(map[string]chan *gateway.Status),
	}

	// Hook up the router servers
	for _, server := range c.serverConns {
		go func(server *serverConn) {
			if server.ready != nil {
				select {
				case <-ctx.Done():
					return
				case <-server.ready:
				}
			}
			if server.conn == nil {
				return
			}
			log := log.WithField("Router", server.name)
			cli := NewRouterClient(server.conn)

			logStreamErr := func(streamName string, err error) {
				switch {
				case err == nil:
					log.Debugf("%s stream closed", streamName)
				case err == io.EOF:
					log.WithError(err).Debugf("%s stream ended", streamName)
				case err == context.Canceled || grpc.Code(err) == codes.Canceled:
					log.WithError(err).Debugf("%s stream canceled", streamName)
				case err == context.DeadlineExceeded || grpc.Code(err) == codes.DeadlineExceeded:
					log.WithError(err).Debugf("%s stream deadline exceeded", streamName)
				case grpc.ErrorDesc(err) == grpc.ErrClientConnClosing.Error():
					log.WithError(err).Debugf("%s stream connection closed", streamName)
				default:
					log.WithError(err).Warnf("%s stream closed unexpectedly", streamName)
				}
			}

			// Stream channels
			chUplink := make(chan *UplinkMessage, c.config.BufferSize)
			chDownlink := make(chan *DownlinkMessage, c.config.BufferSize)
			chStatus := make(chan *gateway.Status, c.config.BufferSize)

			// Uplink stream
			uplink, err := cli.Uplink(ctx)
			if err != nil {
				log.WithError(err).Warn("Could not set up Uplink stream")
			} else {
				go func() {
					err := uplink.RecvMsg(new(empty.Empty))
					logStreamErr("Uplink", err)
				}()
			}

			// Downlink stream
			downlink, err := cli.Subscribe(ctx, &SubscribeRequest{})
			if err != nil {
				log.WithError(err).Warn("Could not set up Subscribe stream")
			} else {
				go func() {
					defer close(chDownlink)
					for {
						msg, err := downlink.Recv()
						if err != nil {
							logStreamErr("Subscribe", err)
							return
						}
						select {
						case chDownlink <- msg:
						default:
							log.Warn("Downlink buffer full")
						}
					}
				}()
			}

			// Status stream
			status, err := cli.GatewayStatus(ctx)
			if err != nil {
				log.WithError(err).Warn("Could not set up GatewayStatus stream")
			} else {
				go func() {
					err := status.RecvMsg(new(empty.Empty))
					logStreamErr("GatewayStatus", err)
				}()
			}

			s.mu.Lock()
			s.uplink[server.name] = chUplink
			s.downlink[server.name] = chDownlink
			s.status[server.name] = chStatus
			s.mu.Unlock()

			log.Debug("Start handling Gateway streams")
			defer log.Debug("Done handling Gateway streams")
			for {
				select {
				case <-ctx.Done():
					return
				case msg := <-chStatus:
					if err := status.Send(msg); err != nil {
						log.WithError(err).Warn("Could not send GatewayStatus to router")
						return
					}
				case msg := <-chUplink:
					if err := uplink.Send(msg); err != nil {
						log.WithError(err).Warn("Could not send UplinkMessage to router")
						return
					}
				}
			}

		}(server)
	}

	return s
}
