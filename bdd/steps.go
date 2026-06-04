//go:build bdd

package bdd

import (
	"context"
	"errors"
	"net"
	"os"
	"time"

	"github.com/cucumber/godog"
	"github.com/wxdqing/go-orm/orm"
	"github.com/wxdqing/go-orm/orm/drivers"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

type bddState struct {
	lastErr error
	conf    *orm.Conf
}

func InitializeScenario(ctx *godog.ScenarioContext) {
	s := &bddState{}
	ctx.Before(func(ctx context.Context, _ *godog.Scenario) (context.Context, error) {
		_ = drivers.Close(context.Background())
		s.lastErr = nil
		return ctx, nil
	})
	ctx.After(func(ctx context.Context, _ *godog.Scenario, _ error) (context.Context, error) {
		_ = drivers.Close(context.Background())
		return ctx, nil
	})

	registerLifecycleSteps(ctx, s)
	registerInitSteps(ctx, s)
}

func registerLifecycleSteps(ctx *godog.ScenarioContext, s *bddState) {
	ctx.Step(`^ORM 进程尚未初始化$`, func() error {
		_ = drivers.Close(context.Background())
		return nil
	})
	ctx.Step(`^我直接向默认驱动保存一条玩家记录$`, func() error {
		s.lastErr = drivers.DefaultDbDriver.Save(context.Background(), wrapperspb.String("bdd"))
		return nil
	})
	ctx.Step(`^操作应失败$`, func() error {
		if s.lastErr == nil {
			return errors.New("expected error")
		}
		return nil
	})
	ctx.Step(`^错误类型应为未初始化$`, func() error {
		if !errors.Is(s.lastErr, orm.ErrDbDriverNotInit) {
			return errors.New("want ErrDbDriverNotInit")
		}
		return nil
	})
	ctx.Step(`^我已使用有效配置成功初始化 ORM$`, func() error {
		return drivers.TryInit(context.Background(),
			drivers.WithDriverType(drivers.DriverTypeNop),
			drivers.WithConfig(&orm.Conf{Driver: string(drivers.DriverTypeNop)}),
			drivers.WithTables([]proto.Message{wrapperspb.String("")}),
		)
	})
	ctx.Step(`^我已优雅关闭 ORM$`, func() error {
		return drivers.Close(context.Background())
	})
	ctx.Step(`^我尝试保存一条玩家记录$`, func() error {
		s.lastErr = drivers.DefaultDbDriver.Save(context.Background(), wrapperspb.String("bdd2"))
		return nil
	})
	ctx.Step(`^错误类型应为已关闭或未初始化$`, func() error {
		if !errors.Is(s.lastErr, orm.ErrDbDriverNotInit) {
			return errors.New("want ErrDbDriverNotInit after close")
		}
		return nil
	})
}

func registerInitSteps(ctx *godog.ScenarioContext, s *bddState) {
	ctx.Step(`^我未提供数据库配置$`, func() error {
		_ = drivers.Close(context.Background())
		return nil
	})
	ctx.Step(`^我调用 TryInit$`, func() error {
		if s.conf != nil {
			s.lastErr = drivers.TryInit(context.Background(), drivers.WithConfig(s.conf))
		} else {
			s.lastErr = drivers.TryInit(context.Background(),
				drivers.WithDriverType(drivers.DriverTypeNop),
				drivers.WithTables([]proto.Message{wrapperspb.String("")}),
			)
		}
		return nil
	})
	ctx.Step(`^应返回配置错误$`, func() error {
		if !errors.Is(s.lastErr, orm.ErrInvalidDriverOptions) {
			return errors.New("want ErrInvalidDriverOptions")
		}
		return nil
	})
	ctx.Step(`^ORM 应处于未初始化状态$`, func() error {
		if drivers.IsInitialized() {
			return errors.New("expected not initialized")
		}
		return nil
	})
	ctx.Step(`^我提供了有效的 MySQL 配置$`, func() error {
		s.conf = &orm.Conf{
			Driver: string(drivers.DriverTypeMySQL),
			Mysql: orm.MysqlConf{
				Addr:     envOr("ORM_TEST_MYSQL_ADDR", "127.0.0.1:3306"),
				Name:     envOr("ORM_TEST_MYSQL_DB", "game"),
				User:     envOr("ORM_TEST_MYSQL_USER", "root"),
				Password: envOr("ORM_TEST_MYSQL_PASSWORD", "root123"),
				Startup:  orm.DefaultGormStartup("mysql"),
			},
		}
		return nil
	})
	ctx.Step(`^我未注册任何表定义$`, func() error {
		return nil
	})
	ctx.Step(`^应返回表定义错误$`, func() error {
		if !errors.Is(s.lastErr, orm.ErrInvalidDriverOptions) {
			return errors.New("want table validation error")
		}
		return nil
	})
	ctx.Step(`^游戏 MySQL 服务可用$`, func() error {
		addr := envOr("ORM_TEST_MYSQL_ADDR", "127.0.0.1:3306")
		conn, err := net.DialTimeout("tcp", addr, 2*time.Second)
		if err != nil {
			return godog.ErrSkip
		}
		_ = conn.Close()
		return nil
	})
	ctx.Step(`^我已使用完整配置 TryInit 成功$`, func() error {
		c := &orm.Conf{
			Driver: string(drivers.DriverTypeMySQL),
			Mysql: orm.MysqlConf{
				Addr:     envOr("ORM_TEST_MYSQL_ADDR", "127.0.0.1:3306"),
				Name:     envOr("ORM_TEST_MYSQL_DB", "game"),
				User:     envOr("ORM_TEST_MYSQL_USER", "root"),
				Password: envOr("ORM_TEST_MYSQL_PASSWORD", "root123"),
				Startup:  orm.DefaultGormStartup("mysql"),
			},
		}
		s.lastErr = drivers.TryInit(context.Background(),
			drivers.WithConfig(c),
			drivers.WithTables([]proto.Message{wrapperspb.String("")}),
		)
		return s.lastErr
	})
	ctx.Step(`^我执行 Ping$`, func() error {
		s.lastErr = drivers.Ping(context.Background())
		return nil
	})
	ctx.Step(`^Ping 应成功$`, func() error {
		if s.lastErr != nil {
			return s.lastErr
		}
		return nil
	})
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
