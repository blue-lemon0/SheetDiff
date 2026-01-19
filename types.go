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
