package orm

import (
	"errors"

	"google.golang.org/protobuf/proto"
)

type (
	TableName       = string
	TablePrimaryKey = any
	PersistDecoder  interface {
		DecodeTo(value proto.Message) error
	}
	PersistEncoder interface {
		EncodeFrom(value proto.Message) error
	}
	PkProvider interface {
		PrimaryKey() []any
		PrimaryKeyNames() []string
		ToPrimaryKeyMap() map[string]any
		ToPrimaryKeyStruct() any
	}
	IndexProvider interface {
		Indexes() []any
		IndexNames() []string
		ToIndexMap() map[string]any
		ToIndexStruct() any
	}
	VersionProvider interface {
		GetVersion() int64
		SetVersion(ver int64)
	}
)

var (
	ErrNotImplemented          = errors.New("not implemented")
	ErrDbDriverNotInit         = errors.New("db driver not init")
	ErrDbDriverClosed          = errors.New("db driver closed")
	ErrInvalidDriverOptions    = errors.New("invalid driver options")
	ErrRecordNotFound        = errors.New("record not found")
	ErrRecordExists          = errors.New("record exists")
	ErrNotTableRecord        = errors.New("not a table record")
	ErrNoPrimaryKeySpecified = errors.New("no primary key specified")
	ErrNoIndexSpecified      = errors.New("no index specified")
	ErrVersionMismatched     = errors.New("version mismatch")
)
