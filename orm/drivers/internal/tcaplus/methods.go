package tcaplus

import (
	"errors"
	"fmt"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/wxdqing/go-orm/orm"
	"github.com/wxdqing/go-orm/orm/drivers/internal/meta"
	logger "gitee.com/wxdqing/logger.git"
	"github.com/mohae/deepcopy"
	"github.com/tencentyun/tcaplusdb-go-sdk/pb/protocol/cmd"
	"github.com/tencentyun/tcaplusdb-go-sdk/pb/protocol/option"
	"github.com/tencentyun/tcaplusdb-go-sdk/pb/response"
	"github.com/tencentyun/tcaplusdb-go-sdk/pb/terror"
	"github.com/tencentyun/tcaplusdb-go-sdk/pb/traverser"
	"google.golang.org/protobuf/proto"
)

type Response struct {
	Message proto.Message
	Version int32
	Index   int32
	RespCh  <-chan response.TcaplusResponse
}

func ParseOptions(options ...Option) *Option {
	if len(options) == 0 {
		return &Option{}
	}
	ret := &Option{}
	for _, v := range options {
		ret.Flags = v.Flags
		ret.AddableIncreaseFlag = v.AddableIncreaseFlag
		ret.TimeOut = v.TimeOut
		ret.Versions = v.Versions
		ret.VersionPolicy = v.VersionPolicy
		ret.ResultFlag = v.ResultFlag
		ret.Condition = v.Condition
		ret.FieldNames = v.FieldNames
		ret.IndexKeys = v.IndexKeys
		ret.BatchResult = v.BatchResult
		ret.ListShiftFlag = v.ListShiftFlag
		ret.Indexs = append(ret.Indexs, v.Indexs...)
		ret.Async = v.Async
	}
	return ret
}

func (t *Driver) Operate(cmdType int, requests []proto.Message, options ...Option) ([]*Response, error) {
	opts := ParseOptions(options...)
	responses, err := t.internalOperate(cmdType, requests, opts)
	if err != nil {
		tbName := ""
		if len(requests) > 0 {
			tbName = meta.GetTableName(requests[0])
		}
		logger.Errorf("tcaplus driver table op error: tb=%s, cmd=0x%x, options=%#v, err:%v", tbName, cmdType, options, err)
	}
	return responses, err
}

type BatchResult struct {
	Errors []error
}

type Option struct {
	Flags               int32
	AddableIncreaseFlag byte
	TimeOut             time.Duration
	Versions            []int32
	VersionPolicy       byte
	ResultFlag          byte
	Condition           string
	FieldNames          []string
	IndexKeys           []string
	BatchResult         *BatchResult
	ListShiftFlag       byte
	Indexs              []int32
	Async               bool
}

func addFailCount(opName string, req []proto.Message, err error) {
	if err == nil || req == nil {
		return
	}
	tblName := meta.GetTableName(req[0])
	// metrics
	logger.Warnf("tcaplus op metrics failing: table:%s, opName:%s, err:%v", tblName, opName, err)
}

func addCostTime(opName string, reqs []proto.Message, startTime time.Time, err error) {
	if len(reqs) == 0 || reqs[0] == nil {
		return
	}
	tblName := meta.GetTableName(reqs[0])
	result := "0"
	if err != nil {
		errcode, ok := err.(*terror.ErrorCode)
		if !ok {
			result = "unknown"
		} else {
			result = strconv.Itoa(errcode.Code)
		}
	}
	// 用来进行统计处理
	costTime := time.Since(startTime)
	logger.Debugf("tcaplus op metrics time-spent: table:%s, opName:%s, result:%s, costTime:%s",
		tblName, opName, result, costTime)
}

func wrapTcaplusError(err error) error {
	if err == nil {
		return nil
	}
	var errCode *terror.ErrorCode
	ok := errors.As(err, &errCode)
	if !ok {
		return fmt.Errorf("%v. cause: tcaplus operate failed: %s", orm.CodeErrSystem, err)
	}
	switch errCode.Code {
	case terror.TXHDB_ERR_RECORD_NOT_EXIST:
		return orm.New(orm.CodeErrTcaplusRecordNotExist, errCode)
	case terror.SVR_ERR_FAIL_INVALID_VERSION:
		return orm.New(orm.CodeErrTcaplusInvalidVersion, errCode)
	case terror.SVR_ERR_FAIL_RECORD_EXIST:
		return orm.New(orm.CodeErrTcaplusInsertRecordExist, errCode)
	case terror.COMMON_ERR_CONDITION_NOT_MATCHED:
		return orm.New(orm.CodeErrTcaplusConditionNotMatched, errCode)
	case terror.API_ERR_UNPACK_MESSAGE:
		// Note: 当用replace接口时，如果出现版本号错误时，应该报错SVR_ERR_FAIL_INVALID_VERSION
		// 但是tcaplus的API目前却返回API_ERR_UNPACK_MESSAGE错误
		// 当前先公开错误码，供业务侧决策如何处理
		// DeleteByPartKey 接口成功删除数据时也返回这个错误
		return orm.New(orm.CodeErrTcaplusUnpackMessageFailed, errCode)
	case terror.SVR_ERR_FAIL_TIMEOUT,
		terror.PROXY_ERR_SWIFT_TIMEOUT,
		terror.TCAPDB_ERR_TIMEOUT,
		terror.PROXY_ERR_ALREADY_CACHED_REQUEST_TIMEOUT,
		terror.TXHDB_ERR_MUTEX_TIMEDLOCK_TIMEOUT,
		terror.API_ERR_DIR_GET_PROXYLIST_TIMEOUT,
		terror.API_ERR_WAIT_RSP_TIMEOUT,
		terror.PROXY_ERR_PROBE_TIMEOUT,
		terror.PROXY_ERR_QUERY_FROM_INDEX_SERVER_TIMEOUT:
		return orm.New(orm.CodeErrTcaplusDbopTimeout, errCode)
	default:
		return orm.Errorf(orm.CodeErrSystem, "tcaplus operate failed: %s", err)
	}
}

func mergeResponses(req proto.Message, resp response.TcaplusResponse) ([]*Response, error) {
	if ret := resp.GetResult(); ret != 0 {
		return nil, errors.New(fmt.Sprint(ret))
	}
	responses := make([]*Response, 0)
	if ret := resp.GetResult(); ret != 0 {
		return nil, errors.New(fmt.Sprint(ret))
	}
	for i := 0; i < resp.GetRecordCount(); i++ {
		rec, err := resp.FetchRecord()
		if err != nil {
			return nil, wrapTcaplusError(err)
		}
		v := deepcopy.Copy(req)
		msg, _ := v.(proto.Message)
		err = rec.GetPBData(msg)
		if err != nil {
			return nil, wrapTcaplusError(err)
		}
		responses = append(responses, &Response{
			Message: msg,
			Version: rec.GetVersion(),
			Index:   rec.GetIndex(),
		})
	}
	return responses, nil
}

func (t *Driver) internalOperate(cmdType int, requests []proto.Message, opts *Option) ([]*Response, error) {
	pbOpt := &option.PBOpt{
		Flags:               opts.Flags,
		AddableIncreaseFlag: opts.AddableIncreaseFlag,
		Timeout:             opts.TimeOut,
	}

	var req proto.Message
	if len(requests) > 0 {
		req = requests[0]
	}
	tbName := meta.GetTableName(req)
	var version int32
	if len(opts.Versions) > 0 {
		version = opts.Versions[0]
	}
	startTime := time.Now()
	switch cmdType {
	case cmd.TcaplusApiGetReq:
		err := t.Cli.DoGet(req, pbOpt)
		addCostTime("Get", requests, startTime, err)
		if err != nil {
			addFailCount("Get", requests, err)
			return nil, wrapTcaplusError(err)
		}
		return []*Response{{Message: req, Version: pbOpt.Version}}, nil
	case cmd.TcaplusApiReplaceReq:
		pbOpt.VersionPolicy = opts.VersionPolicy
		pbOpt.Version = version
		// Note: ResultFlag暂时不要设置，tcaplus那边有问题，如果设置了，当前版本号校验错误时
		// 却返回 API_ERR_UNPACK_MESSAGE, 等待tcaplus团队优化
		// pbOpt.ResultFlag = opts.resultFlag
		pbOpt.ResultFlagForFail = option.TcaplusResultFlagAllOldValue
		pbOpt.ResultFlagForSuccess = option.TcaplusResultFlagAllNewValue
		pbOpt.Condition = opts.Condition
		err := t.Cli.DoReplace(req, pbOpt)
		addCostTime("Replace", requests, startTime, err)
		if err != nil {
			addFailCount("Replace", requests, err)
			return nil, wrapTcaplusError(err)
		}
		return []*Response{{Message: req, Version: pbOpt.Version}}, nil
	case cmd.TcaplusApiUpdateReq:
		pbOpt.VersionPolicy = opts.VersionPolicy
		pbOpt.Version = version
		pbOpt.ResultFlag = opts.ResultFlag
		pbOpt.Condition = opts.Condition
		err := t.Cli.DoUpdate(req, pbOpt)
		addCostTime("Update", requests, startTime, err)
		if err != nil {
			addFailCount("Update", requests, err)
			return nil, wrapTcaplusError(err)
		}
		return []*Response{{Message: req, Version: pbOpt.Version}}, nil
	case cmd.TcaplusApiDeleteReq:
		pbOpt.VersionPolicy = opts.VersionPolicy
		pbOpt.Version = version
		pbOpt.ResultFlag = opts.ResultFlag
		err := t.Cli.DoDelete(req, pbOpt)
		addCostTime("Delete", requests, startTime, err)
		if err != nil {
			addFailCount("Delete", requests, err)
			return nil, wrapTcaplusError(err)
		}
		return []*Response{{Message: req}}, nil
		// PBField
	case cmd.TcaplusApiPBFieldGetReq:
		pbOpt.FieldNames = opts.FieldNames
		err := t.Cli.DoFieldGet(req, pbOpt)
		addCostTime("FieldGet", requests, startTime, err)
		if err != nil {
			addFailCount("FieldGet", requests, err)
			return nil, wrapTcaplusError(err)
		}
		return []*Response{{Message: req}}, nil
	case cmd.TcaplusApiPBFieldUpdateReq:
		pbOpt.VersionPolicy = opts.VersionPolicy
		pbOpt.Version = version
		pbOpt.FieldNames = opts.FieldNames
		pbOpt.Condition = opts.Condition
		err := t.Cli.DoFieldUpdate(req, pbOpt)
		addCostTime("FieldUpdate", requests, startTime, err)
		if err != nil {
			addFailCount("FieldUpdate", requests, err)
			return nil, wrapTcaplusError(err)
		}
		return []*Response{{Message: req, Version: pbOpt.Version}}, nil
	case cmd.TcaplusApiPBFieldIncreaseReq:
		pbOpt.FieldNames = opts.FieldNames
		err := t.Cli.DoFieldIncrease(req, pbOpt)
		addCostTime("FieldIncrease", requests, startTime, err)
		if err != nil {
			addFailCount("FieldIncrease", requests, err)
			return nil, wrapTcaplusError(err)
		}
		return []*Response{{Message: req, Version: pbOpt.Version}}, nil
		// PBFiled End
	case cmd.TcaplusApiInsertReq:
		pbOpt.ResultFlag = opts.ResultFlag
		err := t.Cli.DoInsert(req, pbOpt)
		addCostTime("Insert", requests, startTime, err)
		if err != nil {
			addFailCount("Insert", requests, err)
			return nil, wrapTcaplusError(err)
		}
		return []*Response{{Message: req, Version: pbOpt.Version}}, nil
	case cmd.TcaplusApiGetByPartkeyReq:
		pbOpt.FieldNames = opts.FieldNames
		records, err := t.Cli.DoGetByPartKey(req, opts.IndexKeys, pbOpt)
		addCostTime("GetByPartKey", requests, startTime, err)
		if err != nil {
			addFailCount("GetByPartKey", requests, err)
			return nil, wrapTcaplusError(err)
		}
		if len(records) != len(pbOpt.BatchVersion) {
			return nil, errors.New("tcaplus batch get failed")
		}
		responses := make([]*Response, len(pbOpt.BatchVersion))
		for i, rec := range records {
			responses[i] = &Response{
				Message: rec,
				Version: pbOpt.BatchVersion[i],
			}
		}
		return responses, nil
	case cmd.TcaplusApiDeleteByPartkeyReq:
		records, err := t.Cli.DoDeleteByPartKey(req, opts.IndexKeys, pbOpt)
		addCostTime("DeleteByPartKey", requests, startTime, err)
		if err != nil {
			addFailCount("DeleteByPartKey", requests, err)
			return nil, wrapTcaplusError(err)
		}
		if len(records) != len(pbOpt.BatchVersion) {
			return nil, errors.New("tcaplus delete by part key failed")
		}
		responses := make([]*Response, len(pbOpt.BatchVersion))
		for i, rec := range records {
			responses[i] = &Response{
				Message: rec,
				Version: pbOpt.BatchVersion[i],
			}
		}
		return responses, nil
		// Batch
	case cmd.TcaplusApiBatchGetReq:
		err := t.Cli.DoBatchGet(requests, pbOpt)
		addCostTime("BatchGet", requests, startTime, err)
		if len(requests) != len(pbOpt.BatchVersion) || len(requests) != len(pbOpt.BatchResult) {
			return nil, errors.New("tcaplus batch get failed")
		}
		responses := make([]*Response, len(requests))
		for i, err := range pbOpt.BatchResult {
			responses[i] = &Response{
				Message: requests[i],
				Version: pbOpt.BatchVersion[i],
			}

			if opts.BatchResult != nil {
				opts.BatchResult.Errors = append(opts.BatchResult.Errors, wrapTcaplusError(err))
			}
		}
		if err != nil {
			addFailCount("BatchGet", requests, err)
			tcaplusErr := err.(*terror.ErrorCode)
			if tcaplusErr.Code == terror.TXHDB_ERR_RECORD_NOT_EXIST {
				err = nil
			}
		}
		return responses, err
		// cmd.TcaplusApiPBBatchFieldGetReg:
	case cmd.TcaplusApiBatchInsertReq:
		err := t.Cli.DoBatchInsert(requests, pbOpt)
		addCostTime("BatchInsert", requests, startTime, err)
		if len(requests) != len(pbOpt.BatchResult) || len(requests) != len(pbOpt.BatchVersion) {
			return nil, errors.New("tcaplus batch insert failed")
		}
		responses := make([]*Response, len(requests))
		for i, err := range pbOpt.BatchResult {
			responses[i] = &Response{
				Message: requests[i],
				Version: pbOpt.BatchVersion[i],
			}
			if opts.BatchResult != nil {
				opts.BatchResult.Errors = append(opts.BatchResult.Errors, wrapTcaplusError(err))
			}
		}
		if err != nil {
			addFailCount("BatchInsert", requests, err)
			tcaplusErr := err.(*terror.ErrorCode)
			if tcaplusErr.Code == terror.TXHDB_ERR_RECORD_NOT_EXIST {
				err = nil
			}
		}

		return responses, err
	case cmd.TcaplusApiBatchDeleteReq:
		err := t.Cli.DoBatchDelete(requests, pbOpt)
		addCostTime("BatchDelete", requests, startTime, err)
		if len(requests) != len(pbOpt.BatchResult) || len(requests) != len(pbOpt.BatchVersion) {
			return nil, errors.New("tcaplus batch delete failed")
		}
		responses := make([]*Response, len(requests))
		for i, err := range pbOpt.BatchResult {
			responses[i] = &Response{
				Message: requests[i],
				Version: pbOpt.BatchVersion[i],
			}
			if opts.BatchResult != nil {
				opts.BatchResult.Errors = append(opts.BatchResult.Errors, err)
			}
		}
		if err != nil {
			addFailCount("BatchDelete", requests, err)
			tcaplusErr := err.(*terror.ErrorCode)
			if tcaplusErr.Code == terror.TXHDB_ERR_RECORD_NOT_EXIST {
				err = nil
			}
		}
		return responses, err
	case cmd.TcaplusApiListAddAfterReq:
		pbOpt.VersionPolicy = opts.VersionPolicy
		pbOpt.Version = version
		pbOpt.ResultFlag = opts.ResultFlag
		pbOpt.ListShiftFlag = opts.ListShiftFlag
		index := int32(-1) // default: append to tail
		if len(opts.Indexs) > 0 {
			index = opts.Indexs[0]
		}
		index, err := t.Cli.DoListAddAfter(req, index, pbOpt)
		addCostTime("ListInert", requests, startTime, err)
		if err != nil {
			addFailCount("ListInert", requests, err)
			return nil, wrapTcaplusError(err)
		}
		return []*Response{{Message: req, Version: pbOpt.Version, Index: index}}, nil
	case cmd.TcaplusApiListAddAfterBatchReq:
		pbOpt.VersionPolicy = opts.VersionPolicy
		pbOpt.Version = version
		pbOpt.ResultFlag = opts.ResultFlag
		pbOpt.ListShiftFlag = opts.ListShiftFlag
		indexes := make([]int32, len(requests))
		for i := range requests {
			indexes[i] = opts.Indexs[0]
		}
		err := t.Cli.DoListAddAfterBatch(requests, indexes, pbOpt)
		addCostTime("BatchListInert", requests, startTime, err)
		if err != nil {
			addFailCount("BatchListInert", requests, err)
		}
		responses := make([]*Response, len(requests))
		//for i, err := range pbOpt.BatchResult {
		//	responses[i] = &Response{
		//		Message: requests[i],
		//		Version: pbOpt.BatchVersion[i],
		//	}
		//	if opts.BatchResult != nil {
		//		opts.BatchResult.Errors = append(opts.BatchResult.Errors, err)
		//	}
		//}
		return responses, nil
	case cmd.TcaplusApiListGetReq:
		if len(opts.Indexs) == 0 {
			return nil, errors.New("indexes is empty")
		}
		err := t.Cli.DoListGet(req, opts.Indexs[0], pbOpt)
		addCostTime("ListGet", requests, startTime, err)
		if err != nil {
			addFailCount("ListGet", requests, err)
			return nil, wrapTcaplusError(err)
		}
		return []*Response{{Message: req, Version: pbOpt.Version}}, nil

	case cmd.TcaplusApiListGetAllReq:
		records, err := t.Cli.DoListGetAll(req, pbOpt)
		addCostTime("ListGetAll", requests, startTime, err)
		if err != nil {
			addFailCount("ListGetAll", requests, err)
			return nil, wrapTcaplusError(err)
		}
		responses := make([]*Response, len(records))
		i := 0
		for index, rec := range records {
			responses[i] = &Response{
				Message: rec,
				Version: pbOpt.Version,
				Index:   int32(index),
			}
			i++
		}
		return responses, nil
	case cmd.TcaplusApiListDeleteReq:
		pbOpt.VersionPolicy = opts.VersionPolicy
		pbOpt.Version = version
		pbOpt.ResultFlag = opts.ResultFlag
		if len(opts.Indexs) == 0 {
			return nil, errors.New("indexs is empty")
		}
		err := t.Cli.DoListDelete(req, opts.Indexs[0], pbOpt)
		addCostTime("ListDelete", requests, startTime, err)
		if err != nil {
			addFailCount("ListDelete", requests, err)
			return nil, wrapTcaplusError(err)
		}
		return []*Response{{Message: req, Version: pbOpt.Version}}, nil
	case cmd.TcaplusApiListDeleteBatchReq:
		pbOpt.VersionPolicy = opts.VersionPolicy
		pbOpt.Version = version
		pbOpt.ResultFlag = opts.ResultFlag
		records, err := t.Cli.DoListDeleteBatch(req, opts.Indexs, pbOpt)
		addCostTime("ListDeleteBatch", requests, startTime, err)
		if err != nil {
			addFailCount("ListDeleteBatch", requests, err)
			return nil, wrapTcaplusError(err)
		}
		responses := make([]*Response, len(records))
		for i, record := range records {
			responses[i] = &Response{
				Message: record,
			}
		}
		return responses, nil
	case cmd.TcaplusApiListDeleteAllReq:
		pbOpt.VersionPolicy = opts.VersionPolicy
		pbOpt.Version = version
		err := t.Cli.DoListDeleteAll(req, pbOpt)
		addCostTime("ListDeleteAll", requests, startTime, err)
		if err != nil {
			addFailCount("ListDeleteAll", requests, err)
			return nil, wrapTcaplusError(err)
		}
		return []*Response{{Message: req, Version: pbOpt.Version}}, nil
	case cmd.TcaplusApiTableTraverseReq:
		pbOpt.Condition = opts.Condition
		tra := t.Cli.GetTraverser(t.ZoneId, tbName)
		if tra == nil {
			return nil, fmt.Errorf("no Traverse found for %s", tbName)
		}
		err := tra.SetFieldNames(opts.FieldNames)
		if err != nil {
			addFailCount("Travers", requests, err)
			return nil, wrapTcaplusError(err)
		}

		if opts.Async {
			aid := atomic.AddUint64(&t.asyncId, 1)
			if err := tra.SetAsyncId(aid); err != nil {
				logger.Errorf("failed to set async id %d: %v", aid, err)
			}
			if err := tra.Start(); err != nil {
				addFailCount("Travers", requests, err)
				return nil, wrapTcaplusError(err)
			}
			respCh := make(chan response.TcaplusResponse, 1000)
			go func() {
				defer func() {
					err := tra.Stop()
					if err != nil {
						logger.Errorf("failed to stop Traverse: %v", err)
					}
				}()
				for {
					resp, err := t.Cli.RecvResponse()
					if err != nil {
						logger.Errorf("RecvResponse failed: %v", err)
						continue
					} else {
						if resp != nil && resp.GetAsyncId() == aid {
							respCh <- resp
							if tra.State() != traverser.TraverseStateNormal {
								close(respCh)
								return
							}
						} else {
							time.Sleep(5 * time.Microsecond)
						}
					}
				}
			}()
			return []*Response{{RespCh: respCh}}, nil
		} else {
			defer func() {
				err := tra.Stop()
				if err != nil {
					logger.Errorf("failed to stop Traverse: %v", err)
				}
			}()
			resps, err := t.Cli.DoTraverse(tra, 5*time.Minute)
			if err != nil {
				return nil, wrapTcaplusError(err)
			}
			var response []*Response
			for _, resp := range resps {
				v, err := mergeResponses(req, resp)
				if err != nil {
					logger.Errorf("failed to gather responses: %v", err)
					continue
				}
				response = append(response, v...)
			}
			return response, nil
		}

	default:
		return nil, errors.New(fmt.Sprintf("not supported tcaplus cmd type:%d", cmdType))
	}
}

// SingleGet 查询记录，记录不存在时返回错误 RECORD_NOT_EXISTS
func (t *Driver) SingleGet(request proto.Message, options ...Option) (*Response, error) {
	requests := []proto.Message{request}
	responses, err := t.Operate(cmd.TcaplusApiGetReq, requests, options...)
	if err != nil {
		return nil, err
	}
	return responses[0], nil
}

// Replace 替换记录，记录不存在时插入，存在时更新
func (t *Driver) Replace(request proto.Message, options ...Option) (*Response, error) {
	requests := []proto.Message{request}
	responses, err := t.Operate(cmd.TcaplusApiReplaceReq, requests, options...)
	if err != nil {
		return nil, err
	}
	return responses[0], nil
}

// Update 更新记录，记录不存在时返回错误
func (t *Driver) Update(request proto.Message, options ...Option) (*Response, error) {
	requests := []proto.Message{request}
	responses, err := t.Operate(cmd.TcaplusApiUpdateReq, requests, options...)
	if err != nil {
		return nil, err
	}
	return responses[0], nil
}

// SingleDelete 删除记录， 记录不存在时返回错误
func (t *Driver) SingleDelete(request proto.Message, options ...Option) (*Response, error) {
	requests := []proto.Message{request}
	responses, err := t.Operate(cmd.TcaplusApiDeleteReq, requests, options...)
	if err != nil {
		return nil, err
	}
	return responses[0], nil
}

// Insert 插入记录，记录存在时返回错误
func (t *Driver) Insert(request proto.Message, options ...Option) (*Response, error) {
	requests := []proto.Message{request}
	responses, err := t.Operate(cmd.TcaplusApiInsertReq, requests, options...)
	if err != nil {
		return nil, err
	}
	return responses[0], nil
}

// FieldGet 查询指定字段的值，字段通过FiledName选项设置
func (t *Driver) FieldGet(request proto.Message, options ...Option) (*Response, error) {
	requests := []proto.Message{request}
	responses, err := t.Operate(cmd.TcaplusApiPBFieldGetReq, requests, options...)
	if err != nil {
		return nil, err
	}
	return responses[0], nil
}

// FieldUpdate 更新指定字段的值，字段通过FiledName选项设置
func (t *Driver) FieldUpdate(request proto.Message, options ...Option) (*Response, error) {
	requests := []proto.Message{request}
	responses, err := t.Operate(cmd.TcaplusApiPBFieldUpdateReq, requests, options...)
	if err != nil {
		return nil, err
	}
	return responses[0], nil
}

// FieldIncrease 对指定字段自增，字段通过FiledName选项设置
func (t *Driver) FieldIncrease(request proto.Message, options ...Option) (*Response, error) {
	requests := []proto.Message{request}
	responses, err := t.Operate(cmd.TcaplusApiPBFieldIncreaseReq, requests, options...)
	if err != nil {
		return nil, err
	}
	return responses[0], nil
}

// BatchGet 对同一个表进行批量查询
func (t *Driver) BatchGet(requests []proto.Message, options ...Option) ([]*Response, error) {
	return t.Operate(cmd.TcaplusApiBatchGetReq, requests, options...)
}

//// BatchFieldGet 对同一个表进行批量查询
//func (t *Driver) BatchFieldGet(requests []proto.Message, options ...Option) ([]*Response, error) {
//	// return t.Operate(cmd.TcaplusApiPBBatchFieldGetReq, requests, options...)
//	return nil, nil
//}

// BatchInsert 对同一个表进行批量插入
func (t *Driver) BatchInsert(requests []proto.Message, options ...Option) ([]*Response, error) {
	return t.Operate(cmd.TcaplusApiBatchInsertReq, requests, options...)
}

// BatchDelete 对同一个表进行批量删除
func (t *Driver) BatchDelete(requests []proto.Message, options ...Option) ([]*Response, error) {
	return t.Operate(cmd.TcaplusApiBatchDeleteReq, requests, options...)
}

// GetByPartKey 根据表的部分Key字段查询
func (t *Driver) GetByPartKey(request proto.Message, options ...Option) ([]*Response, error) {
	requests := []proto.Message{request}
	return t.Operate(cmd.TcaplusApiGetByPartkeyReq, requests, options...)
}

// DeleteByPartKey 根据表的部分Key字段删除 成功删除时: API_ERR_UNPACK_MESSAGE (-2579)
func (t *Driver) DeleteByPartKey(request proto.Message, options ...Option) ([]*Response, error) {
	requests := []proto.Message{request}
	return t.Operate(cmd.TcaplusApiDeleteByPartkeyReq, requests, options...)
}

// ListGet 查询List单个表元素
func (t *Driver) ListGet(request proto.Message, options ...Option) (*Response, error) {
	requests := []proto.Message{request}
	responses, err := t.Operate(cmd.TcaplusApiListGetReq, requests, options...)
	if err != nil {
		return nil, err
	}
	return responses[0], nil
}

// ListAppend 在List表后面追加元素 插入元素位置在最后面
func (t *Driver) ListAppend(request proto.Message, options ...Option) (*Response, error) {
	requests := []proto.Message{request}
	// tcaplus_protocol_cs.TCAPLUS_LIST_LAST_INDEX = -1 插入元素位置在最后面
	// tcaplus_protocol_cs.TCAPLUS_LIST_PRE_FIRST_INDEX = -2 插入元素位置在最前面
	options = append(options, Index(-1))
	responses, err := t.Operate(cmd.TcaplusApiListAddAfterReq, requests, options...)
	if err != nil {
		return nil, err
	}
	return responses[0], nil
}

// ListPrepend 在List表后面追加元素  插入元素位置在最前面
func (t *Driver) ListPrepend(request proto.Message, options ...Option) (*Response, error) {
	requests := []proto.Message{request}
	// tcaplus_protocol_cs.TCAPLUS_LIST_LAST_INDEX = -1 插入元素位置在最后面
	// tcaplus_protocol_cs.TCAPLUS_LIST_PRE_FIRST_INDEX = -2 插入元素位置在最前面
	options = append(options, Index(-2))
	responses, err := t.Operate(cmd.TcaplusApiListAddAfterReq, requests, options...)
	if err != nil {
		return nil, err
	}
	return responses[0], nil
}

// ListAppendBatch 批量追加元素
func (t *Driver) ListAppendBatch(requests []proto.Message, options ...Option) ([]*Response, error) {
	for i := 0; i < len(requests); i++ {
		options = append(options, Index(-1))
	}
	responses, err := t.Operate(cmd.TcaplusApiListAddAfterBatchReq, requests, options...)
	if err != nil {
		return nil, err
	}
	return responses, nil
}

// ListGetAll 查询list所有元素
func (t *Driver) ListGetAll(request proto.Message, options ...Option) ([]*Response, error) {
	requests := []proto.Message{request}
	return t.Operate(cmd.TcaplusApiListGetAllReq, requests, options...)
}

// ListDelete 删除List表单个元素
func (t *Driver) ListDelete(request proto.Message, options ...Option) (*Response, error) {
	requests := []proto.Message{request}
	responses, err := t.Operate(cmd.TcaplusApiListDeleteReq, requests, options...)
	if err != nil {
		return nil, err
	}
	return responses[0], nil
}

// ListDeleteBatch 删除List表多个元素 （成功删除时,返回结果为空）
func (t *Driver) ListDeleteBatch(request proto.Message, options ...Option) ([]*Response, error) {
	requests := []proto.Message{request}
	return t.Operate(cmd.TcaplusApiListDeleteBatchReq, requests, options...)
}

// ListDeleteAll 删除List表所有元素
func (t *Driver) ListDeleteAll(request proto.Message, options ...Option) ([]*Response, error) {
	requests := []proto.Message{request}
	return t.Operate(cmd.TcaplusApiListDeleteAllReq, requests, options...)
}

// Traverse 遍历表
func (t *Driver) Traverse(request proto.Message, options ...Option) ([]*Response, error) {
	requests := []proto.Message{request}
	return t.Operate(cmd.TcaplusApiTableTraverseReq, requests, options...)
}

func Index(index int32) Option {
	return Option{
		Indexs: []int32{index},
	}
}

func IndexKeys(indexNames ...string) Option {
	return Option{
		IndexKeys: indexNames,
	}
}
