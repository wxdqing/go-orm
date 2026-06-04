package drivers

import (
	"context"
	"errors"
	"testing"

	"github.com/wxdqing/go-orm/orm"
)

func TestTryInit_UnsupportedDriverReturnsError(t *testing.T) {
	_ = Close(context.Background())
	err := TryInit(context.Background(),
		WithConfig(&orm.Conf{Driver: "unknown"}),
		withTestTable(),
	)
	if !errors.Is(err, orm.ErrInvalidDriverOptions) {
		t.Fatalf("err = %v", err)
	}
	if IsInitialized() {
		t.Fatal("expected not initialized")
	}
}
