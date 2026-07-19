package tcaplus

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/wxdqing/go-orm/orm"
)

func TestCloseDBClosesClient(t *testing.T) {
	closed := false
	driver := &Driver{closeClient: func() { closed = true }}

	if err := driver.CloseDB(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !closed {
		t.Fatal("CloseDB() did not close the Tcaplus client")
	}
}

func TestAsOrmErrorHandlesPlainError(t *testing.T) {
	if got := asOrmError(errors.New("plain")); got != nil {
		t.Fatalf("asOrmError() = %v, want nil", got)
	}
}

func TestAsOrmErrorFindsWrappedError(t *testing.T) {
	want := orm.New(orm.CodeErrTcaplusDbopTimeout).(*orm.OrmError)
	if got := asOrmError(fmt.Errorf("request: %w", want)); got != want {
		t.Fatalf("asOrmError() = %v, want %v", got, want)
	}
}

func TestDeleteErrorMapsRecordNotFound(t *testing.T) {
	err := deleteError(orm.New(orm.CodeErrTcaplusRecordNotExist))
	if !errors.Is(err, orm.ErrRecordNotFound) {
		t.Fatalf("deleteError() = %v, want ErrRecordNotFound", err)
	}
}

func TestDeleteErrorPreservesUnknownError(t *testing.T) {
	want := errors.New("network")
	if got := deleteError(want); !errors.Is(got, want) {
		t.Fatalf("deleteError() = %v, want %v", got, want)
	}
}

func TestVersionForSaveUsesExistingVersion(t *testing.T) {
	got, err := versionForSave(nil, 7)
	if err != nil || got != 7 {
		t.Fatalf("versionForSave() = (%d, %v), want (7, nil)", got, err)
	}
}

func TestVersionForSaveBootstrapsWrappedRecordNotExist(t *testing.T) {
	wrapped := fmt.Errorf("get: %w", orm.New(orm.CodeErrTcaplusRecordNotExist))
	got, err := versionForSave(wrapped, 0)
	if err != nil || got != 1 {
		t.Fatalf("versionForSave() = (%d, %v), want (1, nil)", got, err)
	}
}

func TestVersionForSavePreservesOtherErrors(t *testing.T) {
	want := errors.New("network")
	got, err := versionForSave(want, 0)
	if got != 0 || !errors.Is(err, want) {
		t.Fatalf("versionForSave() = (%d, %v), want (0, network)", got, err)
	}
}
