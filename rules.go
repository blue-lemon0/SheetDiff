package main

import (
	"sort"
	"strings"
)

// Rule 过滤规则
type Rule struct {
	Field   string   // 字段名
	Action  string   // 动作：不等于/不包含/开头不是/结尾不是/不在列表中
	Pattern string   // 模式值
	Values  []string // 所有过滤值
}

// findPureRules 找出纯净字段规则
func findPureRules(onlyRows, commonRows []Row, keyFields []string) []Rule {
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
			rule := buildRule(field, onlyVals)
			rules = append(rules, rule)
		}
	}

	return rules
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

// buildRule 构建规则
func buildRule(field string, values map[string]bool) Rule {
	vals := sortValues(values)
	action, pattern := determineActionPattern(vals)

	return Rule{
		Field:   field,
		Action:  action,
		Pattern: pattern,
		Values:  vals,
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
func determineActionPattern(values []string) (action, pattern string) {
	if len(values) == 1 {
		return "不等于", values[0]
	}

	// 检查共同前缀
	if prefix := commonPrefix(values); len(prefix) > 0 {
		return "开头不是", prefix
	}

	// 检查共同后缀
	if suffix := commonSuffix(values); len(suffix) > 0 {
		return "结尾不是", suffix
	}

	// 检查共同子串
	if substr := commonSubstring(values); len(substr) > 0 {
		return "不包含", substr
	}

	// 兜底
	return "不在列表中", "-"
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
