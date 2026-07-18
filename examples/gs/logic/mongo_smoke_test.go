//go:build db

package logic

import (
	"context"
	"testing"

	"github.com/wxdqing/go-orm/testenv"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// Docker 线管探活：Mongo 实例可达（非 ORM CRUD 路径）。
// ORM 集成见 TestIntegration_Mongo_*。
func TestMongoDockerCompose_Reachable(t *testing.T) {
	ctx := context.Background()
	client, err := mongo.Connect(options.Client().ApplyURI(testenv.MongoURI()))
	if err != nil {
		t.Skipf("mongo connect: %v", err)
	}
	defer client.Disconnect(ctx)
	if err := client.Ping(ctx, nil); err != nil {
		t.Skipf("mongo ping: %v", err)
	}
}
