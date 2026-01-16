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
func WriteResults(f *excelize.File, result MatchResult, analysis []DiffAnalysis, mainHeaders, refHeaders []string) error {
	// 删除已存在的"分析结果"sheet
	f.DeleteSheet("分析结果")

	// 创建新的"分析结果"sheet
	index, err := f.NewSheet("分析结果")
	if err != nil {
		return fmt.Errorf("创建sheet失败: %w", err)
	}

	// 写入统计信息
	writeSummary(f, result)

	// 写入差异分析
	writeAnalysis(f, analysis)

	// 写入详细差异
	writeDetails(f, result, mainHeaders, refHeaders)

	// 设置活动sheet
	f.SetActiveSheet(index)

	return nil
}

// writeSummary 写入统计信息
func writeSummary(f *excelize.File, result MatchResult) {
	f.SetCellValue("分析结果", "A1", "📊 数据比对统计")
	f.SetCellValue("分析结果", "A2", "项目")
	f.SetCellValue("分析结果", "B2", "数量")

	data := [][]interface{}{
		{"总匹配行数", len(result.Matched)},
		{"仅主表有", len(result.OnlyMain)},
		{"仅参考表有", len(result.OnlyRef)},
	}

	for i, row := range data {
		f.SetCellValue("分析结果", fmt.Sprintf("A%d", i+3), row[0])
		f.SetCellValue("分析结果", fmt.Sprintf("B%d", i+3), row[1])
	}

	// 样式
	f.SetCellValue("分析结果", "A8", "说明：")
	f.SetCellValue("分析结果", "A9", "主表：用户提供的主数据表")
	f.SetCellValue("分析结果", "A10", "参考表：用于比对的参考数据表")
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

// writeDetails 写入详细差异
func writeDetails(f *excelize.File, result MatchResult, mainHeaders, refHeaders []string) {
	startRow := 12 + len(result.Matched) + 5

	f.SetCellValue("分析结果", fmt.Sprintf("A%d", startRow), "📝 详细差异（前20行）")
	f.SetCellValue("分析结果", fmt.Sprintf("A%d", startRow+1), "主键")
	f.SetCellValue("分析结果", fmt.Sprintf("B%d", startRow+1), "所在表")
	f.SetCellValue("分析结果", fmt.Sprintf("C%d", startRow+1), "差异字段")

	// 写入仅主表有的行
	rowNum := startRow + 2
	count := 0
	for _, row := range result.OnlyMain {
		if count >= 10 {
			break
		}
		// 构建主键显示
		keyParts := make([]string, 0)
		for _, v := range row {
			if len(keyParts) < 3 { // 只显示前3个字段
				keyParts = append(keyParts, v)
			}
		}
		key := strings.Join(keyParts, "|")

		f.SetCellValue("分析结果", fmt.Sprintf("A%d", rowNum), key)
		f.SetCellValue("分析结果", fmt.Sprintf("B%d", rowNum), "仅主表有")
		rowNum++
		count++
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
