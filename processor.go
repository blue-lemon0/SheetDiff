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

	// 6. 差异分析
	mainOnlyRules, mainCommonRules, refOnlyRules, refCommonRules := AnalyzeDifferences(result, config)

	// 7. 分析过滤字段（简单统计）
	mainFields, refFields := AnalyzeFilterFields(mainData, refData, mainHeaders, refHeaders,
		config.MainKeys, config.RefKeys, config.FieldMappings)

	// 8. 核心算法：分析过滤规则（字段链条）
	mainChains, refChains := AnalyzeRulesForBothSheets(result, mainData, refData,
		mainHeaders, refHeaders, config.MainKeys, config.RefKeys)

	// 9. 输出结果
	if err := WriteResults(f, result,
		mainOnlyRules, mainCommonRules, refOnlyRules, refCommonRules,
		mainFields, refFields,
		mainChains, refChains,
		mainHeaders, refHeaders, len(mainData), len(refData)); err != nil {
		return fmt.Errorf("写入结果失败: %w", err)
	}

	// 9. 保存文件
	if err := f.Save(); err != nil {
		return fmt.Errorf("保存文件失败: %w", err)
	}

	return nil
}
