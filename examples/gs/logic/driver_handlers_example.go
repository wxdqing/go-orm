package logic

import (
	"fmt"
	"sync/atomic"

	"github.com/wxdqing/go-orm/orm"
	"github.com/wxdqing/go-orm/orm/drivers"
	"gs/pbtest"
	logger "gitee.com/wxdqing/logger.git"
	"google.golang.org/protobuf/proto"
)

// ExampleHandlerRegistry 返回示例用处理器注册表。
// 建议 Init 时通过 WithHandlerRegistry 传入，避免与进程默认注册表混用。
//
// 业务 context 传参：ctx := context.WithValue(context.Background(), key, val);
// drivers.DefaultDbDriver.Save(ctx, msg)；Handler 内用 dctx.Ctx.Value(key) 读取。
func ExampleHandlerRegistry() *drivers.HandlerRegistry {
	reg := &drivers.HandlerRegistry{}

	// 1) 函数式：Save 前校验 Player.name
	reg.Register("Player", drivers.HandlerFuncs{
		Table: "Player",
		BeforeFn: func(ctx *orm.DriverContext, value proto.Message) error {
			p, ok := value.(*pbtest.Player)
			if !ok {
				return nil
			}
			if p.GetName() == "" {
				return fmt.Errorf("driver handler: player name is required")
			}
			return nil
		},
		AfterFn: func(ctx *orm.DriverContext, value proto.Message, err error) error {
			if err != nil {
				return nil
			}
			logger.Infof("[driver-handler] %s %s completed", ctx.Driver, ctx.Op)
			return nil
		},
	})

	// 2) struct 实现：见 versionedPlayerSaveHook
	reg.Register("VersionedPlayer", versionedPlayerSaveHook{})

	return reg
}

// versionedPlayerSaveHook 演示 struct 实现 TableHandler + SaveHandler。
// Save 完全由业务接管（不落默认 GORM），适合走缓存/消息队列等路径。
type versionedPlayerSaveHook struct{}

var versionedPlayerCustomSaveCount atomic.Int32

func (versionedPlayerSaveHook) Match(op orm.DriverOp, table orm.TableName, value proto.Message) bool {
	return table == "VersionedPlayer" && op == orm.OpSave
}

func (versionedPlayerSaveHook) Save(ctx *orm.DriverContext, value proto.Message) orm.HandleResult {
	vp, ok := value.(*pbtest.VersionedPlayer)
	if !ok {
		return orm.PassThrough()
	}
	// 示例：自定义持久化（此处仅计数 + 日志，真实场景可写 Redis / 发 MQ 等）
	versionedPlayerCustomSaveCount.Add(1)
	logger.Infof("[driver-handler] custom save VersionedPlayer id=%d name=%s", vp.GetId(), vp.GetName())
	return orm.Handled(nil)
}

// VersionedPlayerCustomSaveCount 返回示例自定义 Save 调用次数（测试断言用）。
func VersionedPlayerCustomSaveCount() int32 {
	return versionedPlayerCustomSaveCount.Load()
}
