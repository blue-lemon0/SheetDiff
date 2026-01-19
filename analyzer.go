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

	// 构建主表的主键映射（保留索引）
	mainMap := make(map[string]int)
	for i, row := range mainData {
		key := buildRowKey(row, config.MainKeys)
		if key != "" {
			mainMap[key] = i
		}
	}

	// 构建参考表的主键映射（保留索引）
	refMap := make(map[string]int)
	for i, row := range refData {
		key := buildRowKey(row, config.RefKeys)
		if key != "" {
			refMap[key] = i
		}
	}

	// 找出匹配的行
	for key, mainIdx := range mainMap {
		if refIdx, exists := refMap[key]; exists {
			result.Matched = append(result.Matched, MatchedPair{
				MainRow:   mainData[mainIdx],
				RefRow:    refData[refIdx],
				Key:       key,
				MainIndex: mainIdx,
				RefIndex:  refIdx,
			})
			delete(refMap, key) // 从参考表map中移除，剩下的就是仅参考表有的
		} else {
			result.OnlyMain = append(result.OnlyMain, mainData[mainIdx])
		}
	}

	// 剩下的参考表行就是仅参考表有的
	for _, refIdx := range refMap {
		result.OnlyRef = append(result.OnlyRef, refData[refIdx])
	}

	return result
}

// buildRowKey 构建行的主键字符串
func buildRowKey(row Row, keyFields []string) string {
	parts := make([]string, 0, len(keyFields))
	for _, field := range keyFields {
		if value, exists := row[field]; exists {
			parts = append(parts, strings.TrimSpace(value))
		} else {
			// 字段不存在时，直接返回空字符串，表示主键无效
			return ""
		}
	}
	return strings.Join(parts, "|")
}

// ========== 核心算法调用 ==========

// RuleAnalysisResult 规则分析结果
type RuleAnalysisResult struct {
	Chains       []*FieldChain        // 过滤条件链
	Forest       []*Tree              // 支配关系森林
	NodeToTree   map[*FieldNode]*Tree // 节点到树的映射
	PerfectRules []PerfectRule        // 完美规则组合
}

// AnalyzeRulesForBothSheets 分析两个sheet各自的过滤规则
// 基于共有行和独有行，为每个sheet独立分析过滤条件
func AnalyzeRulesForBothSheets(result MatchResult, mainData, refData []Row,
	mainHeaders, refHeaders, mainKeyFields, refKeyFields []string) (
	mainResult, refResult RuleAnalysisResult) {

	// 1. 分析主表的过滤规则
	mainResult = analyzeSheetRules(mainData, mainHeaders, mainKeyFields, result.Matched, true, len(result.OnlyMain))

	// 2. 分析参考表的过滤规则
	refResult = analyzeSheetRules(refData, refHeaders, refKeyFields, result.Matched, false, len(result.OnlyRef))

	return mainResult, refResult
}

// analyzeSheetRules 分析单个sheet的过滤规则
func analyzeSheetRules(data []Row, headers, keyFields []string,
	matched []MatchedPair, isMainSheet bool, onlyRowCount int) RuleAnalysisResult {

	result := RuleAnalysisResult{}

	if len(data) == 0 {
		return result
	}

	// 1. 构建共有行集合
	commonSet := NewRowSet(len(data))
	for _, pair := range matched {
		if isMainSheet {
			commonSet.Add(pair.MainIndex)
		} else {
			commonSet.Add(pair.RefIndex)
		}
	}

	// 2. 准备分析用的数据（移除主键字段）
	analysisRows := make([]Row, len(data))
	for i, row := range data {
		filteredRow := Row{}
		for k, v := range row {
			// 检查是否是主键字段
			isKey := false
			for _, key := range keyFields {
				if k == key {
					isKey = true
					break
				}
			}
			// 只保留非主键字段
			if !isKey {
				filteredRow[k] = v
			}
		}
		analysisRows[i] = filteredRow
	}

	// 3. 为每个非主键字段构建链条
	finder := NewRuleFinder()
	chains := finder.buildAllChains(analysisRows, commonSet)

	// 4. 过滤掉没有有效节点的链条
	var validChains []*FieldChain
	for _, chain := range chains {
		if chain != nil && len(chain.Nodes) > 0 {
			validChains = append(validChains, chain)
		}
	}
	result.Chains = validChains

	// 5. 构建支配关系森林
	result.Forest, result.NodeToTree = finder.buildForest(validChains)

	// 6. 寻找完美规则组合
	commonCount := commonSet.Len()
	totalRows := len(data)
	result.PerfectRules = finder.findPerfectRules(result.Forest, commonCount, totalRows)

	return result
}
