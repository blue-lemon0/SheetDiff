package main

import (
	"fmt"
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
	var mainExcl, mainIncl, refExcl, refIncl []Rule

	// 分析仅主表有的行（参考表可能的排除规则）
	if len(result.OnlyMain) > 0 {
		// 提取共有行（主表侧）
		commonMainRows := extractCommonRows(result.Matched, true)

		// 找出排除型规则
		mainExcl = findExclusiveRules(result.OnlyMain, commonMainRows, config.MainKeys)

		if len(mainExcl) > 0 {
			fmt.Printf("🔍 发现 %d 个参考表可能的排除规则\n", len(mainExcl))
		}

		// 找出包含型规则（匹配行特征 → 参考表包含规则）
		if len(commonMainRows) > 0 {
			mainIncl = findInclusiveRules(commonMainRows, result.OnlyMain, config.MainKeys)

			if len(mainIncl) > 0 {
				fmt.Printf("🔍 发现 %d 个参考表可能的包含规则\n", len(mainIncl))
			}
		}
	}

	// 分析仅参考表有的行（主表可能的排除规则）
	if len(result.OnlyRef) > 0 {
		// 提取共有行（参考表侧）
		commonRefRows := extractCommonRows(result.Matched, false)

		// 找出排除型规则
		refExcl = findExclusiveRules(result.OnlyRef, commonRefRows, config.RefKeys)

		if len(refExcl) > 0 {
			fmt.Printf("🔍 发现 %d 个主表可能的排除规则\n", len(refExcl))
		}

		// 找出包含型规则（匹配行特征 → 主表包含规则）
		if len(commonRefRows) > 0 {
			refIncl = findInclusiveRules(commonRefRows, result.OnlyRef, config.RefKeys)

			if len(refIncl) > 0 {
				fmt.Printf("🔍 发现 %d 个主表可能的包含规则\n", len(refIncl))
			}
		}
	}

	return mainExcl, mainIncl, refExcl, refIncl
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

// getAllFields 获取所有非主键字段
func getAllFields(row Row, keyFields []string) []string {
	fields := make([]string, 0, len(row))

	// 将主键字段转为map便于查找
	keyMap := make(map[string]bool)
	for _, key := range keyFields {
		keyMap[key] = true
	}

	for field := range row {
		if !keyMap[field] {
			fields = append(fields, field)
		}
	}

	return fields
}

// countFieldValues 统计字段取值频次
func countFieldValues(rows []Row, field string) map[string]int {
	counts := make(map[string]int)
	for _, row := range rows {
		if value, exists := row[field]; exists {
			counts[value]++
		}
	}
	return counts
}

// countFieldValuesInMatched 统计匹配行中字段取值频次
func countFieldValuesInMatched(matched []MatchedPair, field string, fromMain bool) map[string]int {
	counts := make(map[string]int)
	for _, pair := range matched {
		var row Row
		if fromMain {
			row = pair.MainRow
		} else {
			row = pair.RefRow
		}

		if value, exists := row[field]; exists {
			counts[value]++
		}
	}
	return counts
}
