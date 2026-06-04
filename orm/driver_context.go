package orm

import (
	"context"

	"google.golang.org/protobuf/proto"
)

// DriverOp 驱动层操作类型。
type DriverOp string

const (
	OpInitDB  DriverOp = "InitDB"
	OpCloseDB DriverOp = "CloseDB"
	OpSave    DriverOp = "Save"
	OpGet     DriverOp = "Get"
	OpFind   DriverOp = "Find"
	OpDelete DriverOp = "Delete"
	OpCount   DriverOp = "Count"
)

// DriverContext 传递给外部处理器的上下文。
// Ctx 与 Driver 方法入参为同一 context，可在 Before/Save 等钩子中通过 ctx.Value(key) 读取业务注入的值。
type DriverContext struct {
	Ctx       context.Context
	Op        DriverOp
	Table     TableName // 逻辑表名（value 消息名，如 Player）
	Value     proto.Message
	Condition proto.Message // Find 时与 Value 相同（可选）
	Driver    string        // mysql / pgsql / tcaplus ...
}

// WithContext 返回携带新 context 的副本。
func (c *DriverContext) WithContext(ctx context.Context) *DriverContext {
	if c == nil {
		return &DriverContext{Ctx: ctx}
	}
	cp := *c
	cp.Ctx = ctx
	return &cp
}

// HandleResult 表示外部处理结果。
type HandleResult struct {
	Handled bool // true 时跳过驱动默认实现
	Err     error
}

// Handled 快捷构造「已处理」结果。
func Handled(err error) HandleResult {
	return HandleResult{Handled: true, Err: err}
}

// PassThrough 快捷构造「交回默认实现」结果。
func PassThrough() HandleResult {
	return HandleResult{Handled: false}
}
