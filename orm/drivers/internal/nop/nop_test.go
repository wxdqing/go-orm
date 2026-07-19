package nop

import (
	"context"
	"testing"
)

func TestRunInTxExecutesCallback(t *testing.T) {
	called := false
	driver := New().(*Driver)
	if err := driver.RunInTx(context.Background(), func(context.Context) error {
		called = true
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Fatal("RunInTx() did not execute callback")
	}
}
