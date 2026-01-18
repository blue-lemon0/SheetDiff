package main

import (
	"fmt"
	"strings"
)

func main() {
	fmt.Println("🧪 开始CSV测试...")

	// 1. 加载CSV测试数据
	loader := &TestCSVLoader{}
	testData, err := loader.LoadTestData("test_data/perfect_test.csv") // 修改这里
	if err != nil {
		fmt.Printf("❌ 加载测试数据失败: %v\n", err)
		return
	}

	loader.PrintTestData(testData)

	// 2. 运行算法
	finder := NewRuleFinder()
	fmt.Println("\n🔍 开始规则发现算法...")
	rules := finder.FindRulesFromData(testData.Rows, testData.CommonSet, testData.TotalRows)

	// 3. 打印结果
	fmt.Printf("\n🎯 找到了 %d 个完美规则:\n", len(rules))

	for i, rule := range rules {
		fmt.Printf("\n规则%d (%d个条件):\n", i+1, len(rule.Conditions))
		for _, cond := range rule.Conditions {
			valueStr := cond.Values[0]
			if len(cond.Values) > 1 {
				valueStr = "[" + strings.Join(cond.Values, ",") + "]"
			}
			fmt.Printf("  %s = %s\n", cond.Field, valueStr)
		}
		fmt.Printf("  覆盖: %d/%d 共有行\n", rule.Covered, rule.TotalCommon)
	}

	if len(rules) == 0 {
		fmt.Println("\n❌ 没有找到完美规则")
	}

	fmt.Println("\n✅ 测试完成")
}

// 在RuleFinder中添加这个方法
func (rf *RuleFinder) FindRulesFromCSV(data *TestData) []PerfectRule {
	// 重用原来的FindRules方法
	return rf.FindRulesFromData(data.Rows, data.CommonSet, data.TotalRows)
}

// 这个方法是原有逻辑的包装
func (rf *RuleFinder) FindRulesFromData(rows []Row, commonSet *RowSet, totalRows int) []PerfectRule {
	fmt.Println("🔍 开始分析数据...")

	// 1. 构建所有字段链条
	chains := rf.buildAllChains(rows, commonSet)
	fmt.Printf("✅ 构建了 %d 个字段链条\n", len(chains))

	for _, chain := range chains {
		if chain.Root != nil {
			fmt.Printf("  字段 %s: D大小=%d, 值=%v\n",
				chain.Field, chain.Root.D.Count(), chain.Root.Values)
		}
	}

	if len(chains) == 0 {
		fmt.Println("⚠️ 没有可用的字段链条")
		return []PerfectRule{}
	}

	// 2. 建立支配关系森林
	forest := rf.buildForest(chains)
	fmt.Printf("✅ 构建了 %d 棵树\n", len(forest))

	// 3. 收集所有根节点
	var allRoots []*FieldNode
	for _, tree := range forest {
		if tree.Root != nil {
			allRoots = append(allRoots, tree.Root)
		}
		for _, child := range tree.Children {
			if child.Root != nil {
				allRoots = append(allRoots, child.Root)
			}
		}
	}

	fmt.Printf("📋 共有 %d 个候选节点\n", len(allRoots))

	// 4. 寻找完美规则
	rules := rf.searchRuleCombinations(allRoots, commonSet.Count(), totalRows)

	return rules
}
