package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	if len(os.Args) < 2 {
		// 尝试查找当前目录的xlsx文件
		files, _ := filepath.Glob("*.xlsx")
		if len(files) > 0 {
			// 使用第一个找到的xlsx文件
			runFile(files[0])
		} else {
			fmt.Println("用法: SheetDiff.exe 文件.xlsx")
			fmt.Println("或: 把Excel文件拖到程序图标上")
		}
		return
	}

	runFile(os.Args[1])
}

func runFile(filename string) {
	if !strings.HasSuffix(strings.ToLower(filename), ".xlsx") {
		fmt.Println("请提供.xlsx文件")
		return
	}

	fmt.Printf("分析: %s\n", filename)

	if err := RunAnalysis(filename); err != nil {
		fmt.Printf("错误: %v\n", err)
	} else {
		fmt.Println("完成！请查看Excel中的'分析结果'sheet")
	}
}
