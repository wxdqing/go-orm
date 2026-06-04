package tcaplus

import (
	"context"
	"fmt"

	"github.com/wxdqing/go-orm/orm"
	"github.com/wxdqing/go-orm/orm/driverapi"
	"github.com/wxdqing/go-orm/orm/drivers/internal/codec"
	"github.com/wxdqing/go-orm/orm/drivers/internal/meta"
	logger "git.wxdqing.com/sprout/logger.git"
	"github.com/spf13/cast"
	tcaplus "github.com/tencentyun/tcaplusdb-go-sdk/pb"
	"github.com/tencentyun/tcaplusdb-go-sdk/pb/protocol/option"
	"google.golang.org/protobuf/proto"
)

type Driver struct {
	Cli     *tcaplus.PBClient
	ZoneId  uint32
	asyncId uint64
}

func (t *Driver) InitDB(ctx context.Context, o *driverapi.Options) error {
	cli := tcaplus.NewPBClient()
	zoneId, err := cast.ToUint32E(o.Conf.Tcaplus.ZoneId)
	if err != nil {
		return fmt.Errorf("tcaplus zone id: %w", err)
	}

	m := make([]string, len(o.Tables))
	for i, tb := range o.Tables {
		n := tb.ProtoReflect().Descriptor().Name()
		m[i] = string(n)
	}
	err = cli.Dial(o.Conf.Tcaplus.AppId, []uint32{o.Conf.Tcaplus.ZoneId}, o.Conf.Tcaplus.Addr,
		o.Conf.Tcaplus.Signature, 10, map[uint32][]string{
			zoneId: m,
		})
	if err != nil {
		logger.Errorf("tcaplus client dial failed: %#v conf:%#v", err, o.Conf)
		return err
	}
	t.Cli = cli
	t.ZoneId = zoneId
	return nil
}

func (t *Driver) CloseDB(context.Context) error {
	t.Cli = nil
	return nil
}

func (t *Driver) Save(ctx context.Context, tb proto.Message) error {
	tm := meta.GetMetaByValue(tb)
	if tm == nil {
		return orm.ErrNotTableRecord
	}
	dbObj := tm.NewDbRecordFunc()
	err := codec.EncodeRecord(ctx, dbObj, tb)
	if err != nil {
		logger.Errorf("tcaplus encode record error,err is [%v]", err)
		return err
	}

	opt := Option{
		ResultFlag: option.TcaplusResultFlagAllNewValue,
	}

	// 处理版本控制
	if _, isVp := dbObj.(orm.VersionProvider); isVp {
		var curVersion int32 = 1
		checkObj := tm.NewDbRecordFunc()
		if err := codec.EncodeRecord(ctx, checkObj, tb); err != nil {
			logger.Errorf("tcaplus encode record error,err is [%v]", err)
			return err
		}
		resp, getErr := t.SingleGet(checkObj)
		if getErr == nil {
			curVersion = resp.Version
		} else if ormErr, ok := getErr.(*orm.OrmError); ok && ormErr.Code == int32(orm.CodeErrTcaplusRecordNotExist) {
			// 第一次保存，版本号设置为1
			curVersion = 1
		} else {
			logger.Errorf("Driver.Save failed to get existing record: %v  tb:%v", getErr, tb)
			return getErr
		}
		// 设置当前版本号，并开启自动版本递增检查
		opt.Versions = []int32{curVersion}
		opt.VersionPolicy = option.CheckDataVersionAutoIncrease
	}

	_, err = t.Replace(dbObj, opt)
	if err != nil {
		logger.Errorf("Driver.Save failed: %v  tb:%v", err, tb)
		return err
	}
	return nil
}

func (t *Driver) Get(ctx context.Context, tb proto.Message) error {
	tm := meta.GetMetaByValue(tb)
	if tm == nil {
		return orm.ErrNotTableRecord
	}
	dbObj := tm.NewDbRecordFunc()
	err := codec.EncodeRecord(ctx, dbObj, tb)
	if err != nil {
		logger.Errorf("tcaplus encode record error,err is [%v]", err)
		return err
	}
	resp, err := t.SingleGet(dbObj)
	if err != nil {
		if ormErr := err.(*orm.OrmError); ormErr != nil {
			if ormErr.Code == int32(orm.CodeErrTcaplusRecordNotExist) {
				_ = codec.DecodeRecord(ctx, dbObj, tb)
				return orm.ErrRecordNotFound
			}
			return ormErr
		}
		return err
	}
	dbObj = resp.Message
	if err = codec.DecodeRecord(ctx, dbObj, tb); err != nil {
		logger.Errorf("tcaplus decode record error,err is [%v]", err)
		return err
	}
	return nil
}

func (t *Driver) Find(ctx context.Context, cond proto.Message) ([]proto.Message, error) {
	tm := meta.GetMetaByValue(cond)
	if tm == nil {
		return nil, orm.ErrNotTableRecord
	}
	dbObj := tm.NewDbRecordFunc()
	err := codec.EncodeRecord(ctx, dbObj, cond)
	if err != nil {
		logger.Errorf("tcaplus encode record error,err is [%v]", err)
		return nil, err
	}

	var ret []proto.Message
	idx := dbObj.(orm.IndexProvider).ToIndexMap()
	if len(idx) == 0 {
		return nil, orm.ErrNoIndexSpecified
	}
	idxNames := make([]string, 0, len(idx))
	for k := range idx {
		idxNames = append(idxNames, k)
	}
	resp, err := t.GetByPartKey(dbObj, IndexKeys(idxNames...))
	if err != nil {
		if ormErr := err.(*orm.OrmError); ormErr != nil {
			if ormErr.Code == int32(orm.CodeErrTcaplusRecordNotExist) {
				return ret, nil
			}
		}
		return nil, err
	}
	for _, response := range resp {
		newVal := tm.NewValueFunc()
		dbVal := response.Message
		err = codec.DecodeRecord(ctx, dbVal, newVal)
		if err != nil {
			logger.Errorf("tcaplus decode record error,err is [%v]", err)
			return nil, err
		}
		ret = append(ret, newVal)
	}

	return ret, nil
}

func (t *Driver) Delete(ctx context.Context, tb proto.Message) error {
	tm := meta.GetMetaByValue(tb)
	if tm == nil {
		return orm.ErrNotTableRecord
	}
	dbObj := tm.NewDbRecordFunc()
	err := codec.EncodeRecord(ctx, dbObj, tb)
	if err != nil {
		logger.Errorf("tcaplus encode record error,err is [%v]", err)
		return err
	}

	opt := Option{
		ResultFlag: option.TcaplusResultFlagAllOldValue,
	}
	_, err = t.SingleDelete(dbObj, opt)
	if err != nil {
		logger.Errorf("Driver.Delete tableName:%s, value[%v] err[%v]", meta.GetTableName(tb), tb, err)
		return orm.ErrNotTableRecord
	}
	logger.Debugf("Driver.Delete execute: tableName:%s, value[%v]", meta.GetTableName(tb), tb)
	return nil
}

func (t *Driver) Count(tb proto.Message) (int64, error) {
	return 0, orm.ErrNotImplemented
}

func New() driverapi.Driver {
	return &Driver{}
}
