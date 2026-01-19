package main

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/xuri/excelize/v2"
)

// OpenExcel 打开Excel文件
func OpenExcel(filename string) (*excelize.File, error) {
	f, err := excelize.OpenFile(filename)
	if err != nil {
		return nil, fmt.Errorf("无法打开文件 %s: %w", filename, err)
	}
	return f, nil
}

// ReadConfig 从"配置"sheet读取配置
func ReadConfig(f *excelize.File) (Config, error) {
	rows, err := f.GetRows("配置")
	if err != nil {
		return Config{}, fmt.Errorf("找不到名为'配置'的sheet")
	}

	config := Config{
		MainSheet:     "主表",
		RefSheet:      "参考表",
		HeaderRow:     1,
		RefHeaderRow:  1,
		FieldMappings: []FieldMapping{},
		MainKeys:      []string{},
		RefKeys:       []string{},
	}

	inMappingSection := false
	mappingHeaderRow := -1

	for i, row := range rows {
		// 检查是否空行（作为分区标记）
		if !inMappingSection && isEmptyConfigRow(row) {
			inMappingSection = true
			mappingHeaderRow = i + 1 // 下一行是表头
			continue
		}

		if inMappingSection && i == mappingHeaderRow {
			continue
		}

		if inMappingSection && i > mappingHeaderRow {
			// 字段映射数据行
			if len(row) >= 2 {
				mainField := strings.TrimSpace(row[0])
				refField := strings.TrimSpace(row[1])

				if mainField != "" && refField != "" {
					// 判断是否主键：只认"是"
					isKey := false
					if len(row) >= 3 {
						keyFlag := strings.TrimSpace(row[2])
						if keyFlag == "是" {
							isKey = true
						}
					}

					mapping := FieldMapping{
						MainField: mainField,
						RefField:  refField,
						IsKey:     isKey,
					}
					config.FieldMappings = append(config.FieldMappings, mapping)

					if isKey {
						config.MainKeys = append(config.MainKeys, mainField)
						config.RefKeys = append(config.RefKeys, refField)
					}
				}
			}
		} else {
			// 基础配置区域
			if len(row) >= 2 {
				key := strings.TrimSpace(row[0])
				value := strings.TrimSpace(row[1])

				switch key {
				case "主表":
					config.MainSheet = value
				case "参考表":
					config.RefSheet = value
				case "主表表头行":
					if n, err := strconv.Atoi(value); err == nil {
						config.HeaderRow = n
					}
				case "参考表表头行":
					if n, err := strconv.Atoi(value); err == nil {
						config.RefHeaderRow = n
					}
				}
			}
		}
	}

	// 验证
	if len(config.MainKeys) == 0 {
		return Config{}, fmt.Errorf("配置错误：没有指定主键字段（请在'是否主键'列填写'是'）")
	}

	if len(config.FieldMappings) == 0 {
		return Config{}, fmt.Errorf("配置错误：没有配置字段映射")
	}

	return config, nil
}

// isEmptyConfigRow 判断配置行是否为空
func isEmptyConfigRow(row []string) bool {
	for _, cell := range row {
		if strings.TrimSpace(cell) != "" {
			return false
		}
	}
	return true
}

// LoadSheetData 读取指定sheet的数据
func LoadSheetData(f *excelize.File, sheet string, headerRow int) ([]Row, []string, error) {
	// headerRow : 表头所在行号（Excel行号，从1开始）
	rows, err := f.GetRows(sheet)
	if err != nil {
		return nil, nil, fmt.Errorf("读取sheet %s 失败: %w", sheet, err)
	}

	if len(rows) < headerRow {
		return []Row{}, []string{}, nil
	}

	// 表头行（0-based索引）
	headers := rows[headerRow-1]

	for i, h := range headers {
		headers[i] = strings.TrimSpace(h)
	}

	// 读取数据行（从表头的下一行开始）
	data := make([]Row, 0)
	for i := headerRow; i < len(rows); i++ {
		row := make(Row)
		for j, cell := range rows[i] {
			if j < len(headers) && headers[j] != "" {
				row[headers[j]] = strings.TrimSpace(cell)
			}
		}
		// 跳过全空的行
		if !isEmptyRow(row) {
			data = append(data, row)
		}
	}

	return data, headers, nil
}

// WriteResults 写入分析结果
func WriteResults(f *excelize.File, result MatchResult,
	mainRuleResult, refRuleResult RuleAnalysisResult,
	mainHeaders, refHeaders []string, mainCount, refCount int) error {
	// 删除已存在的"分析结果"sheet
	f.DeleteSheet("分析结果")

	// 创建新的"分析结果"sheet
	index, err := f.NewSheet("分析结果")
	if err != nil {
		return fmt.Errorf("创建sheet失败: %w", err)
	}

	// 写入统计信息（第1-10行）
	writeSummary(f, result, mainCount, refCount)

	// 写入核心算法分析结果（从第12行开始）
	chainEndRow := writeRuleAnalysis(f, mainRuleResult, refRuleResult, result, mainCount, refCount, 12)

	// 写入详细差异样本（从链条分析结束行+2开始）
	writeDetails(f, result, mainHeaders, refHeaders, chainEndRow+2)

	// 设置活动sheet
	f.SetActiveSheet(index)

	return nil
}

// writeRuleAnalysis 写入完整规则分析结果（链条 + 森林 + 完美规则）
func writeRuleAnalysis(f *excelize.File, mainResult, refResult RuleAnalysisResult,
	matchResult MatchResult, mainCount, refCount int, startRow int) int {
	row := startRow

	// 计算共有行和独有行数量
	commonCount := len(matchResult.Matched)
	mainOnlyCount := len(matchResult.OnlyMain)
	refOnlyCount := len(matchResult.OnlyRef)

	// ========== 主表独有行分析 ==========
	f.SetCellValue("分析结果", fmt.Sprintf("A%d", row), "【主表独有行过滤条件分析】")
	setHeaderStyle(f, "分析结果", fmt.Sprintf("A%d", row))
	row++

	f.SetCellValue("分析结果", fmt.Sprintf("A%d", row),
		fmt.Sprintf("基于共有行(%d行) vs 独有行(%d行)分析", commonCount, mainOnlyCount))
	row++
	row++ // 空一行

	// 输出主表的完美规则（最优）
	if len(mainResult.PerfectRules) > 0 {
		f.SetCellValue("分析结果", fmt.Sprintf("A%d", row), "✨ 推荐过滤条件组合 ✨")
		setBoldStyle(f, "分析结果", fmt.Sprintf("A%d", row))
		row++
		row = writePerfectRules(f, mainResult.PerfectRules, commonCount, mainOnlyCount, row)
		row += 2 // 空两行
	}

	// 输出主表的森林结构
	if len(mainResult.Forest) > 0 {
		f.SetCellValue("分析结果", fmt.Sprintf("A%d", row), "🌲 支配关系森林 🌲")
		setBoldStyle(f, "分析结果", fmt.Sprintf("A%d", row))
		row++
		f.SetCellValue("分析结果", fmt.Sprintf("A%d", row),
			fmt.Sprintf("（根节点%d个，D集合更小的节点支配D集合更大的节点）", len(mainResult.Forest)))
		row++
		row = writeForest(f, mainResult.Forest, commonCount, mainOnlyCount, row)
		row += 2 // 空两行
	}

	// 输出主表的过滤条件链详情
	if len(mainResult.Chains) > 0 {
		f.SetCellValue("分析结果", fmt.Sprintf("A%d", row), "🔗 过滤条件链详情 🔗")
		setBoldStyle(f, "分析结果", fmt.Sprintf("A%d", row))
		row++
		for _, chain := range mainResult.Chains {
			row = writeChainDetail(f, chain, mainResult.NodeToTree, commonCount, mainOnlyCount, row)
			row++ // 链条之间空一行
		}
	} else {
		f.SetCellValue("分析结果", fmt.Sprintf("A%d", row), "未找到有效的过滤规律")
		row++
	}

	row += 2 // 空两行

	// ========== 参考表独有行分析 ==========
	f.SetCellValue("分析结果", fmt.Sprintf("A%d", row), "【参考表独有行过滤条件分析】")
	setHeaderStyle(f, "分析结果", fmt.Sprintf("A%d", row))
	row++

	f.SetCellValue("分析结果", fmt.Sprintf("A%d", row),
		fmt.Sprintf("基于共有行(%d行) vs 独有行(%d行)分析", commonCount, refOnlyCount))
	row++
	row++ // 空一行

	// 输出参考表的完美规则（最优）
	if len(refResult.PerfectRules) > 0 {
		f.SetCellValue("分析结果", fmt.Sprintf("A%d", row), "✨ 推荐过滤条件组合 ✨")
		setBoldStyle(f, "分析结果", fmt.Sprintf("A%d", row))
		row++
		row = writePerfectRules(f, refResult.PerfectRules, commonCount, refOnlyCount, row)
		row += 2 // 空两行
	}

	// 输出参考表的森林结构
	if len(refResult.Forest) > 0 {
		f.SetCellValue("分析结果", fmt.Sprintf("A%d", row), "🌲 支配关系森林 🌲")
		setBoldStyle(f, "分析结果", fmt.Sprintf("A%d", row))
		row++
		f.SetCellValue("分析结果", fmt.Sprintf("A%d", row),
			fmt.Sprintf("（根节点%d个，D集合更小的节点支配D集合更大的节点）", len(refResult.Forest)))
		row++
		row = writeForest(f, refResult.Forest, commonCount, refOnlyCount, row)
		row += 2 // 空两行
	}

	// 输出参考表的过滤条件链详情
	if len(refResult.Chains) > 0 {
		f.SetCellValue("分析结果", fmt.Sprintf("A%d", row), "🔗 过滤条件链详情 🔗")
		setBoldStyle(f, "分析结果", fmt.Sprintf("A%d", row))
		row++
		for _, chain := range refResult.Chains {
			row = writeChainDetail(f, chain, refResult.NodeToTree, commonCount, refOnlyCount, row)
			row++ // 链条之间空一行
		}
	} else {
		f.SetCellValue("分析结果", fmt.Sprintf("A%d", row), "未找到有效的过滤规律")
		row++
	}

	return row
}

// writePerfectRules 写入完美规则组合
func writePerfectRules(f *excelize.File, rules []PerfectRule, commonCount, onlyCount int, startRow int) int {
	row := startRow

	for idx, rule := range rules {
		// 规则标题
		f.SetCellValue("分析结果", fmt.Sprintf("A%d", row),
			fmt.Sprintf("方案 %d: %d个条件组合", idx+1, len(rule.Conditions)))
		setBoldStyle(f, "分析结果", fmt.Sprintf("A%d", row))
		row++

		// 表头
		f.SetCellValue("分析结果", fmt.Sprintf("A%d", row), "字段")
		f.SetCellValue("分析结果", fmt.Sprintf("B%d", row), "条件")
		f.SetCellValue("分析结果", fmt.Sprintf("C%d", row), "说明")
		setHeaderStyle(f, "分析结果", fmt.Sprintf("A%d:C%d", row, row))
		row++

		// 总D集合大小（完美规则的D集合互不相交，因此总D大小为0或各条件D之和）
		totalDSize := 0

		// 写入每个条件
		for _, cond := range rule.Conditions {
			condition := formatCondition(cond.Field, cond.Values)

			f.SetCellValue("分析结果", fmt.Sprintf("A%d", row), cond.Field)
			f.SetCellValue("分析结果", fmt.Sprintf("B%d", row), condition)
			f.SetCellValue("分析结果", fmt.Sprintf("C%d", row), "覆盖所有共有行")
			row++
		}

		// 如果总D为0，高亮显示
		if totalDSize == 0 {
			f.SetCellValue("分析结果", fmt.Sprintf("A%d", row), "💚 完美匹配：不误伤任何独有行！")
			setGreenStyle(f, "分析结果", fmt.Sprintf("A%d", row))
			row++
		}

		row++ // 方案之间空一行
	}

	return row
}

// writeForest 写入支配关系森林
func writeForest(f *excelize.File, forest []*Tree, commonCount, onlyCount int, startRow int) int {
	row := startRow

	// 森林概览
	f.SetCellValue("分析结果", fmt.Sprintf("A%d", row), fmt.Sprintf("支配关系森林概览: 共%d棵树", len(forest)))
	setBoldStyle(f, "分析结果", fmt.Sprintf("A%d", row))
	row++
	row++

	for treeIdx, tree := range forest {
		// 树标题
		rootDSize := tree.Root.D.Len()
		f.SetCellValue("分析结果", fmt.Sprintf("A%d", row),
			fmt.Sprintf("🌲 树 %d: 根节点 %s (误伤独有行数=%d)", treeIdx+1, tree.Root.Field, rootDSize))
		setBoldStyle(f, "分析结果", fmt.Sprintf("A%d", row))
		row++

		// 根节点信息
		f.SetCellValue("分析结果", fmt.Sprintf("B%d", row), "条件:")
		f.SetCellValue("分析结果", fmt.Sprintf("C%d", row), formatCondition(tree.Root.Field, tree.Root.Values))
		row++

		hit := float64(rootDSize) / float64(onlyCount) * 100
		f.SetCellValue("分析结果", fmt.Sprintf("B%d", row),
			fmt.Sprintf("误伤独有行数=%d, 误伤率: %.1f%%", rootDSize, hit))
		row++

		// 根节点推荐度
		stars := getRecommendation(rootDSize, hit, true)
		f.SetCellValue("分析结果", fmt.Sprintf("B%d", row), fmt.Sprintf("推荐度: %s", stars))
		if rootDSize == 0 {
			setGreenStyle(f, "分析结果", fmt.Sprintf("B%d:C%d", row, row))
		}
		row++

		// 子节点信息（详细层次结构）
		if len(tree.Children) > 0 {
			f.SetCellValue("分析结果", fmt.Sprintf("B%d", row), fmt.Sprintf("📋 被支配节点: %d个", len(tree.Children)))
			setBoldStyle(f, "分析结果", fmt.Sprintf("B%d", row))
			row++

			// 递归写入子树
			row = writeTreeChildren(f, tree.Children, commonCount, onlyCount, row, 2)
		}

		row++ // 树之间空一行
	}

	return row
}

// writeTreeChildren 递归写入树的子节点
func writeTreeChildren(f *excelize.File, children []*Tree, commonCount, onlyCount int, startRow, indent int) int {
	row := startRow

	for _, child := range children {
		childDSize := child.Root.D.Len()
		hit := float64(childDSize) / float64(onlyCount) * 100
		stars := getRecommendation(childDSize, hit, false)

		// 缩进
		indentStr := strings.Repeat("  ", indent)

		// 子节点信息
		f.SetCellValue("分析结果", fmt.Sprintf("A%d", row), fmt.Sprintf("%s↳ 节点: %s", indentStr, child.Root.Field))
		f.SetCellValue("分析结果", fmt.Sprintf("B%d", row), formatCondition(child.Root.Field, child.Root.Values))
		f.SetCellValue("分析结果", fmt.Sprintf("C%d", row), fmt.Sprintf("误伤独有行数=%d, 误伤率: %.1f%%", childDSize, hit))
		f.SetCellValue("分析结果", fmt.Sprintf("D%d", row), stars)
		row++

		// 递归写入孙节点
		if len(child.Children) > 0 {
			row = writeTreeChildren(f, child.Children, commonCount, onlyCount, row, indent+1)
		}
	}

	return row
}

// writeFieldChains 写入过滤条件链分析结果（核心算法输出）- 已废弃，被writeRuleAnalysis替代
func writeFieldChains(f *excelize.File, mainChains, refChains []*FieldChain,
	result MatchResult, mainCount, refCount int, startRow int) int {
	row := startRow

	// 计算共有行和独有行数量
	commonCount := len(result.Matched)
	mainOnlyCount := len(result.OnlyMain)
	refOnlyCount := len(result.OnlyRef)

	// ========== 主表独有行分析 ==========
	f.SetCellValue("分析结果", fmt.Sprintf("A%d", row), "【主表独有行过滤条件分析】")
	setHeaderStyle(f, "分析结果", fmt.Sprintf("A%d", row))
	row++

	f.SetCellValue("分析结果", fmt.Sprintf("A%d", row),
		fmt.Sprintf("基于共有行(%d行) vs 独有行(%d行)分析", commonCount, mainOnlyCount))
	row++
	row++ // 空一行

	if len(mainChains) > 0 {
		for _, chain := range mainChains {
			row = writeChainDetail(f, chain, nil, commonCount, mainOnlyCount, row)
			row++ // 链条之间空一行
		}
	} else {
		f.SetCellValue("分析结果", fmt.Sprintf("A%d", row), "未找到有效的过滤规律")
		row++
	}

	row++ // 空一行

	// ========== 参考表独有行分析 ==========
	f.SetCellValue("分析结果", fmt.Sprintf("A%d", row), "【参考表独有行过滤条件分析】")
	setHeaderStyle(f, "分析结果", fmt.Sprintf("A%d", row))
	row++

	f.SetCellValue("分析结果", fmt.Sprintf("A%d", row),
		fmt.Sprintf("基于共有行(%d行) vs 独有行(%d行)分析", commonCount, refOnlyCount))
	row++
	row++ // 空一行

	if len(refChains) > 0 {
		for _, chain := range refChains {
			row = writeChainDetail(f, chain, nil, commonCount, refOnlyCount, row)
			row++ // 链条之间空一行
		}
	} else {
		f.SetCellValue("分析结果", fmt.Sprintf("A%d", row), "未找到有效的过滤规律")
		row++
	}

	return row
}

// writeChainDetail 写入单个过滤条件链的详细信息
func writeChainDetail(f *excelize.File, chain *FieldChain, nodeToTree map[*FieldNode]*Tree,
	commonCount, onlyCount int, startRow int) int {
	row := startRow

	// 确定链条所属的树
	treeInfo := "未归类"
	if chain.Root != nil && nodeToTree != nil {
		if tree, exists := nodeToTree[chain.Root]; exists {
			// 找到树的根节点
			current := tree
			for current.Parent != nil {
				current = current.Parent
			}
			// 这里简化处理，实际应该传递树的索引信息
			treeInfo = "已归类到某棵树"
		}
	}

	// 链条标题
	f.SetCellValue("分析结果", fmt.Sprintf("A%d", row),
		fmt.Sprintf("过滤条件链: %s (共%d个规律) - %s", chain.Field, len(chain.Nodes), treeInfo))
	setBoldStyle(f, "分析结果", fmt.Sprintf("A%d", row))
	row++

	// 表头
	f.SetCellValue("分析结果", fmt.Sprintf("A%d", row), "优先级")
	f.SetCellValue("分析结果", fmt.Sprintf("B%d", row), "规律类型")
	f.SetCellValue("分析结果", fmt.Sprintf("C%d", row), "条件")
	f.SetCellValue("分析结果", fmt.Sprintf("D%d", row), "误伤独有行数")
	f.SetCellValue("分析结果", fmt.Sprintf("E%d", row), "误伤率")
	f.SetCellValue("分析结果", fmt.Sprintf("F%d", row), "推荐度")
	f.SetCellValue("分析结果", fmt.Sprintf("G%d", row), "归属")
	setHeaderStyle(f, "分析结果", fmt.Sprintf("A%d:G%d", row, row))
	row++

	// 写入每个节点
	for i, node := range chain.Nodes {
		priority := i + 1
		if node.IsRoot {
			f.SetCellValue("分析结果", fmt.Sprintf("A%d", row), fmt.Sprintf("🎯 [%d]", priority))
		} else {
			f.SetCellValue("分析结果", fmt.Sprintf("A%d", row), fmt.Sprintf("   [%d]", priority))
		}

		// 规律类型
		ruleType := detectRuleType(node.Values)
		f.SetCellValue("分析结果", fmt.Sprintf("B%d", row), ruleType)

		// 条件
		condition := formatCondition(node.Field, node.Values)
		f.SetCellValue("分析结果", fmt.Sprintf("C%d", row), condition)

		// 误伤独有行数
		dSize := node.D.Len()
		f.SetCellValue("分析结果", fmt.Sprintf("D%d", row), dSize)

		// 误伤率
		dPct := 0.0
		if onlyCount > 0 {
			dPct = float64(dSize) / float64(onlyCount) * 100
		}
		f.SetCellValue("分析结果", fmt.Sprintf("E%d", row), fmt.Sprintf("%.1f%%", dPct))

		// 推荐度
		recommendation := getRecommendation(dSize, dPct, node.IsRoot)
		f.SetCellValue("分析结果", fmt.Sprintf("F%d", row), recommendation)

		// 归属信息
		belonging := ""
		if nodeToTree != nil {
			if tree, exists := nodeToTree[node]; exists {
				// 找到树的根节点
				current := tree
				for current.Parent != nil {
					current = current.Parent
				}
				// 使用根节点的条件作为树的标识
				treeRootCondition := formatCondition(current.Root.Field, current.Root.Values)
				if node == current.Root {
					belonging = fmt.Sprintf("树根节点 (树: %s)", treeRootCondition)
				} else {
					belonging = fmt.Sprintf("非根节点 (树: %s)", treeRootCondition)
				}
			} else {
				belonging = "未归类"
			}
		} else {
			if node.IsRoot {
				belonging = "树根节点"
			} else {
				belonging = "非根节点"
			}
		}
		f.SetCellValue("分析结果", fmt.Sprintf("G%d", row), belonging)

		// 如果是完美规律（D=0），高亮整行
		if dSize == 0 {
			setGreenStyle(f, "分析结果", fmt.Sprintf("A%d:G%d", row, row))
		}

		row++
	}

	return row
}

// detectRuleType 检测规律类型
func detectRuleType(values []string) string {
	if len(values) == 0 {
		return "未知"
	}

	// 检查是否所有值都以*结尾（前缀规律）
	allPrefix := true
	for _, v := range values {
		if !strings.HasSuffix(v, "*") {
			allPrefix = false
			break
		}
	}

	// 检查是否是数值范围（包含>=、<=等）
	hasComparator := false
	for _, v := range values {
		if strings.Contains(v, ">=") || strings.Contains(v, "<=") ||
			strings.Contains(v, ">") || strings.Contains(v, "<") {
			hasComparator = true
			break
		}
	}

	if hasComparator {
		return "数值范围"
	} else if allPrefix && len(values) > 1 {
		return "前缀枚举"
	} else if allPrefix && len(values) == 1 {
		return "前缀匹配"
	} else if len(values) > 1 {
		return "枚举"
	} else {
		return "等于"
	}
}

// formatCondition 格式化条件
func formatCondition(field string, values []string) string {
	if len(values) == 0 {
		return field
	}

	if len(values) == 1 {
		return fmt.Sprintf("%s = %s", field, values[0])
	}

	// 多个值的情况，总是显示完整的枚举值
	return fmt.Sprintf("%s ∈ {%s}", field, strings.Join(values, ", "))
}

// getRecommendation 获取推荐度
func getRecommendation(dSize int, dPct float64, isRoot bool) string {
	if dSize == 0 {
		return "⭐⭐⭐⭐⭐ 完美！"
	} else if dPct < 5 {
		return "⭐⭐⭐⭐ 优秀"
	} else if dPct < 15 {
		return "⭐⭐⭐ 良好"
	} else if dPct < 30 {
		return "⭐⭐ 一般"
	} else {
		return "⭐ 较差"
	}
}

// setHeaderStyle 设置表头样式
func setHeaderStyle(f *excelize.File, sheet, cell string) {
	style, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Size: 11},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"#D9E1F2"}, Pattern: 1},
		Alignment: &excelize.Alignment{
			Horizontal: "left",
			Vertical:   "center",
		},
	})
	f.SetCellStyle(sheet, cell, cell, style)
}

// setBoldStyle 设置加粗样式
func setBoldStyle(f *excelize.File, sheet, cell string) {
	style, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Size: 10},
	})
	f.SetCellStyle(sheet, cell, cell, style)
}

// setGreenStyle 设置绿色背景样式（完美规律）
func setGreenStyle(f *excelize.File, sheet, cellRange string) {
	style, _ := f.NewStyle(&excelize.Style{
		Fill: excelize.Fill{Type: "pattern", Color: []string{"#C6EFCE"}, Pattern: 1},
		Font: &excelize.Font{Bold: true},
	})
	f.SetCellStyle(sheet, cellRange, cellRange, style)
}

// boolToText 布尔值转文本
func boolToText(b bool) string {
	if b {
		return "是"
	}
	return "否"
}

// writeSummary 写入统计信息
func writeSummary(f *excelize.File, result MatchResult, mainCount, refCount int) {
	f.SetCellValue("分析结果", "A1", "📊 数据比对统计")
	f.SetCellValue("分析结果", "A2", "项目")
	f.SetCellValue("分析结果", "B2", "数量")
	f.SetCellValue("分析结果", "C2", "占比")

	// 主表总行数
	f.SetCellValue("分析结果", "A3", "主表总行数")
	f.SetCellValue("分析结果", "B3", mainCount)
	f.SetCellValue("分析结果", "C3", "100%")

	// 参考表总行数
	f.SetCellValue("分析结果", "A4", "参考表总行数")
	f.SetCellValue("分析结果", "B4", refCount)
	f.SetCellValue("分析结果", "C4", "100%")

	// 匹配行数
	matchRate := float64(len(result.Matched)) / float64(mainCount) * 100
	f.SetCellValue("分析结果", "A5", "匹配行数")
	f.SetCellValue("分析结果", "B5", len(result.Matched))
	f.SetCellValue("分析结果", "C5", fmt.Sprintf("%.1f%%", matchRate))

	// 仅主表有
	onlyMainRate := float64(len(result.OnlyMain)) / float64(mainCount) * 100
	f.SetCellValue("分析结果", "A6", "仅主表有")
	f.SetCellValue("分析结果", "B6", len(result.OnlyMain))
	f.SetCellValue("分析结果", "C6", fmt.Sprintf("%.1f%%", onlyMainRate))

	// 仅参考表有
	onlyRefRate := float64(len(result.OnlyRef)) / float64(refCount) * 100
	f.SetCellValue("分析结果", "A7", "仅参考表有")
	f.SetCellValue("分析结果", "B7", len(result.OnlyRef))
	f.SetCellValue("分析结果", "C7", fmt.Sprintf("%.1f%%", onlyRefRate))
}

// writeDetails 写入详细差异样本
func writeDetails(f *excelize.File, result MatchResult, mainHeaders, refHeaders []string, startRow int) {
	// startRow 由调用者传入

	// 1. 匹配数据样本
	if len(result.Matched) > 0 {
		f.SetCellValue("分析结果", fmt.Sprintf("A%d", startRow), "【匹配的数据样本】")
		startRow++

		// 写表头：显示主键列
		f.SetCellValue("分析结果", fmt.Sprintf("A%d", startRow), "匹配主键")
		startRow++

		// 写数据（最多5行）
		count := 0
		for _, pair := range result.Matched {
			if count >= 5 {
				break
			}
			f.SetCellValue("分析结果", fmt.Sprintf("A%d", startRow), pair.Key)
			startRow++
			count++
		}
		startRow++ // 空一行
	}

	// 2. 仅主表有的样本（最多5行）
	if len(result.OnlyMain) > 0 {
		f.SetCellValue("分析结果", fmt.Sprintf("A%d", startRow), "【仅主表有的数据样本】")
		startRow++

		// 写表头
		for i, h := range mainHeaders {
			cell, _ := excelize.CoordinatesToCellName(i+1, startRow)
			f.SetCellValue("分析结果", cell, h)
		}
		startRow++

		// 写数据（最多5行）
		count := 0
		for _, row := range result.OnlyMain {
			if count >= 5 {
				break
			}
			for j, h := range mainHeaders {
				cell, _ := excelize.CoordinatesToCellName(j+1, startRow)
				f.SetCellValue("分析结果", cell, row[h])
			}
			startRow++
			count++
		}
		startRow++ // 空一行
	}

	// 3. 仅参考表有的样本（最多5行）
	if len(result.OnlyRef) > 0 {
		f.SetCellValue("分析结果", fmt.Sprintf("A%d", startRow), "【仅参考表有的数据样本】")
		startRow++

		// 写表头
		for i, h := range refHeaders {
			cell, _ := excelize.CoordinatesToCellName(i+1, startRow)
			f.SetCellValue("分析结果", cell, h)
		}
		startRow++

		// 写数据（最多5行）
		count := 0
		for _, row := range result.OnlyRef {
			if count >= 5 {
				break
			}
			for j, h := range refHeaders {
				cell, _ := excelize.CoordinatesToCellName(j+1, startRow)
				f.SetCellValue("分析结果", cell, row[h])
			}
			startRow++
			count++
		}
	}
}

func isEmptyRow(row Row) bool {
	for _, v := range row {
		if v != "" {
			return false
		}
	}
	return true
}
