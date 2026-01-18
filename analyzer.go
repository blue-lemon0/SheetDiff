package main

import (
	"sort"
	"strings"
)

// MatchByKeys 主键匹配
func MatchByKeys(mainData, refData []Row, config Config) MatchResult {
	result := MatchResult{
		Matched:  make([]MatchedPair, 0),
		OnlyMain: make([]Row, 0),
		OnlyRef:  make([]Row, 0),
	}

	// 构建主表的主键映射
	mainMap := make(map[string]Row)
	for _, row := range mainData {
		key := buildRowKey(row, config.MainKeys)
		if key != "" {
			mainMap[key] = row
		}
	}

	// 构建参考表的主键映射
	refMap := make(map[string]Row)
	for _, row := range refData {
		key := buildRowKey(row, config.RefKeys)
		if key != "" {
			refMap[key] = row
		}
	}

	// 找出匹配的行
	for key, mainRow := range mainMap {
		if refRow, exists := refMap[key]; exists {
			result.Matched = append(result.Matched, MatchedPair{
				MainRow: mainRow,
				RefRow:  refRow,
				Key:     key,
			})
			delete(refMap, key) // 从参考表map中移除，剩下的就是仅参考表有的
		} else {
			result.OnlyMain = append(result.OnlyMain, mainRow)
		}
	}

	// 剩下的参考表行就是仅参考表有的
	for _, refRow := range refMap {
		result.OnlyRef = append(result.OnlyRef, refRow)
	}

	return result
}

// AnalyzeDifferences 分析差异原因
func AnalyzeDifferences(result MatchResult, config Config) ([]Rule, []Rule, []Rule, []Rule) {
	var mainOnlyRules, mainCommonRules, refOnlyRules, refCommonRules []Rule

	// 提取数据
	mainCommonRows := extractCommonRows(result.Matched, true)
	refCommonRows := extractCommonRows(result.Matched, false)

	// 1. 分析主表独有行特征
	if len(result.OnlyMain) > 0 && len(mainCommonRows) > 0 {
		mainOnlyRules = findPureFeatures(result.OnlyMain, mainCommonRows, config.MainKeys)
		setRuleType(mainOnlyRules, "主表独有")
	}

	// 2. 分析主表共有行特征
	if len(mainCommonRows) > 0 && len(result.OnlyMain) > 0 {
		mainCommonRules = findPureFeatures(mainCommonRows, result.OnlyMain, config.MainKeys)
		setRuleType(mainCommonRules, "主表共有")
	}

	// 3. 分析参考表独有行特征
	if len(result.OnlyRef) > 0 && len(refCommonRows) > 0 {
		refOnlyRules = findPureFeatures(result.OnlyRef, refCommonRows, config.RefKeys)
		setRuleType(refOnlyRules, "参考表独有")
	}

	// 4. 分析参考表共有行特征
	if len(refCommonRows) > 0 && len(result.OnlyRef) > 0 {
		refCommonRules = findPureFeatures(refCommonRows, result.OnlyRef, config.RefKeys)
		setRuleType(refCommonRules, "参考表共有")
	}

	return mainOnlyRules, mainCommonRules, refOnlyRules, refCommonRules
}

// setRuleType 设置规则类型
func setRuleType(rules []Rule, ruleType string) {
	for i := range rules {
		rules[i].RuleType = ruleType
	}
}

// extractCommonRows 从匹配行中提取共有行
func extractCommonRows(matched []MatchedPair, fromMain bool) []Row {
	var rows []Row
	for _, pair := range matched {
		if fromMain {
			rows = append(rows, pair.MainRow)
		} else {
			rows = append(rows, pair.RefRow)
		}
	}
	return rows
}

// buildRowKey 构建行的主键字符串
func buildRowKey(row Row, keyFields []string) string {
	parts := make([]string, 0, len(keyFields))
	for _, field := range keyFields {
		if value, exists := row[field]; exists {
			parts = append(parts, strings.TrimSpace(value))
		} else {
			parts = append(parts, "") // 字段不存在时用空值
		}
	}
	return strings.Join(parts, "|")
}

// AnalyzeFilterFields 分析可能的过滤字段
func AnalyzeFilterFields(mainData, refData []Row, mainHeaders, refHeaders []string,
	mainKeyFields, refKeyFields []string,
	mappings []FieldMapping) ([]FilterFieldInfo, []FilterFieldInfo) {

	// 分析主表可能的过滤字段
	mainFields := analyzeTableFields(mainData, mainHeaders, mainKeyFields, false, mappings)

	// 分析参考表可能的过滤字段
	refFields := analyzeTableFields(refData, refHeaders, refKeyFields, true, mappings)

	return mainFields, refFields
}

// analyzeTableFields 分析表的字段
func analyzeTableFields(data []Row, headers []string, keyFields []string,
	isRefTable bool, mappings []FieldMapping) []FilterFieldInfo {
	var fields []FilterFieldInfo

	if len(data) == 0 {
		return fields
	}

	// 遍历所有字段
	for _, header := range headers {
		// 跳过主键字段
		isKey := false
		for _, key := range keyFields {
			if key == header {
				isKey = true
				break
			}
		}
		if isKey {
			continue
		}

		info := analyzeSingleField(data, header)

		// 如果是参考表，需要映射字段名
		if isRefTable {
			mappedField := mapFieldToMain(header, mappings)
			if mappedField != "" {
				info.Field = mappedField
			}
		}

		fields = append(fields, info)
	}

	return fields
}

// analyzeSingleField 分析单个字段
func analyzeSingleField(data []Row, field string) FilterFieldInfo {
	info := FilterFieldInfo{
		Field: field,
	}

	// 统计字段取值
	valueCounts := make(map[string]int)
	totalRows := 0

	for _, row := range data {
		totalRows++
		value := ""
		if v, exists := row[field]; exists {
			value = strings.TrimSpace(v)
		}
		valueCounts[value]++
	}

	// 计算取值数量
	info.ValueCount = len(valueCounts)

	// 检查是否有空白值
	_, info.HasEmpty = valueCounts[""]

	// 检查是否所有值都相同
	if info.ValueCount == 0 {
		info.IsConstant = false
	} else if info.ValueCount == 1 {
		// 只有一种取值（可能是空白值或其他值）
		info.IsConstant = true
	} else if info.ValueCount == 2 && info.HasEmpty {
		// 只有空白值和另一个值，也算常量
		info.IsConstant = true
	} else {
		info.IsConstant = false
	}

	// 获取前5个具体取值（排除空白值）
	var uniqueVals []string
	for val := range valueCounts {
		if val != "" {
			uniqueVals = append(uniqueVals, val)
		}
	}
	sort.Strings(uniqueVals)

	// 限制最多显示5个值
	maxShow := 5
	if len(uniqueVals) > maxShow {
		info.UniqueValues = uniqueVals[:maxShow]
		info.UniqueValues = append(info.UniqueValues, "...")
	} else {
		info.UniqueValues = uniqueVals
	}

	// 判断是否可能成为过滤字段
	// 条件1: 不能有空白值
	// 条件2: 不能是常量字段
	// 条件3: 至少要有2个不同的非空值
	nonEmptyValueCount := len(uniqueVals)
	info.CouldBeFilter = !info.HasEmpty && !info.IsConstant && nonEmptyValueCount >= 2

	return info
}

// mapFieldToMain 将参考表字段名映射为主表字段名
func mapFieldToMain(refField string, mappings []FieldMapping) string {
	for _, m := range mappings {
		if m.RefField == refField {
			return m.MainField
		}
	}
	return refField
}
