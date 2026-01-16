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
	// 尝试不同的sheet名
	sheetNames := []string{"配置", "config", "Config", "设置"}
	var rows [][]string
	var err error

	for _, name := range sheetNames {
		rows, err = f.GetRows(name)
		if err == nil && len(rows) > 0 {
			break
		}
	}

	if err != nil || len(rows) == 0 {
		return Config{}, fmt.Errorf("找不到配置sheet")
	}

	config := Config{
		MainSheet: "主表",
		RefSheet:  "参考表",
		MainSkip:  1,
		RefSkip:   1,
	}

	for _, row := range rows {
		if len(row) < 2 {
			continue
		}

		key := strings.TrimSpace(row[0])
		value := strings.TrimSpace(row[1])

		switch key {
		case "主表名称", "主表":
			config.MainSheet = value
		case "参考表名称", "参考表":
			config.RefSheet = value
		case "主表主键列", "主表主键":
			config.MainKeys = splitKeys(value)
		case "参考表主键列", "参考表主键":
			config.RefKeys = splitKeys(value)
		case "主表跳过行数", "主表跳行":
			if n, err := strconv.Atoi(value); err == nil {
				config.MainSkip = n
			}
		case "参考表跳过行数", "参考表跳行":
			if n, err := strconv.Atoi(value); err == nil {
				config.RefSkip = n
			}
		}
	}

	// 默认主键（如果没配置）
	if len(config.MainKeys) == 0 {
		config.MainKeys = []string{"ID"}
	}
	if len(config.RefKeys) == 0 {
		config.RefKeys = []string{"ID"}
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
