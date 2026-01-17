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
	fmt.Println("🔧 开始读取配置...")

	rows, err := f.GetRows("配置")
	if err != nil {
		fmt.Printf("❌ 读取'sheet失败: %v\n", err)
		return Config{}, fmt.Errorf("找不到名为'配置'的sheet")
	}

	fmt.Printf("✅ 配置sheet读取成功，共%d行\n", len(rows))

	// 打印所有行查看
	for i, row := range rows {
		fmt.Printf("  行%d: ", i+1)
		for j, cell := range row {
			if j > 0 {
				fmt.Printf(" | ")
			}
			fmt.Printf("[%d]'%s'", j, cell)
		}
		fmt.Println()
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
			fmt.Printf("🔧 第%d行: 检测到空行，进入字段映射区域\n", i+1)
			inMappingSection = true
			mappingHeaderRow = i + 1 // 下一行是表头
			continue
		}

		if inMappingSection && i == mappingHeaderRow {
			fmt.Printf("🔧 第%d行: 字段映射表头行，跳过\n", i+1)
			continue
		}

		if inMappingSection && i > mappingHeaderRow {
			// 字段映射数据行
			if len(row) >= 2 {
				mainField := strings.TrimSpace(row[0])
				refField := strings.TrimSpace(row[1])

				fmt.Printf("🔧 第%d行: 字段映射行，主表字段='%s', 参考表字段='%s'\n",
					i+1, mainField, refField)

				if mainField != "" && refField != "" {
					// 判断是否主键：只认"是"
					isKey := false
					if len(row) >= 3 {
						keyFlag := strings.TrimSpace(row[2])
						fmt.Printf("   是否主键列值: '%s'\n", keyFlag)
						if keyFlag == "是" {
							isKey = true
							fmt.Printf("   ✅ 标记为主键\n")
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
				} else {
					fmt.Printf("   ⚠️ 主表字段或参考表字段为空，跳过\n")
				}
			} else {
				fmt.Printf("🔧 第%d行: 列数不足%d列，跳过\n", i+1, len(row))
			}
		} else {
			// 基础配置区域
			if len(row) >= 2 {
				key := strings.TrimSpace(row[0])
				value := strings.TrimSpace(row[1])

				fmt.Printf("🔧 第%d行: 配置项 '%s' = '%s'\n", i+1, key, value)

				switch key {
				case "主表":
					config.MainSheet = value
					fmt.Printf("   设置主表sheet为: %s\n", value)
				case "参考表":
					config.RefSheet = value
					fmt.Printf("   设置参考表sheet为: %s\n", value)
				case "主表表头行":
					if n, err := strconv.Atoi(value); err == nil {
						config.HeaderRow = n
						fmt.Printf("   设置主表表头行: %d\n", n)
					} else {
						fmt.Printf("   ⚠️ 主表表头行 '%s' 不是有效数字\n", value)
					}
				case "参考表表头行\n":
					if n, err := strconv.Atoi(value); err == nil {
						config.RefHeaderRow = n
						fmt.Printf("   设置参考表表头行\n: %d\n", n)
					} else {
						fmt.Printf("   ⚠️ 参考表表头行\n值 '%s' 不是有效数字\n", value)
					}
				}
			} else {
				fmt.Printf("🔧 第%d行: 基础配置区域但列数不足，跳过\n", i+1)
			}
		}
	}

	// 验证
	if len(config.MainKeys) == 0 {
		fmt.Printf("❌ 配置错误：没有指定主键字段\n")
		return Config{}, fmt.Errorf("配置错误：没有指定主键字段（请在'是否主键'列填写'是'）")
	}

	if len(config.FieldMappings) == 0 {
		fmt.Printf("❌ 配置错误：没有配置字段映射\n")
		return Config{}, fmt.Errorf("配置错误：没有配置字段映射")
	}

	fmt.Printf("✅ 配置解析完成:\n")
	fmt.Printf("  主表: %s (跳过%d行)\n", config.MainSheet, config.HeaderRow)
	fmt.Printf("  参考表: %s (跳过%d行)\n", config.RefSheet, config.RefHeaderRow)
	fmt.Printf("  主键: %v ↔ %v\n", config.MainKeys, config.RefKeys)
	fmt.Printf("  字段映射: %d 个字段\n", len(config.FieldMappings))
	for i, m := range config.FieldMappings {
		keyMark := ""
		if m.IsKey {
			keyMark = " (主键)"
		}
		fmt.Printf("    %d. %s ↔ %s%s\n", i+1, m.MainField, m.RefField, keyMark)
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
	fmt.Printf("🔧 开始加载sheet '%s' (表头在第%d行)\n", sheet, headerRow)

	rows, err := f.GetRows(sheet)
	if err != nil {
		return nil, nil, fmt.Errorf("读取sheet %s 失败: %w", sheet, err)
	}

	if len(rows) < headerRow {
		return []Row{}, []string{}, nil
	}

	// 表头行（0-based索引）
	headers := rows[headerRow-1]
	fmt.Printf("🔧 sheet '%s' 表头行(第%d行): %v\n", sheet, headerRow, headers)

	for i, h := range headers {
		headers[i] = strings.TrimSpace(h)
	}

	fmt.Printf("🔧 处理后表头: %v\n", headers)

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
			// 打印第一行数据查看
			if i == headerRow {
				fmt.Printf("🔧 sheet '%s' 第一行数据(第%d行): %v\n", sheet, i+1, row)
			}
		}
	}

	fmt.Printf("✅ sheet '%s' 加载完成: %d行数据\n", sheet, len(data))
	return data, headers, nil
}

// ... 前面的代码保持不变 ...

// WriteResults 写入分析结果
func WriteResults(f *excelize.File, result MatchResult,
	mainOnlyRules, mainCommonRules, refOnlyRules, refCommonRules []Rule,
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

	// 写入规则（从第12行开始）
	ruleEndRow := writeAllRules(f, mainOnlyRules, mainCommonRules, refOnlyRules, refCommonRules, 12)

	// 写入详细差异样本（从规则结束行+2开始）
	writeDetails(f, result, mainHeaders, refHeaders, ruleEndRow+2)

	// 设置活动sheet
	f.SetActiveSheet(index)

	return nil
}

// writeAllRules 写入所有规则，返回最后使用的行号
func writeAllRules(f *excelize.File, mainOnlyRules, mainCommonRules, refOnlyRules, refCommonRules []Rule, startRow int) int {
	row := startRow

	// 写入主表独有行规则
	if len(mainOnlyRules) > 0 {
		f.SetCellValue("分析结果", fmt.Sprintf("A%d", row), "【主表-独有行特征】")
		row++

		// 表头
		f.SetCellValue("分析结果", fmt.Sprintf("A%d", row), "字段")
		f.SetCellValue("分析结果", fmt.Sprintf("B%d", row), "动作")
		f.SetCellValue("分析结果", fmt.Sprintf("C%d", row), "值/特征")
		f.SetCellValue("分析结果", fmt.Sprintf("D%d", row), "取值列表")
		row++

		// 数据行
		for _, rule := range mainOnlyRules {
			f.SetCellValue("分析结果", fmt.Sprintf("A%d", row), rule.Field)
			f.SetCellValue("分析结果", fmt.Sprintf("B%d", row), rule.Action)
			f.SetCellValue("分析结果", fmt.Sprintf("C%d", row), rule.Pattern)

			// 写入所有取值（从D列开始）
			for i, val := range rule.Values {
				col, _ := excelize.ColumnNumberToName(i + 4) // D列=4
				f.SetCellValue("分析结果", fmt.Sprintf("%s%d", col, row), val)
			}
			row++
		}

		row++ // 空一行
	}

	// 写入主表共有行规则
	if len(mainCommonRules) > 0 {
		f.SetCellValue("分析结果", fmt.Sprintf("A%d", row), "【主表-共有行特征】")
		row++

		// 表头
		f.SetCellValue("分析结果", fmt.Sprintf("A%d", row), "字段")
		f.SetCellValue("分析结果", fmt.Sprintf("B%d", row), "动作")
		f.SetCellValue("分析结果", fmt.Sprintf("C%d", row), "值/特征")
		f.SetCellValue("分析结果", fmt.Sprintf("D%d", row), "取值列表")
		row++

		// 数据行
		for _, rule := range mainCommonRules {
			f.SetCellValue("分析结果", fmt.Sprintf("A%d", row), rule.Field)
			f.SetCellValue("分析结果", fmt.Sprintf("B%d", row), rule.Action)
			f.SetCellValue("分析结果", fmt.Sprintf("C%d", row), rule.Pattern)

			// 写入所有取值（从D列开始）
			for i, val := range rule.Values {
				col, _ := excelize.ColumnNumberToName(i + 4) // D列=4
				f.SetCellValue("分析结果", fmt.Sprintf("%s%d", col, row), val)
			}
			row++
		}

		row++ // 空一行
	}

	// 写入参考表独有行规则
	if len(refOnlyRules) > 0 {
		f.SetCellValue("分析结果", fmt.Sprintf("A%d", row), "【参考表-独有行特征】")
		row++

		// 表头
		f.SetCellValue("分析结果", fmt.Sprintf("A%d", row), "字段")
		f.SetCellValue("分析结果", fmt.Sprintf("B%d", row), "动作")
		f.SetCellValue("分析结果", fmt.Sprintf("C%d", row), "值/特征")
		f.SetCellValue("分析结果", fmt.Sprintf("D%d", row), "取值列表")
		row++

		// 数据行
		for _, rule := range refOnlyRules {
			f.SetCellValue("分析结果", fmt.Sprintf("A%d", row), rule.Field)
			f.SetCellValue("分析结果", fmt.Sprintf("B%d", row), rule.Action)
			f.SetCellValue("分析结果", fmt.Sprintf("C%d", row), rule.Pattern)

			// 写入所有取值（从D列开始）
			for i, val := range rule.Values {
				col, _ := excelize.ColumnNumberToName(i + 4) // D列=4
				f.SetCellValue("分析结果", fmt.Sprintf("%s%d", col, row), val)
			}
			row++
		}

		row++ // 空一行
	}

	// 写入参考表共有行规则
	if len(refCommonRules) > 0 {
		f.SetCellValue("分析结果", fmt.Sprintf("A%d", row), "【参考表-共有行特征】")
		row++

		// 表头
		f.SetCellValue("分析结果", fmt.Sprintf("A%d", row), "字段")
		f.SetCellValue("分析结果", fmt.Sprintf("B%d", row), "动作")
		f.SetCellValue("分析结果", fmt.Sprintf("C%d", row), "值/特征")
		f.SetCellValue("分析结果", fmt.Sprintf("D%d", row), "取值列表")
		row++

		// 数据行
		for _, rule := range refCommonRules {
			f.SetCellValue("分析结果", fmt.Sprintf("A%d", row), rule.Field)
			f.SetCellValue("分析结果", fmt.Sprintf("B%d", row), rule.Action)
			f.SetCellValue("分析结果", fmt.Sprintf("C%d", row), rule.Pattern)

			// 写入所有取值（从D列开始）
			for i, val := range rule.Values {
				col, _ := excelize.ColumnNumberToName(i + 4) // D列=4
				f.SetCellValue("分析结果", fmt.Sprintf("%s%d", col, row), val)
			}
			row++
		}
	}

	return row
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

// writeAnalysis 写入差异分析
func writeAnalysis(f *excelize.File, analysis []DiffAnalysis) {
	startRow := 12
	f.SetCellValue("分析结果", fmt.Sprintf("A%d", startRow), "🔍 差异原因分析")
	f.SetCellValue("分析结果", fmt.Sprintf("A%d", startRow+1), "字段")
	f.SetCellValue("分析结果", fmt.Sprintf("B%d", startRow+1), "可疑取值")
	f.SetCellValue("分析结果", fmt.Sprintf("C%d", startRow+1), "独有行占比")
	f.SetCellValue("分析结果", fmt.Sprintf("D%d", startRow+1), "匹配行占比")
	f.SetCellValue("分析结果", fmt.Sprintf("E%d", startRow+1), "影响度")
	f.SetCellValue("分析结果", fmt.Sprintf("F%d", startRow+1), "类型")

	for i, item := range analysis {
		row := startRow + 2 + i
		f.SetCellValue("分析结果", fmt.Sprintf("A%d", row), item.Field)
		f.SetCellValue("分析结果", fmt.Sprintf("B%d", row), item.Value)
		f.SetCellValue("分析结果", fmt.Sprintf("C%d", row), fmt.Sprintf("%.1f%%", item.OnlyMainPct*100))
		f.SetCellValue("分析结果", fmt.Sprintf("D%d", row), fmt.Sprintf("%.1f%%", item.MatchedPct*100))
		f.SetCellValue("分析结果", fmt.Sprintf("E%d", row), fmt.Sprintf("%.2f", item.Impact))

		typeText := "仅主表有"
		if item.Type == "only_ref" {
			typeText = "仅参考表有"
		}
		f.SetCellValue("分析结果", fmt.Sprintf("F%d", row), typeText)
	}
}

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

// 辅助函数
func splitKeys(keys string) []string {
	parts := strings.FieldsFunc(keys, func(r rune) bool {
		return r == ',' || r == '，' || r == ';' || r == '；'
	})

	result := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

func isEmptyRow(row Row) bool {
	for _, v := range row {
		if v != "" {
			return false
		}
	}
	return true
}
