package main

// FieldMapping 字段映射
type FieldMapping struct {
	MainField string // 主表字段名
	RefField  string // 参考表字段名
	IsKey     bool   // 是否为主键
}

// Config 配置信息
type Config struct {
	MainSheet     string         // 主表sheet名
	RefSheet      string         // 参考表sheet名
	HeaderRow     int            // 表头行号（Excel行号，从1开始）
	RefHeaderRow  int            // 参考表表头行号
	FieldMappings []FieldMapping // 所有字段映射
	MainKeys      []string       // 主表主键字段
	RefKeys       []string       // 参考表主键字段
}

// Row 数据行
type Row map[string]string

// MatchResult 匹配结果
type MatchResult struct {
	Matched  []MatchedPair // 匹配的行
	OnlyMain []Row         // 仅主表有
	OnlyRef  []Row         // 仅参考表有
}

// MatchedPair 匹配的行对
type MatchedPair struct {
	MainRow Row
	RefRow  Row
	Key     string
}

// DiffAnalysis 差异分析结果
type DiffAnalysis struct {
	Field       string  // 字段名
	Value       string  // 取值
	OnlyMainPct float64 // 在仅主表有中的占比
	MatchedPct  float64 // 在匹配行中的占比
	Impact      float64 // 影响度
	Type        string  // "only_main" 或 "only_ref"
}

// Rule 过滤规则
type Rule struct {
	Field    string   // 字段名
	Action   string   // 动作：不等于/等于/不包含/包含/开头不是/开头是...
	Pattern  string   // 模式值
	Values   []string // 所有过滤值
	RuleType string   // "exclusive"(排除) 或 "inclusive"(包含)
}
