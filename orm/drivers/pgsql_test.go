//go:build db

package drivers

import (
	"context"
	"testing"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func TestPgsqlDriver_InitDB(t *testing.T) {
	defer Close(context.Background())
	if err := TryInit(context.Background(),
		WithDriverType(DriverTypePostgresSQL),
		WithConfig(testPgsqlConf()),
		WithTables([]proto.Message{wrapperspb.String("")}),
	); err != nil {
		t.Fatal(err)
	}
}
