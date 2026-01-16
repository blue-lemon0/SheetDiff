package main

import "fmt"

// RunAnalysis 主分析流程
func RunAnalysis(excelFile string) error {
	fmt.Printf("🔍 开始分析: %s\n", excelFile)

	// 1. 打开Excel文件
	f, err := OpenExcel(excelFile)
	if err != nil {
		return fmt.Errorf("打开Excel失败: %w", err)
	}
	defer f.Close()

	fmt.Println("📄 读取配置...")

	// 2. 读取配置
	config, err := ReadConfig(f)
	if err != nil {
		return fmt.Errorf("读取配置失败: %w", err)
	}

	fmt.Printf("主表: %s (跳过%d行)\n", config.MainSheet, config.MainSkip)
	fmt.Printf("参考表: %s (跳过%d行)\n", config.RefSheet, config.RefSkip)
	fmt.Printf("主键: %v ↔ %v\n", config.MainKeys, config.RefKeys)

	// 3. 读取主表数据
	fmt.Println("📥 加载主表数据...")
	mainData, mainHeaders, err := LoadSheetData(f, config.MainSheet, config.MainSkip)
	if err != nil {
		return fmt.Errorf("读取主表失败: %w", err)
	}
	fmt.Printf("主表加载完成: %d行, %d列\n", len(mainData), len(mainHeaders))

	// 4. 读取参考表数据
	fmt.Println("📥 加载参考表数据...")
	refData, refHeaders, err := LoadSheetData(f, config.RefSheet, config.RefSkip)
	if err != nil {
		return fmt.Errorf("读取参考表失败: %w", err)
	}
	fmt.Printf("参考表加载完成: %d行, %d列\n", len(refData), len(refHeaders))

	// 5. 主键匹配
	fmt.Println("🔗 进行主键匹配...")
	result := MatchByKeys(mainData, refData, config)
	fmt.Printf("匹配结果: 匹配%d行, 仅主表有%d行, 仅参考表有%d行\n",
		len(result.Matched), len(result.OnlyMain), len(result.OnlyRef))

	// 6. 差异分析（暂时不生成analysis，等后续实现）
	// fmt.Println("📊 分析差异原因...")
	// analysis := AnalyzeDifferences(result, mainData, refData, config)
	analysis := []DiffAnalysis{} // 空分析结果

	// 7. 输出结果
	fmt.Println("💾 生成结果...")
	if err := WriteResults(f, result, analysis, mainHeaders, refHeaders, len(mainData), len(refData)); err != nil {
		return fmt.Errorf("写入结果失败: %w", err)
	}

	// 8. 保存文件
	if err := f.Save(); err != nil {
		return fmt.Errorf("保存文件失败: %w", err)
	}

	return nil
}
