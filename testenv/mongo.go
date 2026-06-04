// Package testenv 提供与本地 Docker 线管一致的默认连接（见 projects/tools/docker/*.yml）。
package testenv

import (
	"fmt"
	"net/url"
	"os"
)

// Mongo 默认与 mongo-docker-compose.yml 一致：
//   MONGO_INITDB_ROOT_USERNAME=root, MONGO_INITDB_ROOT_PASSWORD=root123, 端口 27017
const (
	MongoDefaultUser     = "root"
	MongoDefaultPassword = "root123"
	MongoDefaultAddr     = "127.0.0.1:27017"
	MongoDefaultAuthDB   = "admin"
	MongoDefaultDatabase = "orm" // 驱动侧逻辑库名，首次写入时自动创建
)

// MongoURI 返回用于集成测试的 MongoDB 连接串。
// 优先 ORM_TEST_MONGO_URI；否则用 ORM_TEST_MONGO_* 拼 URI（含 authSource=admin）。
func MongoURI() string {
	if u := os.Getenv("ORM_TEST_MONGO_URI"); u != "" {
		return u
	}
	user := EnvOr("ORM_TEST_MONGO_USER", MongoDefaultUser)
	pass := os.Getenv("ORM_TEST_MONGO_PASSWORD")
	if pass == "" {
		pass = MongoDefaultPassword
	}
	addr := EnvOr("ORM_TEST_MONGO_ADDR", MongoDefaultAddr)
	authDB := EnvOr("ORM_TEST_MONGO_AUTH_DB", MongoDefaultAuthDB)
	if user == "" || pass == "" {
		return fmt.Sprintf("mongodb://%s/?authSource=%s", addr, url.QueryEscape(authDB))
	}
	return fmt.Sprintf(
		"mongodb://%s:%s@%s/?authSource=%s",
		url.QueryEscape(user),
		url.QueryEscape(pass),
		addr,
		url.QueryEscape(authDB),
	)
}

// MongoDatabase ORM KV 驱动使用的库名（非 compose 里的业务库）。
func MongoDatabase() string {
	return EnvOr("ORM_TEST_MONGO_DB", MongoDefaultDatabase)
}

func EnvOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
