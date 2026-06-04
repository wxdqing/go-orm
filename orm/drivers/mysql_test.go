//go:build db

package drivers

import (
	"context"
	"testing"
	"time"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

type Base struct {
	CreatedAt time.Time `gorm:"created_at"`
	UpdatedAt time.Time `gorm:"updated_at"`
}

func TestMysqlDriver_InitDB(t *testing.T) {
	defer Close(context.Background())
	if err := TryInit(context.Background(),
		WithDriverType(DriverTypeMySQL),
		WithConfig(testMySQLConf()),
		WithTables([]proto.Message{wrapperspb.String("")}),
	); err != nil {
		t.Fatal(err)
	}
}
