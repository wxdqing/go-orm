package meta

import (
	"github.com/wxdqing/go-orm/orm"
	"google.golang.org/protobuf/proto"
)

type (
	TableMetaData struct {
		DbTableName     string
		DbRecord        proto.Message
		NewDbRecordFunc func() proto.Message

		ValueName    string
		Value        proto.Message
		NewValueFunc func() proto.Message
	}
	ValueProvider interface {
		NewValue() proto.Message
	}
)

var (
	DbTableNameMapping map[string]*TableMetaData
	ValueNameMapping   map[string]*TableMetaData
)

func Init(tables []proto.Message) {
	DbTableNameMapping = make(map[string]*TableMetaData)
	ValueNameMapping = make(map[string]*TableMetaData)

	for _, tb := range tables {
		// meta
		tm := &TableMetaData{}
		// record
		tm.DbRecord = tb
		tm.DbTableName = GetTableName(tb)
		tm.NewDbRecordFunc = func() proto.Message {
			return proto.Clone(tb)
		}
		// value
		r, ok := tb.(ValueProvider)
		if !ok {
			continue
		}
		v := r.NewValue()
		tm.Value = v
		tm.ValueName = GetTableName(v)
		tm.NewValueFunc = func() proto.Message {
			return proto.Clone(v)
		}
		ValueNameMapping[tm.ValueName] = tm
		// table
		DbTableNameMapping[tm.DbTableName] = tm
	}
}

func GetMetaByName(tbName orm.TableName) *TableMetaData {
	if tm, ok := DbTableNameMapping[tbName]; ok {
		return tm
	}
	return nil
}

func GetMetaByValue(value proto.Message) *TableMetaData {
	if value == nil {
		return nil
	}
	if tm, ok := ValueNameMapping[GetTableName(value)]; ok {
		return tm
	}
	return nil
}

func Reset() {
	DbTableNameMapping = nil
	ValueNameMapping = nil
}

func GetTableName(p proto.Message) orm.TableName {
	if p == nil {
		return ""
	}
	return string(p.ProtoReflect().Descriptor().Name())
}
