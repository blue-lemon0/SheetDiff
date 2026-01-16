package main

// Config 配置信息
type Config struct {
	MainSheet string   // 主表sheet名
	RefSheet  string   // 参考表sheet名
	MainKeys  []string // 主表主键字段
	RefKeys   []string // 参考表主键字段
	MainSkip  int      // 主表跳过行数
	RefSkip   int      // 参考表跳过行数
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
