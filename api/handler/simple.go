package handler

import (
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

// SimplifiedApplicationManagerClient uses appID/devID for identifiers instead of identifier structs and does not return empty.Empty
type SimplifiedApplicationManagerClient interface {
	// Applications should first be registered to the Handler with the `RegisterApplication` method
	RegisterApplication(ctx context.Context, appID string, opts ...grpc.CallOption) error
	// GetApplication returns the application with the given identifier (app_id)
	GetApplication(ctx context.Context, appID string, opts ...grpc.CallOption) (*Application, error)
	// SetApplication updates the settings for the application. All fields must be supplied.
	SetApplication(ctx context.Context, in *Application, opts ...grpc.CallOption) error
	// DeleteApplication deletes the application with the given identifier (app_id)
	DeleteApplication(ctx context.Context, appID string, opts ...grpc.CallOption) error
	// GetDevice returns the device with the given identifier (app_id and dev_id)
	GetDevice(ctx context.Context, appID, devID string, opts ...grpc.CallOption) (*Device, error)
	// SetDevice creates or updates a device. All fields must be supplied.
	SetDevice(ctx context.Context, in *Device, opts ...grpc.CallOption) error
	// DeleteDevice deletes the device with the given identifier (app_id and dev_id)
	DeleteDevice(ctx context.Context, appID, devID string, opts ...grpc.CallOption) error
	// GetDevicesForApplication returns all devices that belong to the application with the given identifier (app_id)
	GetDevicesForApplication(ctx context.Context, appID string, opts ...grpc.CallOption) ([]*Device, error)
	// DryUplink simulates processing a downlink message and returns the result
	DryDownlink(ctx context.Context, in *DryDownlinkMessage, opts ...grpc.CallOption) (*DryDownlinkResult, error)
	// DryUplink simulates processing an uplink message and returns the result
	DryUplink(ctx context.Context, in *DryUplinkMessage, opts ...grpc.CallOption) (*DryUplinkResult, error)
	// SimulateUplink simulates an uplink message
	SimulateUplink(ctx context.Context, in *SimulatedUplinkMessage, opts ...grpc.CallOption) error
}

// NewSimplifiedApplicationManagerClient returns a new SimplifiedApplicationManagerClient on the given ClientConn. The
// optional contextFunc is called on every request, passing the context that was passed to the original function.
func NewSimplifiedApplicationManagerClient(cc *grpc.ClientConn, contextFunc func(context.Context) context.Context) SimplifiedApplicationManagerClient {
	return &sAMC{
		cli: NewApplicationManagerClient(cc),
	}
}

type sAMC struct {
	cli         ApplicationManagerClient
	contextFunc func(context.Context) context.Context
}

func (c *sAMC) SetContextFunc(f func(context.Context) context.Context) {
	c.contextFunc = f
}

func (c *sAMC) getContext(ctx context.Context) context.Context {
	if c.contextFunc != nil {
		return c.contextFunc(ctx)
	}
	return ctx
}

func (c *sAMC) RegisterApplication(ctx context.Context, appID string, opts ...grpc.CallOption) error {
	_, err := c.cli.RegisterApplication(c.getContext(ctx), &ApplicationIdentifier{appID}, opts...)
	return err
}
func (c *sAMC) GetApplication(ctx context.Context, appID string, opts ...grpc.CallOption) (*Application, error) {
	return c.cli.GetApplication(c.getContext(ctx), &ApplicationIdentifier{appID}, opts...)
}
func (c *sAMC) SetApplication(ctx context.Context, in *Application, opts ...grpc.CallOption) error {
	_, err := c.cli.SetApplication(c.getContext(ctx), in, opts...)
	return err
}
func (c *sAMC) DeleteApplication(ctx context.Context, appID string, opts ...grpc.CallOption) error {
	_, err := c.cli.DeleteApplication(c.getContext(ctx), &ApplicationIdentifier{appID}, opts...)
	return err
}
func (c *sAMC) GetDevice(ctx context.Context, appID, devID string, opts ...grpc.CallOption) (*Device, error) {
	return c.cli.GetDevice(c.getContext(ctx), &DeviceIdentifier{appID, devID}, opts...)
}
func (c *sAMC) SetDevice(ctx context.Context, in *Device, opts ...grpc.CallOption) error {
	_, err := c.cli.SetDevice(c.getContext(ctx), in, opts...)
	return err
}
func (c *sAMC) DeleteDevice(ctx context.Context, appID, devID string, opts ...grpc.CallOption) error {
	_, err := c.cli.DeleteDevice(c.getContext(ctx), &DeviceIdentifier{appID, devID}, opts...)
	return err
}
func (c *sAMC) GetDevicesForApplication(ctx context.Context, appID string, opts ...grpc.CallOption) ([]*Device, error) {
	devices, err := c.cli.GetDevicesForApplication(c.getContext(ctx), &ApplicationIdentifier{appID}, opts...)
	if err != nil {
		return nil, err
	}
	return devices.Devices, nil
}
func (c *sAMC) DryDownlink(ctx context.Context, in *DryDownlinkMessage, opts ...grpc.CallOption) (*DryDownlinkResult, error) {
	return c.cli.DryDownlink(c.getContext(ctx), in, opts...)
}
func (c *sAMC) DryUplink(ctx context.Context, in *DryUplinkMessage, opts ...grpc.CallOption) (*DryUplinkResult, error) {
	return c.cli.DryUplink(c.getContext(ctx), in, opts...)
}
func (c *sAMC) SimulateUplink(ctx context.Context, in *SimulatedUplinkMessage, opts ...grpc.CallOption) error {
	_, err := c.cli.SimulateUplink(c.getContext(ctx), in, opts...)
	return err
}
