// pattern_analyzer.go - 规律分析核心逻辑
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

	valueCount := make(map[string]int)
	for i, row := range rows {
		val := strings.TrimSpace(row[field])
		if val == "" {
			continue
		}
		if set, exists := stats.ValueRows[val]; exists {
			set.Add(i)
		} else {
			set := NewRowSet(len(rows))
			set.Add(i)
			stats.ValueRows[val] = set
		}
		valueCount[val]++
	}

	for val := range stats.ValueRows {
		stats.Values = append(stats.Values, val)
	}

	sort.Slice(stats.Values, func(i, j int) bool {
		return valueCount[stats.Values[i]] > valueCount[stats.Values[j]]
	})

	stats.detectType()
	return stats
}

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

func analyzeFieldPatterns(field string, rows []Row, commonSet *RowSet) *FieldChain {
	stats := preprocessField(field, rows)
	if len(stats.Values) == 0 {
		return nil
	}

	var allNodes []*FieldNode

	equalNodes := analyzeEqualPatterns(stats, commonSet)
	allNodes = append(allNodes, equalNodes...)

	enumNodes := analyzeEnumPatterns(stats, commonSet, 10)
	allNodes = append(allNodes, enumNodes...)

	if stats.IsString && len(stats.Values) > 1 {
		prefixNodes := analyzePrefixPatterns(stats, commonSet)
		allNodes = append(allNodes, prefixNodes...)
	}

	if stats.IsString && len(stats.Values) > 1 {
		prefixEnumNodes := analyzePrefixEnumPatterns(stats, commonSet, 10)
		allNodes = append(allNodes, prefixEnumNodes...)
	}

	if (stats.IsNumeric || stats.IsDate) && len(stats.Values) > 1 {
		rangeNodes := analyzeRangePatterns(stats, commonSet)
		allNodes = append(allNodes, rangeNodes...)
	}

	return buildFieldChainWithLimit(allNodes, stats.Field, 4)
}

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
	return nodes
}

func analyzeEnumPatterns(stats *FieldStats, commonSet *RowSet, maxValues int) []*FieldNode {
	var nodes []*FieldNode

	// 剪枝：只保留与 commonSet 有交集的值
	candidateValues := []string{}
	for _, val := range stats.Values {
		if rowSet, exists := stats.ValueRows[val]; exists {
			if !rowSet.Disjoint(commonSet) {
				candidateValues = append(candidateValues, val)
			}
		}
	}
	if len(candidateValues) == 0 {
		return nodes
	}

	// 最多前15个高频值
	if len(candidateValues) > 15 {
		candidateValues = candidateValues[:15]
	}

	// 尝试不同集合大小
	for setSize := 2; setSize <= maxValues && setSize <= len(candidateValues); setSize++ {
		forEachCombination(candidateValues, setSize, 500, func(combo []string) bool {
			unionSet := NewRowSet(stats.TotalRows)
			for _, val := range combo {
				if rs, ok := stats.ValueRows[val]; ok {
					unionSet.UnionWith(rs)
				}
			}
			if coversAllCommon(unionSet, commonSet) {
				dSet := unionSet.Clone()
				dSet.Subtract(commonSet)
				nodes = append(nodes, &FieldNode{
					Field:  stats.Field,
					Values: combo,
					D:      dSet,
				})
			}
			return len(nodes) < 20 // 最多20个节点
		})
	}

	return nodes
}

func analyzePrefixPatterns(stats *FieldStats, commonSet *RowSet) []*FieldNode {
	var nodes []*FieldNode
	prefixRows := make(map[string]*RowSet)

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

	return nodes
}

func analyzePrefixEnumPatterns(stats *FieldStats, commonSet *RowSet, maxPrefixes int) []*FieldNode {
	var nodes []*FieldNode

	prefixRows := make(map[string]*RowSet)
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

	candidatePrefixes := []string{}
	for prefix, rowSet := range prefixRows {
		if !rowSet.Disjoint(commonSet) {
			candidatePrefixes = append(candidatePrefixes, prefix)
		}
	}
	if len(candidatePrefixes) == 0 {
		return nodes
	}

	if len(candidatePrefixes) > 15 {
		candidatePrefixes = candidatePrefixes[:15]
	}

	for setSize := 2; setSize <= maxPrefixes && setSize <= len(candidatePrefixes); setSize++ {
		forEachCombination(candidatePrefixes, setSize, 500, func(combo []string) bool {
			unionSet := NewRowSet(stats.TotalRows)
			for _, prefix := range combo {
				if rowSet, exists := prefixRows[prefix]; exists {
					unionSet.UnionWith(rowSet)
				}
			}
			if !coversAllCommon(unionSet, commonSet) {
				return true
			}
			dSet := unionSet.Clone()
			dSet.Subtract(commonSet)
			prefixPatterns := make([]string, len(combo))
			for i, p := range combo {
				prefixPatterns[i] = p + "*"
			}
			nodes = append(nodes, &FieldNode{
				Field:  stats.Field,
				Values: prefixPatterns,
				D:      dSet,
			})
			return len(nodes) < 20
		})
	}

	return nodes
}

func analyzeRangePatterns(stats *FieldStats, commonSet *RowSet) []*FieldNode {
	var nodes []*FieldNode

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

	thresholds := []float64{0.1, 0.25, 0.5, 0.75, 0.9}
	for _, p := range thresholds {
		idx := int(float64(len(numbers)-1) * p)
		if idx < 0 || idx >= len(numbers) {
			continue
		}
		threshold := numbers[idx]

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

func coversAllCommon(rowSet, commonSet *RowSet) bool {
	temp := commonSet.Clone()
	temp.Subtract(rowSet)
	return temp.Empty()
}

func sortNodesByD(nodes []*FieldNode) {
	sort.Slice(nodes, func(i, j int) bool {
		return nodes[i].D.Len() < nodes[j].D.Len()
	})
}

func buildFieldChainWithLimit(nodes []*FieldNode, field string, maxNodes int) *FieldChain {
	if len(nodes) == 0 {
		return nil
	}

	nodes = deduplicateNodes(nodes)
	sortNodesByD(nodes)
	if len(nodes) > maxNodes {
		nodes = nodes[:maxNodes]
	}
	if len(nodes) > 0 {
		nodes[0].IsRoot = true
	}
	return &FieldChain{
		Field: field,
		Nodes: nodes,
		Root:  nodes[0],
	}
}

// isPrefixPattern 判断是否为前缀模式
func isPrefixPattern(values []string) bool {
	for _, val := range values {
		if strings.HasSuffix(val, "*") {
			return true
		}
	}
	return false
}

// deduplicateNodes 使用 RowSet.String() 去重（依赖你独立文件中的正确实现）
func deduplicateNodes(nodes []*FieldNode) []*FieldNode {
	if len(nodes) <= 1 {
		return nodes
	}

	groups := make(map[string][]*FieldNode)
	for _, node := range nodes {
		key := node.D.String() // ← 依赖你 RowSet 中已修正的 String()
		groups[key] = append(groups[key], node)
	}

	var result []*FieldNode
	for _, group := range groups {
		best := group[0]
		for _, node := range group[1:] {
			// 优先选择非前缀模式
			bestIsPrefix := isPrefixPattern(best.Values)
			nodeIsPrefix := isPrefixPattern(node.Values)

			if !nodeIsPrefix && bestIsPrefix {
				// 节点不是前缀模式，当前最佳是前缀模式 → 选择节点
				best = node
			} else if !nodeIsPrefix && !bestIsPrefix {
				// 两者都不是前缀模式 → 选择值长度更短的
				if len(node.Values) < len(best.Values) {
					best = node
				}
			} else if nodeIsPrefix && bestIsPrefix {
				// 两者都是前缀模式 → 选择值长度更短的
				if len(node.Values) < len(best.Values) {
					best = node
				}
			}
		}
		result = append(result, best)
	}
	return result
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// ========== 新增：流式组合生成器 ==========

func forEachCombination(items []string, size, maxTries int, callback func(combo []string) bool) {
	if size <= 0 || size > len(items) || maxTries <= 0 {
		return
	}

	count := 0
	var current []string

	var backtrack func(start int) bool
	backtrack = func(start int) bool {
		if len(current) == size {
			combo := make([]string, size)
			copy(combo, current)
			count++
			if !callback(combo) || count >= maxTries {
				return false
			}
			return true
		}

		for i := start; i <= len(items)-(size-len(current)); i++ {
			current = append(current, items[i])
			if !backtrack(i + 1) {
				return false
			}
			current = current[:len(current)-1]
		}
		return true
	}

	backtrack(0)
}
