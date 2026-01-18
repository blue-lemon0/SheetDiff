package main

import (
	"fmt"
	"sort"
	"strings"
)

// ========== 基本数据类型 ==========

// Row 数据行类型
//type Row map[string]string

// ========== 算法核心数据结构 ==========

// FieldNode 字段节点
type FieldNode struct {
	Field  string
	Values []string
	D      *RowSet // 独有行集合
	IsRoot bool
}

// FieldChain 字段链条
type FieldChain struct {
	Field string
	Nodes []*FieldNode
	Root  *FieldNode
}

// Tree 支配关系树
type Tree struct {
	Root     *FieldNode
	Children []*Tree
	Parent   *Tree
}

// PerfectRule 完美规则
type PerfectRule struct {
	Conditions  []RuleCondition
	Covered     int
	TotalCommon int
}

// RuleCondition 规则条件
type RuleCondition struct {
	Field  string
	Values []string
}

// ========== 构建器类 ==========

// candidate 候选组合
type candidate struct {
	values []string
	dSet   *RowSet
}

// FieldChainBuilder 字段链条构建器
type FieldChainBuilder struct {
	totalRows int
	commonSet *RowSet
}

// ========== 主算法类 ==========

// RuleFinder 规则发现器
type RuleFinder struct {
	MaxValuesPerRule int
	MaxConditions    int
	MaxRowsToAnalyze int
}

// NewRuleFinder 创建规则发现器
func NewRuleFinder() *RuleFinder {
	return &RuleFinder{
		MaxValuesPerRule: 3,
		MaxConditions:    3,
		MaxRowsToAnalyze: 10000,
	}
}

// ========== 核心算法方法 ==========

// FindRules 发现规则（主要入口）
func (rf *RuleFinder) FindRows(rows []Row, commonSet *RowSet) []PerfectRule {
	// 1. 构建所有字段链条
	chains := rf.buildAllChains(rows, commonSet)

	// 2. 建立支配关系森林
	forest := rf.buildForest(chains)

	// 3. 寻找完美规则
	rules := rf.findPerfectRules(forest, commonSet.Count(), len(rows))

	return rules
}

// buildAllChains 构建所有字段链条
func (rf *RuleFinder) buildAllChains(rows []Row, commonSet *RowSet) []*FieldChain {
	// 获取所有字段名
	if len(rows) == 0 {
		return []*FieldChain{}
	}

	fields := make([]string, 0, len(rows[0]))
	for field := range rows[0] {
		fields = append(fields, field)
	}

	// 为每个字段构建链条
	builder := &FieldChainBuilder{
		totalRows: len(rows),
		commonSet: commonSet,
	}

	var chains []*FieldChain
	for _, field := range fields {
		chain := builder.BuildChain(field, rows)
		if chain != nil && chain.Root != nil {
			chains = append(chains, chain)
		}
	}

	return chains
}

// buildForest 建立支配关系森林
func (rf *RuleFinder) buildForest(chains []*FieldChain) []*Tree {
	// 收集所有根节点
	var roots []*FieldNode
	for _, chain := range chains {
		if chain.Root != nil {
			roots = append(roots, chain.Root)
		}
	}

	return rf.buildTreesFromRoots(roots)
}

// buildTreesFromRoots 从根节点构建树
func (rf *RuleFinder) buildTreesFromRoots(roots []*FieldNode) []*Tree {
	if len(roots) == 0 {
		return []*Tree{}
	}

	// 1. 建立支配关系
	childrenMap := make(map[*FieldNode][]*FieldNode)

	for i := 0; i < len(roots); i++ {
		for j := 0; j < len(roots); j++ {
			if i == j {
				continue
			}

			// 检查支配关系
			if roots[i].D.IsSubsetOf(roots[j].D) {
				// roots[i] ⊆ roots[j]，roots[i]更好（更小的D）
				childrenMap[roots[i]] = append(childrenMap[roots[i]], roots[j])
			} else if roots[j].D.IsSubsetOf(roots[i].D) {
				childrenMap[roots[j]] = append(childrenMap[roots[j]], roots[i])
			}
		}
	}

	// 2. 找到顶级节点（不被任何其他节点支配的）
	var topLevelNodes []*FieldNode
	for _, node := range roots {
		isTopLevel := true
		for _, other := range roots {
			if node == other {
				continue
			}
			// 如果其他节点支配这个节点，那么它不是顶级
			if other.D.IsSubsetOf(node.D) && !node.D.IsSubsetOf(other.D) {
				isTopLevel = false
				break
			}
		}
		if isTopLevel {
			topLevelNodes = append(topLevelNodes, node)
		}
	}

	// 3. 构建树
	var forest []*Tree
	for _, rootNode := range topLevelNodes {
		tree := &Tree{Root: rootNode}
		if children, exists := childrenMap[rootNode]; exists {
			for _, childNode := range children {
				// 检查这个child是否确实被root支配
				if childNode.D.IsSubsetOf(rootNode.D) {
					tree.Children = append(tree.Children, &Tree{
						Root:   childNode,
						Parent: tree,
					})
				}
			}
		}
		forest = append(forest, tree)
	}

	return forest
}

// findPerfectRules 寻找完美规则
func (rf *RuleFinder) findPerfectRules(forest []*Tree, commonCount, totalRows int) []PerfectRule {
	// 收集所有树的根节点
	var allRoots []*FieldNode
	for _, tree := range forest {
		if tree.Root != nil {
			allRoots = append(allRoots, tree.Root)
		}
	}

	return rf.searchRuleCombinations(allRoots, commonCount, totalRows)
}

// searchRuleCombinations 搜索规则组合
func (rf *RuleFinder) searchRuleCombinations(roots []*FieldNode, commonCount, totalRows int) []PerfectRule {
	var rules []PerfectRule
	nonCRows := totalRows - commonCount

	// 1. 检查单个完美字段（D为空）
	for _, root := range roots {
		if root.D.Count() == 0 {
			rules = append(rules, PerfectRule{
				Conditions:  []RuleCondition{{Field: root.Field, Values: root.Values}},
				Covered:     commonCount,
				TotalCommon: commonCount,
			})
			return rules // 有完美单字段，直接返回
		}
	}

	// 2. 尝试两个字段的组合
	for i := 0; i < len(roots); i++ {
		for j := i + 1; j < len(roots); j++ {
			// 检查两个D是否不相交
			if !roots[i].D.HasIntersection(roots[j].D) {
				totalDSize := roots[i].D.Count() + roots[j].D.Count()

				// 如果正好覆盖所有非共有行
				if totalDSize == nonCRows {
					rules = append(rules, PerfectRule{
						Conditions: []RuleCondition{
							{Field: roots[i].Field, Values: roots[i].Values},
							{Field: roots[j].Field, Values: roots[j].Values},
						},
						Covered:     commonCount,
						TotalCommon: commonCount,
					})
				}
			}
		}
	}

	if len(rules) > 0 {
		return rules
	}

	// 3. 尝试三个字段的组合
	for i := 0; i < len(roots); i++ {
		for j := i + 1; j < len(roots); j++ {
			for k := j + 1; k < len(roots); k++ {
				// 检查三个D是否两两不相交
				if !roots[i].D.HasIntersection(roots[j].D) &&
					!roots[i].D.HasIntersection(roots[k].D) &&
					!roots[j].D.HasIntersection(roots[k].D) {

					totalDSize := roots[i].D.Count() + roots[j].D.Count() + roots[k].D.Count()

					if totalDSize == nonCRows {
						rules = append(rules, PerfectRule{
							Conditions: []RuleCondition{
								{Field: roots[i].Field, Values: roots[i].Values},
								{Field: roots[j].Field, Values: roots[j].Values},
								{Field: roots[k].Field, Values: roots[k].Values},
							},
							Covered:     commonCount,
							TotalCommon: commonCount,
						})
					}
				}
			}
		}
	}

	return rules
}

// ========== FieldChainBuilder 的方法 ==========

func (b *FieldChainBuilder) BuildChain(field string, rows []Row) *FieldChain {
	// 1. 找出所有能覆盖commonSet的取值组合
	candidates := b.findAllCoveringCombinations(field, rows)

	fmt.Printf("  字段 %s: 找到 %d 个候选组合", field, len(candidates))
	if len(candidates) > 0 {
		fmt.Printf(" (最小D=%d)", candidates[0].dSet.Count())
	}
	fmt.Println()

	if len(candidates) == 0 {
		return nil
	}

	// 2. 按独有行数排序
	b.sortCandidatesByDSize(candidates)

	// 3. 构建节点链条
	chain := &FieldChain{Field: field}
	for i, cand := range candidates {
		node := &FieldNode{
			Field:  field,
			Values: cand.values,
			D:      cand.dSet,
			IsRoot: i == 0,
		}
		chain.Nodes = append(chain.Nodes, node)
	}

	if len(chain.Nodes) > 0 {
		chain.Root = chain.Nodes[0]
	}

	return chain
}

// findAllCoveringCombinations 找出所有能覆盖C的取值组合
func (b *FieldChainBuilder) findAllCoveringCombinations(field string, rows []Row) []candidate {
	var candidates []candidate

	// 简单实现：只考虑单个值
	candidates = append(candidates, b.findSingleValueCombinations(field, rows)...)

	return candidates
}

func (b *FieldChainBuilder) findSingleValueCombinations(field string, rows []Row) []candidate {
	// 统计每个值对应的行
	valueRows := make(map[string]*RowSet)
	for i, row := range rows {
		val := strings.TrimSpace(row[field])
		if set, exists := valueRows[val]; exists {
			set.Set(i)
		} else {
			set = NewRowSet(b.totalRows)
			set.Set(i)
			valueRows[val] = set
		}
	}

	var candidates []candidate
	for val, rowSet := range valueRows {
		// 检查是否包含所有共有行
		// 正确方法：commonSet ⊆ rowSet
		containsAllCommon := true

		// 遍历所有行，检查共有行是否都在rowSet中
		for i := 0; i < b.totalRows; i++ {
			// 如果这是共有行
			if b.commonSet.IntersectionCount(NewRowSetWithValue(b.totalRows, i)) > 0 {
				// 检查是否在rowSet中
				if rowSet.IntersectionCount(NewRowSetWithValue(b.totalRows, i)) == 0 {
					containsAllCommon = false
					break
				}
			}
		}

		if containsAllCommon {
			// 计算独有行集合 D = rowSet - commonSet
			dSet := rowSet.Clone()
			dSet.Subtract(b.commonSet)

			candidates = append(candidates, candidate{
				values: []string{val},
				dSet:   dSet,
			})
		}
	}

	return candidates
}

// sortCandidatesByDSize 按D集合大小排序
func (b *FieldChainBuilder) sortCandidatesByDSize(candidates []candidate) {
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].dSet.Count() < candidates[j].dSet.Count()
	})
}
