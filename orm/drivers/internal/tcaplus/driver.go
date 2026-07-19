package tcaplus

import (
	"context"
	"errors"
	"fmt"

	"github.com/spf13/cast"
	tcaplus "github.com/tencentyun/tcaplusdb-go-sdk/pb"
	"github.com/tencentyun/tcaplusdb-go-sdk/pb/protocol/option"
	"github.com/wxdqing/go-orm/orm"
	"github.com/wxdqing/go-orm/orm/driverapi"
	"github.com/wxdqing/go-orm/orm/drivers/internal/codec"
	"github.com/wxdqing/go-orm/orm/drivers/internal/meta"
	"google.golang.org/protobuf/proto"
)

type Driver struct {
	Cli         *tcaplus.PBClient
	ZoneId      uint32
	asyncId     uint64
	closeClient func()
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
		cli.Close()
		orm.GetLogger().Errorf("tcaplus client dial failed: %v", err)
		return err
	}
	t.Cli = cli
	t.closeClient = cli.Close
	t.ZoneId = zoneId
	return nil
}

func (t *Driver) CloseDB(context.Context) error {
	if t.closeClient != nil {
		t.closeClient()
	}
	t.Cli = nil
	t.closeClient = nil
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
		orm.GetLogger().Errorf("tcaplus encode record error,err is [%v]", err)
		return err
	}

	opt := Option{
		ResultFlag: option.TcaplusResultFlagAllNewValue,
	}

	// 处理版本控制
	if _, isVp := dbObj.(orm.VersionProvider); isVp {
		checkObj := tm.NewDbRecordFunc()
		if err := codec.EncodeRecord(ctx, checkObj, tb); err != nil {
			orm.GetLogger().Errorf("tcaplus encode record error,err is [%v]", err)
			return err
		}
		resp, getErr := t.SingleGet(checkObj)
		existingVersion := int32(0)
		if getErr == nil {
			existingVersion = resp.Version
		}
		curVersion, verErr := versionForSave(getErr, existingVersion)
		if verErr != nil {
			orm.GetLogger().Errorf("Driver.Save failed to get existing record: %v type=%T", verErr, tb)
			return verErr
		}
		// 设置当前版本号，并开启自动版本递增检查
		opt.Versions = []int32{curVersion}
		opt.VersionPolicy = option.CheckDataVersionAutoIncrease
	}

	_, err = t.Replace(dbObj, opt)
	if err != nil {
		orm.GetLogger().Errorf("Driver.Save failed: %v type=%T", err, tb)
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
		orm.GetLogger().Errorf("tcaplus encode record error,err is [%v]", err)
		return err
	}
	resp, err := t.SingleGet(dbObj)
	if err != nil {
		if ormErr := asOrmError(err); ormErr != nil {
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
		orm.GetLogger().Errorf("tcaplus decode record error,err is [%v]", err)
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
		orm.GetLogger().Errorf("tcaplus encode record error,err is [%v]", err)
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
		if ormErr := asOrmError(err); ormErr != nil {
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
			orm.GetLogger().Errorf("tcaplus decode record error,err is [%v]", err)
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
		orm.GetLogger().Errorf("tcaplus encode record error,err is [%v]", err)
		return err
	}

	opt := Option{
		ResultFlag: option.TcaplusResultFlagAllOldValue,
	}
	_, err = t.SingleDelete(dbObj, opt)
	if err != nil {
		orm.GetLogger().Errorf("Driver.Delete tableName:%s type=%T err[%v]", meta.GetTableName(tb), tb, err)
		return deleteError(err)
	}
	orm.GetLogger().Debugf("Driver.Delete execute: tableName:%s type=%T", meta.GetTableName(tb), tb)
	return nil
}

func asOrmError(err error) *orm.OrmError {
	var ormErr *orm.OrmError
	if errors.As(err, &ormErr) {
		return ormErr
	}
	return nil
}

// versionForSave maps SingleGet outcome to the optimistic-lock version for Replace.
// Wrapped OrmError values are accepted so callers/helpers can annotate the cause.
func versionForSave(getErr error, existingVersion int32) (int32, error) {
	if getErr == nil {
		return existingVersion, nil
	}
	if ormErr := asOrmError(getErr); ormErr != nil && ormErr.Code == int32(orm.CodeErrTcaplusRecordNotExist) {
		return 1, nil
	}
	return 0, getErr
}

func deleteError(err error) error {
	if ormErr := asOrmError(err); ormErr != nil && ormErr.Code == int32(orm.CodeErrTcaplusRecordNotExist) {
		return orm.ErrRecordNotFound
	}
	return err
}

func New() driverapi.CoreDriver {
	return &Driver{}
}
