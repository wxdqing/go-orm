package drivers

import (
	"context"
	"testing"

	"github.com/wxdqing/go-orm/orm"
	"github.com/wxdqing/go-orm/orm/driverapi"
	"github.com/wxdqing/go-orm/orm/drivers/internal/hook"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func TestHookDriverSaveHandled(t *testing.T) {
	reg := &HandlerRegistry{}
	reg.Register("StringValue", HandlerFuncs{
		Table: "StringValue",
		SaveFn: func(ctx *orm.DriverContext, value proto.Message) orm.HandleResult {
			return orm.Handled(nil)
		},
	})

	inner := &recordingDriver{}
	h := hook.Wrap(inner, string(DriverTypeMySQL), reg)

	v := wrapperspb.String("x")
	if err := h.Save(context.Background(), v); err != nil {
		t.Fatal(err)
	}
	if inner.saveCalled {
		t.Fatal("inner save should be skipped when handled")
	}
}

func TestHookDriverBeforePassesThrough(t *testing.T) {
	var before bool
	reg := &HandlerRegistry{}
	reg.Register("StringValue", HandlerFuncs{
		Table: "StringValue",
		BeforeFn: func(ctx *orm.DriverContext, value proto.Message) error {
			before = true
			return nil
		},
	})

	inner := &recordingDriver{}
	h := hook.Wrap(inner, string(DriverTypeMySQL), reg)

	if err := h.Save(context.Background(), wrapperspb.String("x")); err != nil {
		t.Fatal(err)
	}
	if !before || !inner.saveCalled {
		t.Fatalf("before=%v saveCalled=%v", before, inner.saveCalled)
	}
}

func TestHookDriver_CloseDBHandled(t *testing.T) {
	reg := &HandlerRegistry{}
	reg.RegisterGlobal(HandlerFuncs{
		MatchFn: func(op orm.DriverOp, _ orm.TableName, _ proto.Message) bool {
			return op == orm.OpCloseDB
		},
		CloseDBFn: func(ctx *orm.DriverContext) orm.HandleResult {
			return orm.Handled(nil)
		},
	})

	inner := &recordingDriver{}
	h := hook.Wrap(inner, string(DriverTypeMySQL), reg)

	if err := h.CloseDB(context.Background()); err != nil {
		t.Fatal(err)
	}
	if inner.closeCalled {
		t.Fatal("inner CloseDB should be skipped when handled")
	}
}

type recordingDriver struct {
	saveCalled  bool
	closeCalled bool
}

func (r *recordingDriver) InitDB(context.Context, *driverapi.Options) error { return nil }
func (r *recordingDriver) CloseDB(context.Context) error {
	r.closeCalled = true
	return nil
}
func (r *recordingDriver) Save(context.Context, proto.Message) error {
	r.saveCalled = true
	return nil
}
func (r *recordingDriver) Get(context.Context, proto.Message) error { return nil }
func (r *recordingDriver) Find(context.Context, proto.Message) ([]proto.Message, error) {
	return nil, nil
}
func (r *recordingDriver) Delete(context.Context, proto.Message) error { return nil }
func (r *recordingDriver) Count(context.Context, proto.Message) (int64, error) {
	return 0, nil
}
func (r *recordingDriver) RunInTx(context.Context, func(context.Context) error) error {
	return nil
}
func (r *recordingDriver) Ping(context.Context) error { return nil }
