package main

import (
	"fmt"
	"strings"
)

//func main() {
//	if len(os.Args) < 2 {
//		// 尝试查找当前目录的xlsx文件
//		files, _ := filepath.Glob("*.xlsx")
//		if len(files) > 0 {
//			// 使用第一个找到的xlsx文件
//			runFile(files[0])
//		} else {
//			fmt.Println("用法: SheetDiff.exe 文件.xlsx")
//			fmt.Println("或: 把Excel文件拖到程序图标上")
//		}
//		return
//	}
//
//	runFile(os.Args[1])
//}

//func main() {
//	fmt.Println("🧪 开始规则发现算法测试...")
//
//	finder := NewRuleFinder()
//
//	// 方式1：使用内置模拟数据
//	// finder.TestBasicLogic()
//
//	// 方式2：从文件加载测试数据
//	err := finder.RunTestFromFile("test_data/test_case1.csv")
//	if err != nil {
//		fmt.Printf("❌ 测试失败: %v\n", err)
//		return
//	}
//
//	fmt.Println("\n✅ 测试完成")
//}

func runFile(filename string) {
	if !strings.HasSuffix(strings.ToLower(filename), ".xlsx") {
		fmt.Println("错误: 请提供.xlsx文件")
		return
	}

	fmt.Printf("分析文件: %s\n", filename)

	if err := RunAnalysis(filename); err != nil {
		fmt.Printf("错误: %v\n", err)
	} else {
		fmt.Println("✅ 分析完成！请查看Excel中的'分析结果'sheet")
	}
}
