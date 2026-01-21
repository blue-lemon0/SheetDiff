package main

import "fmt"

// RunAnalysis 主分析流程
func RunAnalysis(excelFile string) error {
	// 1. 打开Excel文件
	f, err := OpenExcel(excelFile)
	if err != nil {
		return fmt.Errorf("打开Excel失败: %w", err)
	}
	defer f.Close()

	// 2. 读取配置
	config, err := ReadConfig(f)
	if err != nil {
		return fmt.Errorf("读取配置失败: %w", err)
	}

	// 3. 读取主表数据
	mainData, mainHeaders, err := LoadSheetData(f, config.MainSheet, config.HeaderRow)
	if err != nil {
		return fmt.Errorf("读取主表失败: %w", err)
	}

	// 4. 读取参考表数据
	refData, refHeaders, err := LoadSheetData(f, config.RefSheet, config.RefHeaderRow)
	if err != nil {
		return fmt.Errorf("读取参考表失败: %w", err)
	}

	// 5. 主键匹配
	result := MatchByKeys(mainData, refData, config)

	// 6. 为主表和参考表中匹配的行的主键单元格添加绿色背景
	if err := HighlightMatchedKeyCells(f, result, config); err != nil {
		return fmt.Errorf("高亮匹配的主键单元格失败: %w", err)
	}

	// 7. 核心算法：分析过滤规则（过滤条件链 + 森林 + 完美规则）
	mainRuleResult, refRuleResult := AnalyzeRulesForBothSheets(result, mainData, refData,
		mainHeaders, refHeaders, config.MainKeys, config.RefKeys)

	// 8. 输出结果
	if err := WriteResults(f, result,
		mainRuleResult, refRuleResult,
		mainHeaders, refHeaders, len(mainData), len(refData)); err != nil {
		return fmt.Errorf("写入结果失败: %w", err)
	}

	// 9. 保存文件
	if err := f.Save(); err != nil {
		return fmt.Errorf("保存文件失败: %w", err)
	}

	return nil
}
