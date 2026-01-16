package main

import (
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
func AnalyzeDifferences(result MatchResult, mainData, refData []Row, config Config) []DiffAnalysis {
	analysis := make([]DiffAnalysis, 0)

	// 1. 分析仅主表有的行
	if len(result.OnlyMain) > 0 {
		// 获取所有字段名（排除主键字段）
		fields := getAllFields(result.OnlyMain[0], config.MainKeys)

		for _, field := range fields {
			// 统计在仅主表有行中的取值分布
			onlyMainStats := countFieldValues(result.OnlyMain, field)

			// 统计在匹配行中的取值分布
			matchedStats := countFieldValuesInMatched(result.Matched, field, true)

			// 找出差异明显的取值
			for value, onlyMainCount := range onlyMainStats {
				onlyMainPct := float64(onlyMainCount) / float64(len(result.OnlyMain))
				matchedCount := matchedStats[value]
				matchedPct := float64(matchedCount) / float64(len(result.Matched))

				// 如果这个值在仅主表有中很常见，但在匹配中很少见
				if onlyMainPct > 0.5 && matchedPct < 0.1 && onlyMainPct-matchedPct > 0.3 {
					analysis = append(analysis, DiffAnalysis{
						Field:       field,
						Value:       value,
						OnlyMainPct: onlyMainPct,
						MatchedPct:  matchedPct,
						Impact:      onlyMainPct - matchedPct,
						Type:        "only_main",
					})
				}
			}
		}
	}

	// 2. 分析仅参考表有的行
	if len(result.OnlyRef) > 0 {
		// 获取所有字段名（排除主键字段）
		fields := getAllFields(result.OnlyRef[0], config.RefKeys)

		for _, field := range fields {
			// 统计在仅参考表有行中的取值分布
			onlyRefStats := countFieldValues(result.OnlyRef, field)

			// 统计在匹配行中的取值分布
			matchedStats := countFieldValuesInMatched(result.Matched, field, false)

			// 找出差异明显的取值
			for value, onlyRefCount := range onlyRefStats {
				onlyRefPct := float64(onlyRefCount) / float64(len(result.OnlyRef))
				matchedCount := matchedStats[value]
				matchedPct := float64(matchedCount) / float64(len(result.Matched))

				// 如果这个值在仅参考表有中很常见，但在匹配中很少见
				if onlyRefPct > 0.5 && matchedPct < 0.1 && onlyRefPct-matchedPct > 0.3 {
					analysis = append(analysis, DiffAnalysis{
						Field:       field,
						Value:       value,
						OnlyMainPct: onlyRefPct, // 注意：这里复用OnlyMainPct字段
						MatchedPct:  matchedPct,
						Impact:      onlyRefPct - matchedPct,
						Type:        "only_ref",
					})
				}
			}
		}
	}

	return analysis
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
