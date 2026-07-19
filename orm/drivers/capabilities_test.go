package drivers

import (
	"context"
	"testing"

	"google.golang.org/protobuf/proto"
)

type testFinder interface {
	Find(context.Context, proto.Message) ([]proto.Message, error)
}

type testCounter interface {
	Count(context.Context, proto.Message) (int64, error)
}

type testTransactor interface {
	RunInTx(context.Context, func(context.Context) error) error
}

type testPinger interface {
	Ping(context.Context) error
}

func TestRedisExposesOnlySupportedCapabilities(t *testing.T) {
	driver := NewRedisDriver()
	if _, ok := driver.(testFinder); ok {
		t.Fatal("redis unexpectedly implements Finder")
	}
	if _, ok := driver.(testCounter); ok {
		t.Fatal("redis unexpectedly implements Counter")
	}
	if _, ok := driver.(testTransactor); ok {
		t.Fatal("redis unexpectedly implements Transactor")
	}
}

func TestMongoExposesOnlySupportedCapabilities(t *testing.T) {
	driver := NewMongoDBDriver()
	if _, ok := driver.(testFinder); ok {
		t.Fatal("mongo unexpectedly implements Finder")
	}
	if _, ok := driver.(testCounter); ok {
		t.Fatal("mongo unexpectedly implements Counter")
	}
	if _, ok := driver.(testTransactor); ok {
		t.Fatal("mongo unexpectedly implements Transactor")
	}
}

func TestTcaplusExposesOnlySupportedCapabilities(t *testing.T) {
	driver := NewTcaplusDbDriver()
	if _, ok := driver.(testCounter); ok {
		t.Fatal("tcaplus unexpectedly implements Counter")
	}
	if _, ok := driver.(testTransactor); ok {
		t.Fatal("tcaplus unexpectedly implements Transactor")
	}
	if _, ok := driver.(testPinger); ok {
		t.Fatal("tcaplus unexpectedly implements Pinger")
	}
}
