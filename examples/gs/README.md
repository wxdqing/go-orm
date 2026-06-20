# orm/examples/gs

GS 业务示例 + **ORM 四库集成**（MySQL / PostgreSQL / Redis / Mongo）。  
`orm.option.proto` 真源：[protoc-gen-go-orm/options/orm.option.proto](../../../../tools/protoc-gen-go-orm/options/orm.option.proto)（Tcaplus 选项见 tcaplus 生成物，本示例不测）。

## Schema 职责

| Proto | 覆盖选项 |
|-------|----------|
| `schema/version_player.proto` | `table`, `node_type`, **显式 PAYLOAD**, `primary_key`, `version` |
| `schema/tables.proto` | 大嵌套 **PAYLOAD** `Player` + `index` |
| `schema/fields_player.proto` | **FIELDS**, `tags`(blob), `skip_set_default` |
| `schema/game_role.proto` | **FIELDS**, 复合 PK, **composite_index**, embed/blob/json |
| `schema/list.proto` | **PAYLOAD**, 复合 PK, **composite_index** |
| `schema/tables_tags.proto` | `tags`, **oneof_tags**（非表，tag 验证） |

插件侧更全的选项演示见 [protoc-gen-go-orm/examples](../../../../tools/protoc-gen-go-orm/examples/)。

## 生成

```bash
cd orm/examples/gs
bash gen_pb.sh
```

产物：`pbtest/` 业务代码 + `pbtest/internal/{mysql,pgsql,redis,mongo,tcaplus}/`。

FIELDS 表对 Redis/Mongo 会输出 warning，生成物仍为 **KV PAYLOAD**（`kv.tmpl`）。

## 测试

```bash
# 单元（metadata / skip_set_default / tag 形状，不连 DB）
cd orm/examples/gs/logic && go test -run 'TestCoverage|TestFieldsPlayer_Skip' -count=1

# 四库集成（需本地 Docker 线管，见 docs/orm/README.md）
cd orm/examples/gs/logic && go test -tags=db -count=1

# 按 Phase 跑覆盖用例
go test -tags=db -run 'TestIntegration_Fields|TestIntegration_Composite|TestIntegration_PlayerPayload|TestIntegration_SkipDefault|TestIntegration_.*_VersionedPlayer' -count=1
```

## 后端能力矩阵（非 Tcaplus）

| 能力 | MySQL | PG | Redis | Mongo |
|------|-------|-----|-------|-------|
| PAYLOAD CRUD | ✅ | ✅ | ✅ | ✅ |
| FIELDS CRUD | ✅ | ✅ | PAYLOAD 整包 | 同左 |
| `index` + Find | ✅ | ✅ | N/A | N/A |
| composite_index | ✅ | ✅ | N/A | N/A |
| 分表 | ✅ | ✅ | — | — |

清单与用例 ID：[docs/orm/checklist-examples-coverage.md](../../../docs/orm/checklist-examples-coverage.md)
