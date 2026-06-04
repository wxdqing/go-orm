//go:build db

package logic

import (
	"context"
	"testing"

	"github.com/wxdqing/go-orm/testenv"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// 依赖 mongo-docker-compose.yml：root/root123，端口 27017。
func TestMongoDockerCompose_SmokeReplaceAndFind(t *testing.T) {
	ctx := context.Background()
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(testenv.MongoURI()))
	if err != nil {
		t.Fatal(err)
	}
	defer client.Disconnect(ctx)

	coll := client.Database(testenv.MongoDatabase()).Collection("versioned_player_smoke")
	_ = coll.Drop(ctx)

	filter := bson.M{"_pk": bson.M{"id": int64(1)}}
	doc := bson.M{"_pk": bson.M{"id": int64(1)}, "_payload": []byte("x")}
	res, err := coll.ReplaceOne(ctx, filter, doc, options.Replace().SetUpsert(true))
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("replace: matched=%d upserted=%d", res.MatchedCount, res.UpsertedCount)

	var out struct {
		Payload []byte `bson:"_payload"`
	}
	if err := coll.FindOne(ctx, filter).Decode(&out); err != nil {
		t.Fatalf("find: %v", err)
	}
	if string(out.Payload) != "x" {
		t.Fatalf("payload=%q", out.Payload)
	}
}
