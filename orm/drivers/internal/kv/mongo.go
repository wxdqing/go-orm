package kv

import (
	"context"
	"fmt"

	"github.com/wxdqing/go-orm/orm"
	"github.com/wxdqing/go-orm/orm/driverapi"
	"github.com/wxdqing/go-orm/orm/drivers/internal/codec"
	"github.com/wxdqing/go-orm/orm/drivers/internal/meta"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"google.golang.org/protobuf/proto"
)

type Mongo struct {
	client *mongo.Client
	dbName string
}

func NewMongo() driverapi.Driver {
	return &Mongo{}
}

// Client returns the underlying mongo client after InitDB.
func (m *Mongo) Client() *mongo.Client {
	return m.client
}

func (m *Mongo) InitDB(ctx context.Context, o *driverapi.Options) error {
	uri := o.Conf.Mongo.URI
	if uri == "" {
		uri = "mongodb://localhost:27017"
	}
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		return fmt.Errorf("mongo connect: %w", err)
	}
	if err := client.Ping(ctx, nil); err != nil {
		_ = client.Disconnect(ctx)
		return fmt.Errorf("mongo ping: %w", err)
	}
	m.client = client
	m.dbName = o.Conf.Mongo.Database
	if m.dbName == "" {
		m.dbName = "orm"
	}
	return nil
}

func (m *Mongo) CloseDB(ctx context.Context) error {
	if m.client == nil {
		return nil
	}
	err := m.client.Disconnect(ctx)
	m.client = nil
	return err
}

func (m *Mongo) Save(ctx context.Context, value proto.Message) error {
	if m.client == nil {
		return orm.ErrDbDriverNotInit
	}
	tm := meta.GetMetaByValue(value)
	if tm == nil {
		return orm.ErrNotTableRecord
	}
	dbObj := tm.NewDbRecordFunc()
	if err := codec.EncodeRecord(ctx, dbObj, value); err != nil {
		return err
	}
	pk := dbObj.(orm.PkProvider).ToPrimaryKeyMap()
	filter, err := primaryKeyFilter(pk)
	if err != nil {
		return err
	}
	payload, err := proto.Marshal(value)
	if err != nil {
		return err
	}
	doc := bson.M{"_payload": payload, "_pk": pkBSON(pk)}
	coll := m.collection(tableName(dbObj))
	_, err = coll.ReplaceOne(ctx, filter, doc, options.Replace().SetUpsert(true))
	return err
}

func (m *Mongo) Get(ctx context.Context, value proto.Message) error {
	if m.client == nil {
		return orm.ErrDbDriverNotInit
	}
	tm := meta.GetMetaByValue(value)
	if tm == nil {
		return orm.ErrNotTableRecord
	}
	dbObj := tm.NewDbRecordFunc()
	if err := codec.EncodeRecord(ctx, dbObj, value); err != nil {
		return err
	}
	pk := dbObj.(orm.PkProvider).ToPrimaryKeyMap()
	filter, err := primaryKeyFilter(pk)
	if err != nil {
		return err
	}
	var doc struct {
		Payload []byte `bson:"_payload"`
	}
	err = m.collection(tableName(dbObj)).FindOne(ctx, filter).Decode(&doc)
	if err == mongo.ErrNoDocuments {
		return orm.ErrRecordNotFound
	}
	if err != nil {
		return err
	}
	proto.Reset(value)
	return proto.Unmarshal(doc.Payload, value)
}

func (m *Mongo) Find(context.Context, proto.Message) ([]proto.Message, error) {
	return nil, orm.ErrNotImplemented
}
func (m *Mongo) Count(context.Context, proto.Message) (int64, error) {
	return 0, orm.ErrNotImplemented
}
func (m *Mongo) RunInTx(context.Context, func(context.Context) error) error {
	return orm.ErrNotImplemented
}
func (m *Mongo) Ping(ctx context.Context) error {
	if m.client == nil {
		return orm.ErrDbDriverNotInit
	}
	return m.client.Ping(ctx, nil)
}

func (m *Mongo) Delete(ctx context.Context, value proto.Message) error {
	if m.client == nil {
		return orm.ErrDbDriverNotInit
	}
	tm := meta.GetMetaByValue(value)
	if tm == nil {
		return orm.ErrNotTableRecord
	}
	dbObj := tm.NewDbRecordFunc()
	if err := codec.EncodeRecord(ctx, dbObj, value); err != nil {
		return err
	}
	pk := dbObj.(orm.PkProvider).ToPrimaryKeyMap()
	filter, err := primaryKeyFilter(pk)
	if err != nil {
		return err
	}
	res, err := m.collection(tableName(dbObj)).DeleteOne(ctx, filter)
	if err != nil {
		return err
	}
	if res.DeletedCount == 0 {
		return orm.ErrRecordNotFound
	}
	return nil
}

func (m *Mongo) collection(table string) *mongo.Collection {
	return m.client.Database(m.dbName).Collection(table)
}

func pkBSON(pk map[string]any) bson.M {
	out := make(bson.M, len(pk))
	for k, v := range pk {
		out[k] = v
	}
	return out
}

func primaryKeyFilter(pk map[string]any) (bson.M, error) {
	if len(pk) == 0 {
		return nil, orm.ErrNoPrimaryKeySpecified
	}
	return bson.M{"_pk": pkBSON(pk)}, nil
}
