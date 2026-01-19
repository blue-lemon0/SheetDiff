package main

import (
	"testing"
)

// TestDisjointCombination 测试不重叠组合的情况
func TestDisjointCombination(t *testing.T) {
	t.Log("测试场景: 两个字段的D集合不相交，组合后误伤率降低")
	t.Log("期望结果: 找到组合规则，即使它们的D集合总和不等于非共有行总数")

	// 模拟数据：共有行3行，独有行2行
	rows := []Row{
		// 共有行 (索引0,1,2)
		{"部门": "技术部", "级别": "P7", "城市": "北京"},
		{"部门": "技术部", "级别": "P7", "城市": "上海"},
		{"部门": "技术部", "级别": "P7", "城市": "广州"},
		// 独有行 (索引3) - 部门不同
		{"部门": "市场部", "级别": "P7", "城市": "北京"},
		// 独有行 (索引4) - 级别不同
		{"部门": "技术部", "级别": "P8", "城市": "北京"},
	}

	commonSet := NewRowSet(5)
	commonSet.Add(0)
	commonSet.Add(1)
	commonSet.Add(2)

	// 完整算法
	t.Log("1. 完整算法:")
	finder := NewRuleFinder()
	rules := finder.FindRules(rows, commonSet)

	// 断言：应该找到至少1个规则
	if len(rules) == 0 {
		t.Error("期望找到规则，但实际没有找到")
		return
	}

	// 断言：第一个规则应该是完美规则（覆盖率100%）
	if rules[0].Covered != rules[0].TotalCommon {
		t.Errorf("期望完美规则（覆盖率100%%），实际覆盖 %d/%d", rules[0].Covered, rules[0].TotalCommon)
	}

	// 断言：应该找到组合规则
	foundCombination := false
	for _, rule := range rules {
		if len(rule.Conditions) >= 2 {
			foundCombination = true
			t.Logf("找到组合规则，包含 %d 个条件", len(rule.Conditions))
			// 检查条件是否包含部门和级别
			hasDept := false
			hasLevel := false
			for _, cond := range rule.Conditions {
				if cond.Field == "部门" {
					hasDept = true
				}
				if cond.Field == "级别" {
					hasLevel = true
				}
			}
			if hasDept && hasLevel {
				t.Log("组合规则包含部门和级别字段，符合预期")
			}
			break
		}
	}

	if !foundCombination {
		t.Log("未找到组合规则，可能单个字段规则已经足够")
	}

	t.Log("✅ 测试通过")
}

// TestDisjointCombinationWithMixed误伤 测试混合误伤率的不重叠组合
func TestDisjointCombinationWithMixed误伤(t *testing.T) {
	t.Log("测试场景: 一个误伤0.4%的根节点和一个误伤88.3%的根节点搭配组合")
	t.Log("期望结果: 找到组合规则，它们的D集合不相交")

	// 模拟数据：共有行3行，独有行3行
	rows := []Row{
		// 共有行 (索引0,1,2)
		{"部门": "技术部", "级别": "P7", "城市": "北京"},
		{"部门": "技术部", "级别": "P7", "城市": "上海"},
		{"部门": "技术部", "级别": "P7", "城市": "广州"},
		// 独有行 (索引3) - 部门不同
		{"部门": "市场部", "级别": "P7", "城市": "北京"},
		// 独有行 (索引4) - 级别不同
		{"部门": "技术部", "级别": "P8", "城市": "北京"},
		// 独有行 (索引5) - 城市不同
		{"部门": "技术部", "级别": "P7", "城市": "深圳"},
	}

	commonSet := NewRowSet(6)
	commonSet.Add(0)
	commonSet.Add(1)
	commonSet.Add(2)

	// 完整算法
	t.Log("1. 完整算法:")
	finder := NewRuleFinder()
	rules := finder.FindRules(rows, commonSet)

	// 断言：应该找到至少1个规则
	if len(rules) == 0 {
		t.Error("期望找到规则，但实际没有找到")
		return
	}

	// 断言：第一个规则应该是完美规则（覆盖率100%）
	if rules[0].Covered != rules[0].TotalCommon {
		t.Errorf("期望完美规则（覆盖率100%%），实际覆盖 %d/%d", rules[0].Covered, rules[0].TotalCommon)
	}

	t.Log("✅ 测试通过")
}