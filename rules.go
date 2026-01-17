package main

import (
	"sort"
	"strings"
)

// findPureFeatures 找出纯净特征（忽略有空白值的字段）
func findPureFeatures(rowsA, rowsB []Row, keyFields []string) []Rule {
	var rules []Rule

	if len(rowsA) == 0 || len(rowsB) == 0 {
		return rules
	}

	fields := getNonKeyFields(rowsA[0], keyFields)

	for _, field := range fields {
		valsA := getFieldValues(rowsA, field)
		valsB := getFieldValues(rowsB, field)

		// 有空白值就不纯净
		if isPureField(valsA, valsB) && hasNonEmptyValues(valsA) {
			nonEmptyVals := getNonEmptyValues(valsA)
			rule := buildRule(field, nonEmptyVals)
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

// getFieldValues 获取字段取值集合
func getFieldValues(rows []Row, field string) map[string]bool {
	values := make(map[string]bool)
	for _, row := range rows {
		val := ""
		if v, exists := row[field]; exists {
			val = strings.TrimSpace(v)
		}
		values[val] = true
	}
	return values
}

// isPureField 检查字段是否是纯净字段（有空白值就不纯净）
func isPureField(vals1, vals2 map[string]bool) bool {
	// 规则1：任意一边有空白值就不纯净
	if vals1[""] || vals2[""] {
		return false
	}

	// 规则2：值有交集就不纯净
	for val := range vals1 {
		if vals2[val] {
			return false
		}
	}
	return true
}

// hasNonEmptyValues 检查是否有非空白值
func hasNonEmptyValues(values map[string]bool) bool {
	for val := range values {
		if val != "" {
			return true
		}
	}
	return false
}

// getNonEmptyValues 获取非空白值
func getNonEmptyValues(values map[string]bool) map[string]bool {
	result := make(map[string]bool)
	for val := range values {
		if val != "" {
			result[val] = true
		}
	}
	return result
}

// buildRule 构建规则（只处理非空白值）
func buildRule(field string, values map[string]bool) Rule {
	vals := sortValues(values)
	action, pattern := determineAction(vals)

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

// determineAction 确定动作和模式
func determineAction(values []string) (action, pattern string) {
	if len(values) == 0 {
		return "", ""
	}

	if len(values) == 1 {
		return "等于", values[0]
	}

	// 检查共同前缀
	if prefix := commonPrefix(values); len(prefix) > 0 {
		return "开头是", prefix
	}

	// 检查共同后缀
	if suffix := commonSuffix(values); len(suffix) > 0 {
		return "结尾是", suffix
	}

	// 检查共同子串
	if substr := commonSubstring(values); len(substr) > 0 {
		return "包含", substr
	}

	// 兜底
	return "在列表中", "-"
}

// commonPrefix 找共同前缀
func commonPrefix(values []string) string {
	if len(values) == 0 {
		return ""
	}

	prefix := values[0]
	for i := 1; i < len(values); i++ {
		for !strings.HasPrefix(values[i], prefix) {
			if len(prefix) == 0 {
				return ""
			}
			prefix = prefix[:len(prefix)-1]
		}
	}

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

// commonSubstring 找共同子串
func commonSubstring(values []string) string {
	if len(values) < 2 {
		return ""
	}

	base := values[0]
	for i := len(base); i >= 2; i-- {
		for j := 0; j <= len(base)-i; j++ {
			substr := base[j : j+i]

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
