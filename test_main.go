// simple_test.go
package main

import (
	"fmt"
)

// ========== 测试函数 ==========

// 测试1：完美等于条件
func testPerfectEqual() {
	fmt.Println("🧪 测试1：完美等于条件")
	fmt.Println("  场景: 所有共有行都是'技术部'")
	fmt.Println("  期望: 找到完美规则 部门=技术部")

	rows := []Row{
		{"部门": "技术部", "城市": "北京"},
		{"部门": "技术部", "城市": "上海"},
		{"部门": "技术部", "城市": "广州"},
		{"部门": "市场部", "城市": "北京"},
		{"部门": "研发部", "城市": "上海"},
	}

	commonSet := NewRowSet(5)
	commonSet.Add(0)
	commonSet.Add(1)
	commonSet.Add(2)

	fmt.Println("\n  1. 单字段分析:")
	chain := analyzeFieldPatterns("部门", rows, commonSet)
	printChain("部门", chain)

	chain = analyzeFieldPatterns("城市", rows, commonSet)
	printChain("城市", chain)

	fmt.Println("\n  2. 完整算法:")
	finder := NewRuleFinder()
	rules := finder.FindRules(rows, commonSet)
	printRules(rules)
}

// 测试2：前缀条件
func testPrefix() {
	fmt.Println("\n🧪 测试2：前缀条件")
	fmt.Println("  场景: 共有行的编码都以A开头")
	fmt.Println("  期望: 找到完美规则 编码=A*")

	rows := []Row{
		{"编码": "A001", "部门": "技术部"},
		{"编码": "A002", "部门": "技术部"},
		{"编码": "A003", "部门": "技术部"},
		{"编码": "B001", "部门": "市场部"},
		{"编码": "C001", "部门": "研发部"},
	}

	commonSet := NewRowSet(5)
	commonSet.Add(0)
	commonSet.Add(1)
	commonSet.Add(2)

	fmt.Println("\n  1. 单字段分析:")
	chain := analyzeFieldPatterns("编码", rows, commonSet)
	printChain("编码", chain)

	chain = analyzeFieldPatterns("部门", rows, commonSet)
	printChain("部门", chain)

	fmt.Println("\n  2. 完整算法:")
	finder := NewRuleFinder()
	rules := finder.FindRules(rows, commonSet)
	printRules(rules)
}

// 测试3：范围条件
func testRange() {
	fmt.Println("\n🧪 测试3：范围条件")
	fmt.Println("  场景: 共有行的年龄都>=32")
	fmt.Println("  期望: 找到完美规则 年龄>=32")

	rows := []Row{
		{"年龄": "35", "姓名": "张三"},
		{"年龄": "32", "姓名": "李四"},
		{"年龄": "38", "姓名": "王五"},
		{"年龄": "25", "姓名": "赵六"},
		{"年龄": "28", "姓名": "钱七"},
	}

	commonSet := NewRowSet(5)
	commonSet.Add(0)
	commonSet.Add(1)
	commonSet.Add(2)

	fmt.Println("\n  1. 单字段分析:")
	chain := analyzeFieldPatterns("年龄", rows, commonSet)
	printChain("年龄", chain)

	fmt.Println("\n  2. 完整算法:")
	finder := NewRuleFinder()
	rules := finder.FindRules(rows, commonSet)
	printRules(rules)
}

// 测试4：组合条件
func testCombination() {
	fmt.Println("\n🧪 测试4：组合条件")
	fmt.Println("  场景: 需要两个字段组合才能完美覆盖")
	fmt.Println("  期望: 找到组合规则 部门=技术部 且 级别=P7")

	rows := []Row{
		{"部门": "技术部", "级别": "P7", "城市": "北京"},
		{"部门": "技术部", "级别": "P7", "城市": "上海"},
		{"部门": "技术部", "级别": "P7", "城市": "广州"},
		{"部门": "技术部", "级别": "P8", "城市": "北京"}, // 独有：级别不同
		{"部门": "市场部", "级别": "P7", "城市": "北京"}, // 独有：部门不同
	}

	commonSet := NewRowSet(5)
	commonSet.Add(0)
	commonSet.Add(1)
	commonSet.Add(2)

	fmt.Println("\n  1. 单字段分析:")
	fields := []string{"部门", "级别", "城市"}
	for _, field := range fields {
		chain := analyzeFieldPatterns(field, rows, commonSet)
		printChain(field, chain)
	}

	fmt.Println("\n  2. 完整算法:")
	finder := NewRuleFinder()

	// 先看支配关系森林
	chains := finder.buildAllChains(rows, commonSet)
	forest := finder.buildForest(chains)
	printForest(forest)

	// 再看最终规则
	rules := finder.FindRules(rows, commonSet)
	printRules(rules)
}

// 测试5：无完美规则
func testNoPerfect() {
	fmt.Println("\n🧪 测试5：无完美规则")
	fmt.Println("  场景: 没有单个字段能完美覆盖")
	fmt.Println("  期望: 可能找到组合规则或无规则")

	rows := []Row{
		{"部门": "技术部", "城市": "北京"},
		{"部门": "市场部", "城市": "上海"},
		{"部门": "研发部", "城市": "广州"},
		{"部门": "技术部", "城市": "深圳"},
		{"部门": "市场部", "城市": "杭州"},
	}

	commonSet := NewRowSet(5)
	commonSet.Add(0)
	commonSet.Add(1)
	commonSet.Add(2)

	fmt.Println("\n  1. 单字段分析:")
	chain := analyzeFieldPatterns("部门", rows, commonSet)
	printChain("部门", chain)

	chain = analyzeFieldPatterns("城市", rows, commonSet)
	printChain("城市", chain)

	fmt.Println("\n  2. 完整算法:")
	finder := NewRuleFinder()
	rules := finder.FindRules(rows, commonSet)
	printRules(rules)
}

// ========== 辅助函数 ==========

// 打印链条（显示链条结构）
func printChain(field string, chain *FieldChain) {
	if chain == nil || len(chain.Nodes) == 0 {
		fmt.Printf("  %s: 空链条\n", field)
		return
	}

	fmt.Printf("  %s: %d个节点的链条", field, len(chain.Nodes))
	if chain.Root != nil {
		fmt.Printf(" (根节点D=%d)", chain.Root.D.Len())
	}
	fmt.Println()

	// 显示链条结构
	for i, node := range chain.Nodes {
		prefix := "   "
		if node.IsRoot {
			prefix = " 🎯"
		} else if i == len(chain.Nodes)-1 {
			prefix = " └─"
		} else {
			prefix = " ├─"
		}

		// 显示包含关系
		relation := ""
		if i > 0 {
			relation = "⊂ "
		}

		fmt.Printf("%s [D%d] %s = %v (D大小=%d) %s\n",
			prefix, i+1, node.Field, node.Values, node.D.Len(), relation)

		// 如果D=0，标记为完美
		if node.D.Len() == 0 {
			fmt.Printf("      ✅ 完美节点！\n")
		}
	}

	// 显示链条总结
	if len(chain.Nodes) > 1 {
		fmt.Printf("  链条关系: D₁")
		for i := 1; i < len(chain.Nodes); i++ {
			fmt.Printf(" ⊂ D%d", i+1)
		}
		fmt.Println()
	}
}

// 打印支配关系树
func printTree(tree *Tree, depth int) {
	if tree == nil || tree.Root == nil {
		return
	}

	prefix := ""
	for i := 0; i < depth; i++ {
		prefix += "  "
	}

	if depth == 0 {
		prefix = "🌲 "
	} else {
		prefix += "└─ "
	}

	fmt.Printf("%s%s = %v (D=%d)\n",
		prefix, tree.Root.Field, tree.Root.Values, tree.Root.D.Len())

	for _, child := range tree.Children {
		printTree(child, depth+1)
	}
}

// 打印森林
func printForest(forest []*Tree) {
	if len(forest) == 0 {
		fmt.Println("  空森林")
		return
	}

	fmt.Printf("  支配关系森林 (%d棵树):\n", len(forest))
	for i, tree := range forest {
		fmt.Printf("  树%d:\n", i+1)
		printTree(tree, 0)
	}
}

// 打印规则
func printRules(rules []PerfectRule) {
	if len(rules) == 0 {
		fmt.Println("  无规则")
		return
	}

	fmt.Printf("  找到%d个规则:\n", len(rules))
	for i, rule := range rules {
		fmt.Printf("  规则%d: ", i+1)
		for j, cond := range rule.Conditions {
			if j > 0 {
				fmt.Printf(" 且 ")
			}
			fmt.Printf("%s = %v", cond.Field, cond.Values)
		}
		perfect := ""
		if rule.Covered == rule.TotalCommon {
			perfect = " ✅"
		}
		fmt.Printf(" (覆盖%d/%d行)%s\n", rule.Covered, rule.TotalCommon, perfect)
	}
}

// ========== 主函数 ==========

func main() {
	fmt.Println("🚀 规则发现算法测试")
	fmt.Println("==================")

	testPerfectEqual()
	testPrefix()
	testRange()
	testCombination()
	testNoPerfect()

	fmt.Println("\n==================")
	fmt.Println("✅ 所有测试运行完成")
}
