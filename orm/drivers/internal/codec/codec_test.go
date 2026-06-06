package codec

import (
	"context"
	"testing"

	"github.com/wxdqing/go-orm/orm"
	"google.golang.org/protobuf/types/known/structpb"
)

func TestDecodeRecord_ResetsValueBeforeDecode(t *testing.T) {
	value, err := structpb.NewStruct(map[string]any{"stale": "data"})
	if err != nil {
		t.Fatal(err)
	}
	if len(value.Fields) != 1 {
		t.Fatalf("setup: got %d fields", len(value.Fields))
	}
	err = DecodeRecord(context.Background(), nil, value)
	if err != orm.ErrNotTableRecord {
		t.Fatalf("err = %v, want ErrNotTableRecord", err)
	}
	if len(value.Fields) != 0 {
		t.Fatalf("expected value reset before decode, still has fields: %v", value.Fields)
	}
}
