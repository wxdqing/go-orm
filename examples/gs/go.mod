module gs

go 1.26.3

require (
	github.com/wxdqing/go-orm v0.0.0-00010101000000-000000000000
	gitee.com/wxdqing/logger.git v0.0.0-20260607044921-ce5e5f28f04a
	github.com/wxdqing/protoc-gen-go-orm v0.0.0-00010101000000-000000000000
	google.golang.org/protobuf v1.36.11
)

require (
	filippo.io/edwards25519 v1.2.0 // indirect
	github.com/go-sql-driver/mysql v1.10.0 // indirect
	github.com/golang/protobuf v1.5.4 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/pgx/v5 v5.10.0 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jinzhu/now v1.1.5 // indirect
	github.com/mitchellh/mapstructure v1.5.0 // indirect
	github.com/mohae/deepcopy v0.0.0-20170929034955-c48cc78d4826 // indirect
	github.com/natefinch/lumberjack v2.0.0+incompatible // indirect
	github.com/samber/lo v1.53.0 // indirect
	github.com/samber/slog-common v0.22.0 // indirect
	github.com/sirupsen/logrus v1.9.4 // indirect
	github.com/spf13/cast v1.10.0 // indirect
	github.com/tencentyun/tcaplusdb-go-sdk v0.2.3 // indirect
	github.com/tencentyun/tsf4g v0.0.1 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	go.uber.org/zap v1.28.0 // indirect
	golang.org/x/sync v0.20.0 // indirect
	golang.org/x/sys v0.45.0 // indirect
	golang.org/x/text v0.37.0 // indirect
	gorm.io/driver/mysql v1.6.0 // indirect
	gorm.io/driver/postgres v1.6.0 // indirect
	gorm.io/gorm v1.31.1 // indirect
	gorm.io/plugin/dbresolver v1.6.2 // indirect
)

replace (
	github.com/wxdqing/go-orm => ../..
	gitee.com/wxdqing/logger.git => ../../../logger
	github.com/wxdqing/protoc-gen-go-orm => ../../../protoc-gen-go-orm
)
