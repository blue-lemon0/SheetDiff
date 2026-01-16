package main

import (
	"fmt"
	"log"
	"os"
)

func main() {
	// 检查命令行参数
	if len(os.Args) < 2 {
		fmt.Println("📊 SheetDiff - Excel数据差异分析工具")
		fmt.Println("")
		fmt.Println("使用方法:")
		fmt.Println("  SheetDiff.exe 数据文件.xlsx")
		fmt.Println("")
		fmt.Println("要求:")
		fmt.Println("  数据文件需包含三个sheet:")
		fmt.Println("  1. 主表    - 主数据表")
		fmt.Println("  2. 参考表  - 参考数据表")
		fmt.Println("  3. 配置    - 配置信息表")
		fmt.Println("")
		os.Exit(1)
	}

	excelFile := os.Args[1]

	// 执行分析
	if err := RunAnalysis(excelFile); err != nil {
		log.Fatal("❌ 分析失败:", err)
	}

	fmt.Println("✅ 分析完成！请打开Excel查看'分析结果'sheet")
}
