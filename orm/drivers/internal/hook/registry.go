package hook

import (
	"context"
	"sync"

	"github.com/wxdqing/go-orm/orm"
	"github.com/wxdqing/go-orm/orm/drivers/internal/codec"
	"google.golang.org/protobuf/proto"
)

// handlerEntry 注册项，带可选表名过滤（空表名表示全局）。
type handlerEntry struct {
	table   orm.TableName
	handler orm.TableHandler
}

// HandlerRegistry 外部处理器注册表。
type HandlerRegistry struct {
	mu       sync.RWMutex
	entries  []handlerEntry
	fallback []handlerEntry // table==""
}

var defaultRegistry = &HandlerRegistry{}

// DefaultRegistry 返回进程级默认注册表。
func DefaultRegistry() *HandlerRegistry {
	return defaultRegistry
}

// Register 注册表级处理器。table 为空表示全局兜底（优先级低于具名表）。
func (r *HandlerRegistry) Register(table orm.TableName, handler orm.TableHandler) {
	if r == nil || handler == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	e := handlerEntry{table: table, handler: handler}
	if table == "" {
		r.fallback = append(r.fallback, e)
		return
	}
	r.entries = append(r.entries, e)
}

// RegisterGlobal 注册全局处理器（匹配所有表，在具名表之后尝试）。
func (r *HandlerRegistry) RegisterGlobal(handler orm.TableHandler) {
	r.Register("", handler)
}

func (r *HandlerRegistry) match(op orm.DriverOp, value proto.Message) []orm.TableHandler {
	if r == nil || value == nil {
		return nil
	}
	table := valueTableName(value)
	r.mu.RLock()
	defer r.mu.RUnlock()
	var matched []orm.TableHandler
	for _, e := range r.entries {
		if e.table != table {
			continue
		}
		if e.handler.Match(op, table, value) {
			matched = append(matched, e.handler)
		}
	}
	for _, e := range r.fallback {
		if e.handler.Match(op, table, value) {
			matched = append(matched, e.handler)
		}
	}
	return matched
}

func valueTableName(value proto.Message) orm.TableName {
	if value == nil {
		return ""
	}
	return orm.TableName(value.ProtoReflect().Descriptor().Name())
}

func (r *HandlerRegistry) matchInitDB() []orm.TableHandler {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var matched []orm.TableHandler
	for _, e := range append(r.entries, r.fallback...) {
		if e.handler.Match(orm.OpInitDB, e.table, nil) {
			matched = append(matched, e.handler)
		}
	}
	return matched
}

func newDriverContext(ctx context.Context, op orm.DriverOp, driver string, value proto.Message) *orm.DriverContext {
	return &orm.DriverContext{
		Ctx:    codec.EnsureCtx(ctx),
		Op:     op,
		Table:  valueTableName(value),
		Value:  value,
		Driver: driver,
	}
}

func (r *HandlerRegistry) before(ctx *orm.DriverContext, value proto.Message) error {
	for _, h := range r.match(ctx.Op, value) {
		if b, ok := h.(orm.BeforeHandler); ok {
			if err := b.Before(ctx, value); err != nil {
				return err
			}
		}
	}
	return nil
}

func (r *HandlerRegistry) after(ctx *orm.DriverContext, value proto.Message, err error) error {
	for _, h := range r.match(ctx.Op, value) {
		if a, ok := h.(orm.AfterHandler); ok {
			if err2 := a.After(ctx, value, err); err2 != nil {
				return err2
			}
		}
	}
	return nil
}

func (r *HandlerRegistry) matchCloseDB() []orm.TableHandler {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var matched []orm.TableHandler
	for _, e := range append(r.entries, r.fallback...) {
		if e.handler.Match(orm.OpCloseDB, e.table, nil) {
			matched = append(matched, e.handler)
		}
	}
	return matched
}

func (r *HandlerRegistry) closeDB(ctx *orm.DriverContext) (bool, error) {
	for _, h := range r.matchCloseDB() {
		if x, ok := h.(orm.CloseDBHandler); ok {
			hr := x.CloseDB(ctx)
			if hr.Handled {
				return true, hr.Err
			}
		}
	}
	return false, nil
}

func (r *HandlerRegistry) initDB(ctx *orm.DriverContext) (bool, error) {
	for _, h := range r.matchInitDB() {
		if x, ok := h.(orm.InitDBHandler); ok {
			hr := x.InitDB(ctx)
			if hr.Handled {
				return true, hr.Err
			}
		}
	}
	return false, nil
}

func (r *HandlerRegistry) save(ctx *orm.DriverContext, value proto.Message) (bool, error) {
	for _, h := range r.match(ctx.Op, value) {
		if x, ok := h.(orm.SaveHandler); ok {
			hr := x.Save(ctx, value)
			if hr.Handled {
				return true, hr.Err
			}
		}
	}
	return false, nil
}

func (r *HandlerRegistry) get(ctx *orm.DriverContext, value proto.Message) (bool, error) {
	for _, h := range r.match(ctx.Op, value) {
		if x, ok := h.(orm.GetHandler); ok {
			hr := x.Get(ctx, value)
			if hr.Handled {
				return true, hr.Err
			}
		}
	}
	return false, nil
}

func (r *HandlerRegistry) find(ctx *orm.DriverContext, cond proto.Message) ([]proto.Message, bool, error) {
	for _, h := range r.match(ctx.Op, cond) {
		if x, ok := h.(orm.FindHandler); ok {
			result, hr := x.Find(ctx, cond)
			if hr.Handled {
				return result, true, hr.Err
			}
		}
	}
	return nil, false, nil
}

func (r *HandlerRegistry) delete(ctx *orm.DriverContext, value proto.Message) (bool, error) {
	for _, h := range r.match(ctx.Op, value) {
		if x, ok := h.(orm.DeleteHandler); ok {
			hr := x.Delete(ctx, value)
			if hr.Handled {
				return true, hr.Err
			}
		}
	}
	return false, nil
}

func (r *HandlerRegistry) count(ctx *orm.DriverContext, cond proto.Message) (int64, bool, error) {
	for _, h := range r.match(ctx.Op, cond) {
		if x, ok := h.(orm.CountHandler); ok {
			n, hr := x.Count(ctx, cond)
			if hr.Handled {
				return n, true, hr.Err
			}
		}
	}
	return 0, false, nil
}

// HandlerFuncs 函数式处理器，便于快速注册。
// 回调字段使用 *Fn 后缀，避免与 TableHandler 接口方法同名冲突。
type HandlerFuncs struct {
	Table string
	MatchFn func(op orm.DriverOp, table orm.TableName, value proto.Message) bool

	BeforeFn  func(ctx *orm.DriverContext, value proto.Message) error
	AfterFn   func(ctx *orm.DriverContext, value proto.Message, err error) error
	InitDBFn  func(ctx *orm.DriverContext) orm.HandleResult
	CloseDBFn func(ctx *orm.DriverContext) orm.HandleResult
	SaveFn    func(ctx *orm.DriverContext, value proto.Message) orm.HandleResult
	GetFn     func(ctx *orm.DriverContext, value proto.Message) orm.HandleResult
	FindFn   func(ctx *orm.DriverContext, cond proto.Message) ([]proto.Message, orm.HandleResult)
	DeleteFn func(ctx *orm.DriverContext, value proto.Message) orm.HandleResult
	CountFn  func(ctx *orm.DriverContext, cond proto.Message) (int64, orm.HandleResult)
}

func (f HandlerFuncs) tableName() orm.TableName {
	return orm.TableName(f.Table)
}

func (f HandlerFuncs) Match(op orm.DriverOp, table orm.TableName, value proto.Message) bool {
	if f.Table != "" && f.Table != string(table) {
		return false
	}
	if f.MatchFn != nil {
		return f.MatchFn(op, table, value)
	}
	return f.Table == string(table) || f.Table == ""
}

func (f HandlerFuncs) Before(ctx *orm.DriverContext, value proto.Message) error {
	if f.BeforeFn != nil {
		return f.BeforeFn(ctx, value)
	}
	return nil
}

func (f HandlerFuncs) After(ctx *orm.DriverContext, value proto.Message, err error) error {
	if f.AfterFn != nil {
		return f.AfterFn(ctx, value, err)
	}
	return nil
}

func (f HandlerFuncs) InitDB(ctx *orm.DriverContext) orm.HandleResult {
	if f.InitDBFn != nil {
		return f.InitDBFn(ctx)
	}
	return orm.PassThrough()
}

func (f HandlerFuncs) CloseDB(ctx *orm.DriverContext) orm.HandleResult {
	if f.CloseDBFn != nil {
		return f.CloseDBFn(ctx)
	}
	return orm.PassThrough()
}

func (f HandlerFuncs) Save(ctx *orm.DriverContext, value proto.Message) orm.HandleResult {
	if f.SaveFn != nil {
		return f.SaveFn(ctx, value)
	}
	return orm.PassThrough()
}

func (f HandlerFuncs) Get(ctx *orm.DriverContext, value proto.Message) orm.HandleResult {
	if f.GetFn != nil {
		return f.GetFn(ctx, value)
	}
	return orm.PassThrough()
}

func (f HandlerFuncs) Find(ctx *orm.DriverContext, cond proto.Message) ([]proto.Message, orm.HandleResult) {
	if f.FindFn != nil {
		return f.FindFn(ctx, cond)
	}
	return nil, orm.PassThrough()
}

func (f HandlerFuncs) Delete(ctx *orm.DriverContext, value proto.Message) orm.HandleResult {
	if f.DeleteFn != nil {
		return f.DeleteFn(ctx, value)
	}
	return orm.PassThrough()
}

func (f HandlerFuncs) Count(ctx *orm.DriverContext, cond proto.Message) (int64, orm.HandleResult) {
	if f.CountFn != nil {
		return f.CountFn(ctx, cond)
	}
	return 0, orm.PassThrough()
}

