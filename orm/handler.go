package orm

import "google.golang.org/protobuf/proto"

// TableHandler 表级外部处理器，按需实现对应方法。
// 返回 Handled=true 时，驱动层不再执行该操作的默认逻辑。
type TableHandler interface {
	// Match 是否接管本次调用；table 为 value 的消息名（如 Player）。
	Match(op DriverOp, table TableName, value proto.Message) bool
}

// InitDBHandler 自定义 InitDB。
type InitDBHandler interface {
	TableHandler
	InitDB(ctx *DriverContext) HandleResult
}

// SaveHandler 自定义 Save。
type SaveHandler interface {
	TableHandler
	Save(ctx *DriverContext, value proto.Message) HandleResult
}

// GetHandler 自定义 Get。
type GetHandler interface {
	TableHandler
	Get(ctx *DriverContext, value proto.Message) HandleResult
}

// FindHandler 自定义 Find（按条件查询多条）。
type FindHandler interface {
	TableHandler
	Find(ctx *DriverContext, cond proto.Message) (result []proto.Message, hr HandleResult)
}

// CloseDBHandler 自定义 CloseDB。
type CloseDBHandler interface {
	TableHandler
	CloseDB(ctx *DriverContext) HandleResult
}

// DeleteHandler 自定义 Delete。
type DeleteHandler interface {
	TableHandler
	Delete(ctx *DriverContext, value proto.Message) HandleResult
}

// CountHandler 自定义 Count。
type CountHandler interface {
	TableHandler
	Count(ctx *DriverContext, value proto.Message) (count int64, hr HandleResult)
}

// BeforeHandler 操作前钩子；返回 error 将中断后续流程。
type BeforeHandler interface {
	TableHandler
	Before(ctx *DriverContext, value proto.Message) error
}

// AfterHandler 操作后钩子；在默认实现或自定义实现返回后总会执行。
// After 收到驱动层 error（成功时为 nil）；若 After 返回非 nil，
// hook 包装器用 errors.Join 将其与驱动错误一并返回给调用方。
type AfterHandler interface {
	TableHandler
	After(ctx *DriverContext, value proto.Message, err error) error
}
