package orm

import (
	"fmt"
)

const (
	CodeOk        OrmErrorCode = 0
	CodeErrSystem OrmErrorCode = 1

	CodeErrTcaplusRecordNotExist      OrmErrorCode = 20000
	CodeErrTcaplusDbopTimeout         OrmErrorCode = 20001
	CodeErrTcaplusUnpackMessageFailed OrmErrorCode = 20002
	CodeErrTcaplusInvalidVersion      OrmErrorCode = 20003
	CodeErrTcaplusInsertRecordExist   OrmErrorCode = 20004
	CodeErrTcaplusConditionNotMatched OrmErrorCode = 20005
)

var (
	codemap = make(map[OrmErrorCode]string)
)

type OrmErrorCode int32

type OrmError struct {
	Code  int32
	Msg   string
	Cause error
}

func (x *OrmError) Error() string {
	if x.Cause != nil {
		return fmt.Sprintf("%s, cause: %v", x.Msg, x.Cause)
	}
	return x.Msg
}

func (x *OrmError) Unwrap() error {
	if x == nil {
		return nil
	}
	return x.Cause
}

func Errorf(code OrmErrorCode, format string, args ...interface{}) error {
	return newOrmError(code, fmt.Errorf(format, args...))
}

func New(code OrmErrorCode, e ...error) error {
	return newOrmError(code, e...)
}

func newOrmError(code OrmErrorCode, anyCause ...error) *OrmError {
	c := code
	m := code.String()
	cause := error(nil)
	if len(anyCause) > 0 {
		cause = anyCause[0]
	}
	return &OrmError{
		Code:  int32(c),
		Msg:   m,
		Cause: cause,
	}
}

func (c OrmErrorCode) String() string {
	if s, ok := codemap[c]; ok {
		return s
	}
	switch c {
	case CodeOk:
		return "ok"
	case CodeErrSystem:
		return "err_system"
	case CodeErrTcaplusRecordNotExist:
		return "err_tcaplus_record_not_exist"
	case CodeErrTcaplusDbopTimeout:
		return "err_tcaplus_dbop_timeout"
	case CodeErrTcaplusUnpackMessageFailed:
		return "err_tcaplus_unpack_message_faild"
	case CodeErrTcaplusInvalidVersion:
		return "err_tcaplus_invalid_version"
	case CodeErrTcaplusInsertRecordExist:
		return "err_tcaplus_insert_record_exist"
	case CodeErrTcaplusConditionNotMatched:
		return "err_tcaplus_condition_not_matched"
	default:
		return fmt.Sprintf("unknown orm error code %d", c)
	}
}
