package orm

import (
	"errors"
	"testing"
)

func TestOrmErrorUnwrapsCause(t *testing.T) {
	cause := errors.New("backend")
	err := New(CodeErrSystem, cause)
	if !errors.Is(err, cause) {
		t.Fatalf("errors.Is(%v, cause) = false", err)
	}
}
