package main

import (
	"fmt"
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
	for i, name := range f.GetSheetList() {
		fmt.Printf("%d. %s\n", i+1, name)
	}
	// 只读名为"配置"的sheet，不尝试其他名字
	rows, err := f.GetRows("配置")
	if err != nil {
		return Config{}, fmt.Errorf("找不到名为'配置'的sheet")
	}

	if len(rows) == 0 {
		return Config{}, fmt.Errorf("'配置'sheet是空的")
	}

	// 第一行必须是标题
	if len(rows[0]) < 2 || rows[0][0] != "配置项" || rows[0][1] != "值" {
		return Config{}, fmt.Errorf("配置表格式错误，第一行必须是'配置项'和'值'")
	}

	config := Config{
		MainSheet: "主表",
		RefSheet:  "参考表",
		MainKeys:  []string{"ID"},
		RefKeys:   []string{"ID"},
	}

	// 从第二行开始读取配置
	for i := 1; i < len(rows); i++ {
		if len(rows[i]) < 2 {
			continue
		}

		key := strings.TrimSpace(rows[i][0])
		value := strings.TrimSpace(rows[i][1])

		switch key {
		case "主表":
			config.MainSheet = value
		case "参考表":
			config.RefSheet = value
		case "主表主键":
			if value != "" {
				config.MainKeys = splitKeys(value)
			}
		case "参考表主键":
			if value != "" {
				config.RefKeys = splitKeys(value)
			}
		}
	}

	return config, nil
}

// LoadSheetData 读取指定sheet的数据
func LoadSheetData(f *excelize.File, sheet string, skipRows int) ([]Row, []string, error) {
	rows, err := f.GetRows(sheet)
	if err != nil {
		return nil, nil, fmt.Errorf("读取sheet %s 失败: %w", sheet, err)
	}

	if len(rows) <= skipRows {
		return []Row{}, []string{}, nil
	}

	// 提取表头
	headers := rows[skipRows]
	for i, h := range headers {
		headers[i] = strings.TrimSpace(h)
	}

	// 读取数据行
	data := make([]Row, 0)
	for i := skipRows + 1; i < len(rows); i++ {
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
func WriteResults(f *excelize.File, result MatchResult, analysis []DiffAnalysis,
	mainHeaders, refHeaders []string, mainCount, refCount int) error {
	// 删除已存在的"分析结果"sheet
	f.DeleteSheet("分析结果")

	// 创建新的"分析结果"sheet
	index, err := f.NewSheet("分析结果")
	if err != nil {
		return fmt.Errorf("创建sheet失败: %w", err)
	}

	// 写入统计信息
	writeSummary(f, result, mainCount, refCount)

	// 写入差异分析（先保留，后面可能要删）
	// writeAnalysis(f, analysis)

	// 写入详细差异样本
	writeDetails(f, result, mainHeaders, refHeaders)

	// 设置活动sheet
	f.SetActiveSheet(index)

	return nil
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

func writeDetails(f *excelize.File, result MatchResult, mainHeaders, refHeaders []string) {
	startRow := 10

	// 1. 匹配数据样本
	if len(result.Matched) > 0 {
		f.SetCellValue("分析结果", fmt.Sprintf("A%d", startRow), "【匹配的数据样本】")
		startRow++

		// 写表头：显示主键列
		f.SetCellValue("分析结果", fmt.Sprintf("A%d", startRow), "匹配主键")
		// 如果有多个主键，可以在B列、C列继续显示
		startRow++

		// 写数据（最多5行）
		count := 0
		for _, pair := range result.Matched {
			if count >= 5 {
				break
			}
			// 显示主键（完整的，不是截断的）
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
