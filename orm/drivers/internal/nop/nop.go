package nop

import (
	"context"

	"github.com/wxdqing/go-orm/orm/driverapi"
	"google.golang.org/protobuf/proto"
)

type Driver struct{}

func New() driverapi.Driver {
	return &Driver{}
}

func (n *Driver) InitDB(context.Context, *driverapi.Options) error  { return nil }
func (n *Driver) CloseDB(context.Context) error                     { return nil }
func (n *Driver) Save(context.Context, proto.Message) error         { return nil }
func (n *Driver) Get(context.Context, proto.Message) error          { return nil }
func (n *Driver) Find(context.Context, proto.Message) ([]proto.Message, error) {
	return nil, nil
}
func (n *Driver) Delete(context.Context, proto.Message) error { return nil }
