package drivers

import (
	"context"
	"testing"
)

func TestTcaplus_CloseDB_Idempotent(t *testing.T) {
	d := &TcaplusDbDriver{}
	ctx := context.Background()
	if err := d.CloseDB(ctx); err != nil {
		t.Fatal(err)
	}
	if err := d.CloseDB(ctx); err != nil {
		t.Fatal(err)
	}
}
