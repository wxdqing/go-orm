package orm

// ShardMode 分片模式。
type ShardMode string

const (
	ShardModeNone     ShardMode = "none"     // 单库单表（默认）
	ShardModeDatabase ShardMode = "database" // 分库：按分片键路由到不同数据源
	ShardModeTable    ShardMode = "table"    // 分表：按分片键拼接物理表名
)

// GormStartupConf GORM 启动参数（MySQL / PostgreSQL 共用）。
type GormStartupConf struct {
	SkipDefaultTransaction bool              `json:"skip_default_transaction" mapstructure:"skip_default_transaction"`
	PrepareStmt            bool              `json:"prepare_stmt" mapstructure:"prepare_stmt"`
	DisableAutomaticPing   bool              `json:"disable_automatic_ping" mapstructure:"disable_automatic_ping"`
	TableOptions           string            `json:"table_options" mapstructure:"table_options"` // MySQL: ENGINE=InnoDB
	TimeZone               string            `json:"time_zone" mapstructure:"time_zone"`         // PG DSN / MySQL loc
	SSLMode                string            `json:"ssl_mode" mapstructure:"ssl_mode"`           // PostgreSQL
	ExtraDSN               map[string]string `json:"extra_dsn" mapstructure:"extra_dsn"`         // 额外 DSN 查询参数
}

// DefaultGormStartup 返回默认启动参数。
func DefaultGormStartup(driver string) GormStartupConf {
	c := GormStartupConf{
		SkipDefaultTransaction: true,
		PrepareStmt:            true,
		TimeZone:               "Asia/Shanghai",
		SSLMode:                "disable",
	}
	if driver == "mysql" {
		c.TableOptions = "ENGINE=InnoDB"
	}
	return c
}

// SQLShardConf SQL 分库 / 分表配置（MySQL、PostgreSQL 共用结构）。
type SQLShardConf struct {
	Mode     ShardMode           `json:"mode" mapstructure:"mode"`
	KeyField string              `json:"key_field" mapstructure:"key_field"` // 默认分片字段，如 id、user_id
	Policy   string              `json:"policy" mapstructure:"policy"`       // random | round_robin（分库源负载）
	Sources  []DatabaseShardSource `json:"sources" mapstructure:"sources"`   // 分库数据源列表
	Tables   []TableShardRule    `json:"tables" mapstructure:"tables"`       // 按表覆盖分片规则
}

// DatabaseShardSource 分库数据源。
type DatabaseShardSource struct {
	Name     string `json:"name" mapstructure:"name"` // 逻辑名，可用于日志
	Addr     string `json:"addr" mapstructure:"addr"` // MySQL host:port
	Host     string `json:"host" mapstructure:"host"` // PostgreSQL
	Port     string `json:"port" mapstructure:"port"`
	User     string `json:"user" mapstructure:"user"`
	Password string `json:"password" mapstructure:"password"`
	DBName   string `json:"dbname" mapstructure:"dbname"`
}

// TableShardRule 分表规则。
type TableShardRule struct {
	Table        string `json:"table" mapstructure:"table"`                 // 逻辑表名（与 TableName() 一致）
	KeyField     string `json:"key_field" mapstructure:"key_field"`         // 覆盖全局 KeyField
	ShardCount   int    `json:"shard_count" mapstructure:"shard_count"`     // 分表数量，>=2 生效
	SuffixFormat string `json:"suffix_format" mapstructure:"suffix_format"` // 默认 _%d，如 player_3
}

// Normalize 填充默认值并校验。
func (c *SQLShardConf) Normalize() {
	if c.Mode == "" {
		c.Mode = ShardModeNone
	}
	if c.Policy == "" {
		c.Policy = "random"
	}
	for i := range c.Tables {
		if c.Tables[i].SuffixFormat == "" {
			c.Tables[i].SuffixFormat = "_%d"
		}
	}
}

// TableRule 按逻辑表名查找分表规则。
func (c *SQLShardConf) TableRule(table string) *TableShardRule {
	for i := range c.Tables {
		if c.Tables[i].Table == table {
			return &c.Tables[i]
		}
	}
	return nil
}
