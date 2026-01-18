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
	MainRow   Row
	RefRow    Row
	Key       string
	MainIndex int // 主表中的行索引
	RefIndex  int // 参考表中的行索引
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

// Rule 规则
type Rule struct {
	Field    string   // 字段名
	Action   string   // 动作：等于/开头是/结尾是/包含/在列表中
	Pattern  string   // 模式值
	Values   []string // 所有取值
	RuleType string   // 规则类型："主表独有"/"主表共有"/"参考表独有"/"参考表共有"
}

// FilterFieldInfo 过滤字段信息
type FilterFieldInfo struct {
	Field         string   // 字段名
	ValueCount    int      // 取值数量
	UniqueValues  []string // 具体取值（最多显示5个）
	HasEmpty      bool     // 是否有空白值
	IsConstant    bool     // 是否所有值都相同
	CouldBeFilter bool     // 是否可能成为过滤字段
}
