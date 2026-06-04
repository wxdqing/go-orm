//go:build db

package kv

import (
	"context"
	"testing"

	"github.com/wxdqing/go-orm/orm"
	"github.com/wxdqing/go-orm/orm/driverapi"
	"github.com/wxdqing/go-orm/testenv"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// 与 projects/tools/docker/mongo-docker-compose.yml（root/root123 @ 27017）对齐。
func TestMongo_DockerCompose_InitAndPKRoundtrip(t *testing.T) {
	ctx := context.Background()
	m := NewMongo().(*Mongo)
	if err := m.InitDB(ctx, &driverapi.Options{
		Conf: &orm.Conf{
			Mongo: orm.MongoConf{
				URI:      testenv.MongoURI(),
				Database: testenv.MongoDatabase(),
			},
		},
	}); err != nil {
		t.Fatalf("InitDB (start mongo-docker-compose.yml): %v", err)
	}
	defer m.CloseDB(ctx)

	coll := m.collection("kv_docker_pk_test")
	_ = coll.Drop(ctx)

	const id int64 = 77001
	filter, err := primaryKeyFilter(map[string]any{"id": id})
	if err != nil {
		t.Fatal(err)
	}
	doc := bson.M{"_pk": pkBSON(map[string]any{"id": id}), "_payload": []byte("docker")}
	if _, err := coll.ReplaceOne(ctx, filter, doc, options.Replace().SetUpsert(true)); err != nil {
		t.Fatal(err)
	}
	var got struct {
		Payload []byte `bson:"_payload"`
	}
	if err := coll.FindOne(ctx, filter).Decode(&got); err != nil {
		t.Fatal(err)
	}
	if string(got.Payload) != "docker" {
		t.Fatalf("payload=%q", got.Payload)
	}
	if _, err := coll.DeleteOne(ctx, filter); err != nil {
		t.Fatal(err)
	}
}
