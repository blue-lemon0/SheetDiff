package main

import (
	"sort"
	"strings"
)

// findPureRules 找出纯净字段规则（排除型）
func findPureRules(onlyRows, commonRows []Row, keyFields []string) []Rule {
	return findExclusiveRules(onlyRows, commonRows, keyFields)
}

// findExclusiveRules 找出排除型规则
func findExclusiveRules(onlyRows, commonRows []Row, keyFields []string) []Rule {
	var rules []Rule

	if len(onlyRows) == 0 {
		return rules
	}

	// 获取所有非主键字段（从第一行取样）
	fields := getNonKeyFields(onlyRows[0], keyFields)

	for _, field := range fields {
		// 获取字段在独有行和共有行中的取值集合
		onlyVals := getFieldValues(onlyRows, field)
		commonVals := getFieldValues(commonRows, field)

		// 检查是否是纯净字段（无交集）
		if isPureField(onlyVals, commonVals) {
			rule := buildRuleWithType(field, onlyVals, "exclusive")
			rules = append(rules, rule)
		}
	}

	return rules
}

// findInclusiveRules 找出包含型规则
func findInclusiveRules(commonRows, onlyRows []Row, keyFields []string) []Rule {
	var rules []Rule

	if len(commonRows) == 0 {
		return rules
	}

	// 获取所有非主键字段（从第一行取样）
	fields := getNonKeyFields(commonRows[0], keyFields)

	// 阈值参数
	minRows := 3                  // 最小行数
	concentrationThreshold := 0.8 // 集中度阈值 80%

	for _, field := range fields {
		// 计算匹配行中字段值的集中度
		valueCounts := getFieldValuesWithCount(commonRows, field)

		// 总行数
		total := len(commonRows)
		if total < minRows {
			continue
		}

		// 找出前几个主要值
		topValues := getTopValues(valueCounts, 3)

		// 检查主要值的集中度
		mainCount := 0
		for _, val := range topValues {
			mainCount += valueCounts[val]
		}

		concentration := float64(mainCount) / float64(total)

		if concentration >= concentrationThreshold {
			// 检查这些值是否在独有行中很少出现
			onlyCounts := getFieldValuesWithCount(onlyRows, field)
			onlyMainCount := 0
			for _, val := range topValues {
				onlyMainCount += onlyCounts[val]
			}

			// 独有行中这些值的占比应该很低
			onlyTotal := len(onlyRows)
			if onlyTotal > 0 {
				onlyPercent := float64(onlyMainCount) / float64(onlyTotal)
				if onlyPercent <= 0.2 { // 在独有行中不超过20%
					// 构建包含型规则
					valsMap := make(map[string]bool)
					for _, val := range topValues {
						valsMap[val] = true
					}
					rule := buildRuleWithType(field, valsMap, "inclusive")
					rules = append(rules, rule)
				}
			}
		}
	}

	return rules
}

// getFieldValuesWithCount 获取字段值及计数
func getFieldValuesWithCount(rows []Row, field string) map[string]int {
	counts := make(map[string]int)
	for _, row := range rows {
		if val, exists := row[field]; exists && val != "" {
			counts[val]++
		}
	}
	return counts
}

// getTopValues 获取前N个最常见的值
func getTopValues(counts map[string]int, n int) []string {
	type kv struct {
		key   string
		value int
	}

	var pairs []kv
	for k, v := range counts {
		pairs = append(pairs, kv{k, v})
	}

	// 按计数降序排序
	sort.Slice(pairs, func(i, j int) bool {
		return pairs[i].value > pairs[j].value
	})

	// 取前N个
	var result []string
	for i := 0; i < n && i < len(pairs); i++ {
		result = append(result, pairs[i].key)
	}

	return result
}

// getNonKeyFields 获取所有非主键字段
func getNonKeyFields(sampleRow Row, keyFields []string) []string {
	var fields []string
	keyMap := make(map[string]bool)

	for _, k := range keyFields {
		keyMap[k] = true
	}

	for field := range sampleRow {
		if !keyMap[field] {
			fields = append(fields, field)
		}
	}

	return fields
}

// getFieldValues 获取字段在所有行中的取值集合
func getFieldValues(rows []Row, field string) map[string]bool {
	values := make(map[string]bool)
	for _, row := range rows {
		if val, exists := row[field]; exists && val != "" {
			values[val] = true
		}
	}
	return values
}

// isPureField 检查字段是否是纯净字段
func isPureField(onlyVals, commonVals map[string]bool) bool {
	for val := range onlyVals {
		if commonVals[val] {
			return false
		}
	}
	return true
}

// buildRuleWithType 构建带类型的规则
func buildRuleWithType(field string, values map[string]bool, ruleType string) Rule {
	vals := sortValues(values)
	action, pattern := determineActionPattern(vals, ruleType)

	return Rule{
		Field:    field,
		Action:   action,
		Pattern:  pattern,
		Values:   vals,
		RuleType: ruleType,
	}
}

// sortValues 对值进行排序
func sortValues(values map[string]bool) []string {
	var vals []string
	for val := range values {
		vals = append(vals, val)
	}
	sort.Strings(vals)
	return vals
}

// determineActionPattern 确定动作和模式
func determineActionPattern(values []string, ruleType string) (action, pattern string) {
	if len(values) == 1 {
		if ruleType == "exclusive" {
			return "不等于", values[0]
		} else {
			return "等于", values[0]
		}
	}

	// 检查共同前缀
	if prefix := commonPrefix(values); len(prefix) > 0 {
		if ruleType == "exclusive" {
			return "开头不是", prefix
		} else {
			return "开头是", prefix
		}
	}

	// 检查共同后缀
	if suffix := commonSuffix(values); len(suffix) > 0 {
		if ruleType == "exclusive" {
			return "结尾不是", suffix
		} else {
			return "结尾是", suffix
		}
	}

	// 检查共同子串
	if substr := commonSubstring(values); len(substr) > 0 {
		if ruleType == "exclusive" {
			return "不包含", substr
		} else {
			return "包含", substr
		}
	}

	// 兜底
	if ruleType == "exclusive" {
		return "不在列表中", "-"
	} else {
		return "在列表中", "-"
	}
}

// commonPrefix 找共同前缀
func commonPrefix(values []string) string {
	if len(values) == 0 {
		return ""
	}

	// 以第一个字符串为基准
	prefix := values[0]
	for i := 1; i < len(values); i++ {
		// 缩短前缀直到匹配
		for !strings.HasPrefix(values[i], prefix) {
			if len(prefix) == 0 {
				return ""
			}
			prefix = prefix[:len(prefix)-1]
		}
	}

	// 如果前缀太短（小于2个字符），不算有效前缀
	if len(prefix) < 2 {
		return ""
	}

	return prefix
}

// commonSuffix 找共同后缀
func commonSuffix(values []string) string {
	if len(values) == 0 {
		return ""
	}

	// 反转字符串找前缀
	reversed := make([]string, len(values))
	for i, v := range values {
		reversed[i] = reverseString(v)
	}

	suffix := commonPrefix(reversed)
	if suffix != "" {
		return reverseString(suffix)
	}

	return ""
}

// commonSubstring 找共同子串（简化版：找最长公共子串中的前几个字符）
func commonSubstring(values []string) string {
	if len(values) < 2 {
		return ""
	}

	// 简化的共同子串查找：找任意两个字符串的共同部分
	base := values[0]
	for i := len(base); i >= 2; i-- {
		for j := 0; j <= len(base)-i; j++ {
			substr := base[j : j+i]

			// 检查是否所有值都包含这个子串
			allContain := true
			for _, v := range values {
				if !strings.Contains(v, substr) {
					allContain = false
					break
				}
			}

			if allContain {
				return substr
			}
		}
	}

	return ""
}

// reverseString 反转字符串
func reverseString(s string) string {
	runes := []rune(s)
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}
	return string(runes)
}
