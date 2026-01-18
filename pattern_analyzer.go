// pattern_analyzer.go - 所有规律分析在一个文件里
package main

import (
	"sort"
	"strconv"
	"strings"
	"time"
)

// ========== 预处理 ==========

// FieldStats 字段统计
type FieldStats struct {
	Field     string
	ValueRows map[string]*RowSet
	Values    []string
	IsString  bool
	IsNumeric bool
	IsDate    bool
	TotalRows int
}

// preprocessField 预处理字段
func preprocessField(field string, rows []Row) *FieldStats {
	stats := &FieldStats{
		Field:     field,
		ValueRows: make(map[string]*RowSet),
		TotalRows: len(rows),
	}

	// 统计值分布
	valueCount := make(map[string]int)
	for i, row := range rows {
		val := strings.TrimSpace(row[field])
		if val == "" {
			continue
		}
		if set, exists := stats.ValueRows[val]; exists {
			set.Add(i)
		} else {
			set = NewRowSet(len(rows))
			set.Add(i)
			stats.ValueRows[val] = set
		}
		valueCount[val]++
	}

	// 提取所有不同的值
	for val := range stats.ValueRows {
		stats.Values = append(stats.Values, val)
	}

	// 按频率排序
	sort.Slice(stats.Values, func(i, j int) bool {
		return valueCount[stats.Values[i]] > valueCount[stats.Values[j]]
	})

	// 检测字段类型
	stats.detectType()

	return stats
}

// detectType 检测字段类型
func (fs *FieldStats) detectType() {
	if len(fs.Values) == 0 {
		return
	}

	sampleCount := min(3, len(fs.Values))
	allNumeric := true
	allDate := true

	for i := 0; i < sampleCount; i++ {
		val := fs.Values[i]
		if _, err := strconv.ParseFloat(val, 64); err != nil {
			allNumeric = false
		}
		if !isDateLike(val) {
			allDate = false
		}
	}

	fs.IsString = !allNumeric && !allDate
	fs.IsNumeric = allNumeric
	fs.IsDate = allDate
}

// isDateLike 简单日期检测
func isDateLike(val string) bool {
	patterns := []string{
		"2006-01-02", "2006/01/02",
		"2006年01月02日", "01/02/2006",
	}
	for _, p := range patterns {
		if _, err := time.Parse(p, val); err == nil {
			return true
		}
	}
	return false
}

// ========== 规律分析 ==========

// analyzeFieldPatterns 分析字段的所有规律
func analyzeFieldPatterns(field string, rows []Row, commonSet *RowSet) *FieldChain {
	// 1. 预处理
	stats := preprocessField(field, rows)
	if len(stats.Values) == 0 {
		return nil
	}

	// 2. 分析各种规律
	var allNodes []*FieldNode

	// 等于规律
	equalNodes := analyzeEqualPatterns(stats, commonSet)
	allNodes = append(allNodes, equalNodes...)

	// 如果没有完美条件，尝试其他
	if !hasPerfectNode(equalNodes) {
		// 前缀规律（字符串）
		if stats.IsString && len(stats.Values) > 1 {
			prefixNodes := analyzePrefixPatterns(stats, commonSet)
			allNodes = append(allNodes, prefixNodes...)
		}

		// 范围规律（数值/日期）
		if (stats.IsNumeric || stats.IsDate) && len(stats.Values) > 1 {
			rangeNodes := analyzeRangePatterns(stats, commonSet)
			allNodes = append(allNodes, rangeNodes...)
		}
	}

	// 3. 构建链条
	return buildFieldChain(allNodes, stats.Field)
}

// analyzeEqualPatterns 等于规律
func analyzeEqualPatterns(stats *FieldStats, commonSet *RowSet) []*FieldNode {
	var nodes []*FieldNode

	for val, rowSet := range stats.ValueRows {
		if !coversAllCommon(rowSet, commonSet) {
			continue
		}

		dSet := rowSet.Clone()
		dSet.Subtract(commonSet)

		nodes = append(nodes, &FieldNode{
			Field:  stats.Field,
			Values: []string{val},
			D:      dSet,
		})
	}

	sortNodesByD(nodes)
	if len(nodes) > 0 {
		nodes[0].IsRoot = true
	}

	return nodes
}

// analyzePrefixPatterns 前缀规律
func analyzePrefixPatterns(stats *FieldStats, commonSet *RowSet) []*FieldNode {
	var nodes []*FieldNode
	prefixRows := make(map[string]*RowSet)

	// 生成前缀（前1-3个字符）
	for val, rowSet := range stats.ValueRows {
		maxLen := min(len(val), 3)
		for length := 1; length <= maxLen; length++ {
			prefix := val[:length]
			if set, exists := prefixRows[prefix]; exists {
				set.UnionWith(rowSet)
			} else {
				set = rowSet.Clone()
				prefixRows[prefix] = set
			}
		}
	}

	// 检查前缀
	for prefix, rowSet := range prefixRows {
		if !coversAllCommon(rowSet, commonSet) {
			continue
		}

		dSet := rowSet.Clone()
		dSet.Subtract(commonSet)

		nodes = append(nodes, &FieldNode{
			Field:  stats.Field,
			Values: []string{prefix + "*"},
			D:      dSet,
		})
	}

	sortNodesByD(nodes)
	return nodes
}

// analyzeRangePatterns 范围规律
func analyzeRangePatterns(stats *FieldStats, commonSet *RowSet) []*FieldNode {
	var nodes []*FieldNode

	// 尝试转换为数值
	var numbers []float64
	numberRows := make(map[float64]*RowSet)

	for val, rowSet := range stats.ValueRows {
		if f, err := strconv.ParseFloat(val, 64); err == nil {
			numbers = append(numbers, f)
			numberRows[f] = rowSet
		}
	}

	if len(numbers) < 2 {
		return nil
	}

	sort.Float64s(numbers)

	// 尝试几个分位点
	thresholds := []float64{0.1, 0.25, 0.5, 0.75, 0.9}
	for _, p := range thresholds {
		idx := int(float64(len(numbers)-1) * p)
		if idx < 0 || idx >= len(numbers) {
			continue
		}

		threshold := numbers[idx]

		// 收集大于等于阈值的行
		unionSet := NewRowSet(stats.TotalRows)
		for i := idx; i < len(numbers); i++ {
			if rowSet, exists := numberRows[numbers[i]]; exists {
				unionSet.UnionWith(rowSet)
			}
		}

		if !coversAllCommon(unionSet, commonSet) {
			continue
		}

		dSet := unionSet.Clone()
		dSet.Subtract(commonSet)

		nodes = append(nodes, &FieldNode{
			Field:  stats.Field,
			Values: []string{">=" + strconv.FormatFloat(threshold, 'f', -1, 64)},
			D:      dSet,
		})
	}

	sortNodesByD(nodes)
	return nodes
}

// ========== 工具函数 ==========

// coversAllCommon 检查是否包含所有共有行
func coversAllCommon(rowSet, commonSet *RowSet) bool {
	temp := commonSet.Clone()
	temp.Subtract(rowSet)
	return temp.Empty()
}

// hasPerfectNode 检查是否有完美节点
func hasPerfectNode(nodes []*FieldNode) bool {
	for _, node := range nodes {
		if node.D.Empty() {
			return true
		}
	}
	return false
}

// sortNodesByD 按D大小排序
func sortNodesByD(nodes []*FieldNode) {
	sort.Slice(nodes, func(i, j int) bool {
		return nodes[i].D.Len() < nodes[j].D.Len()
	})
}

// buildFieldChain 构建字段链条
func buildFieldChain(nodes []*FieldNode, field string) *FieldChain {
	if len(nodes) == 0 {
		return nil
	}

	sortNodesByD(nodes)
	nodes[0].IsRoot = true

	return &FieldChain{
		Field: field,
		Nodes: nodes,
		Root:  nodes[0],
	}
}

// min 最小值
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
