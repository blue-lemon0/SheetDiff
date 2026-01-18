package main

//
//import (
//	"encoding/csv"
//	"fmt"
//	"os"
//	"strings"
//)
//
//// TestCSVLoader 测试专用的CSV加载器
//type TestCSVLoader struct {
//}
//
//// TestData 测试数据结构
//type TestData struct {
//	Rows      []Row
//	CommonSet *RowSet
//	Fields    []string
//	TotalRows int
//}
//
//// LoadTestData 从CSV文件加载测试数据
//func (l *TestCSVLoader) LoadTestData(filepath string) (*TestData, error) {
//	file, err := os.Open(filepath)
//	if err != nil {
//		return nil, fmt.Errorf("无法打开文件: %v", err)
//	}
//	defer file.Close()
//
//	reader := csv.NewReader(file)
//	records, err := reader.ReadAll()
//	if err != nil {
//		return nil, fmt.Errorf("读取CSV失败: %v", err)
//	}
//
//	if len(records) < 2 {
//		return nil, fmt.Errorf("CSV文件至少需要表头和数据行")
//	}
//
//	// 解析表头
//	headers := records[0]
//	fmt.Printf("CSV表头: %v\n", headers)
//
//	// 确定字段列和共有行列
//	fieldCols := []int{}
//	commonCol := -1
//
//	for i, header := range headers {
//		header = strings.TrimSpace(header)
//		if strings.Contains(strings.ToLower(header), "共有") {
//			commonCol = i
//		} else {
//			fieldCols = append(fieldCols, i)
//		}
//	}
//
//	if commonCol == -1 {
//		return nil, fmt.Errorf("CSV需要包含'共有'列来标识共有行")
//	}
//
//	// 提取字段名
//	fields := make([]string, len(fieldCols))
//	for i, col := range fieldCols {
//		fields[i] = headers[col]
//	}
//
//	// 解析数据行
//	rows := []Row{}
//	commonSet := NewRowSet(len(records) - 1) // 减去表头
//
//	for rowIdx, record := range records[1:] {
//		row := make(Row)
//
//		// 解析字段值
//		for i, col := range fieldCols {
//			if col < len(record) {
//				row[fields[i]] = strings.TrimSpace(record[col])
//			}
//		}
//
//		rows = append(rows, row)
//
//		// 解析是否共有行
//		if commonCol < len(record) {
//			isCommon := l.parseCommonFlag(record[commonCol])
//			if isCommon {
//				commonSet.Set(rowIdx)
//			}
//		}
//	}
//
//	return &TestData{
//		Rows:      rows,
//		CommonSet: commonSet,
//		Fields:    fields,
//		TotalRows: len(rows),
//	}, nil
//}
//
//// parseCommonFlag 解析共有标志
//func (l *TestCSVLoader) parseCommonFlag(value string) bool {
//	value = strings.TrimSpace(strings.ToLower(value))
//	return value == "是" || value == "y" || value == "yes" || value == "true" || value == "1"
//}
//
//// PrintTestData 打印测试数据信息
//func (l *TestCSVLoader) PrintTestData(data *TestData) {
//	fmt.Printf("测试数据加载完成:\n")
//	fmt.Printf("- 总行数: %d\n", data.TotalRows)
//	fmt.Printf("- 共有行数: %d\n", data.CommonSet.Count())
//	fmt.Printf("- 字段数量: %d\n", len(data.Fields))
//	fmt.Printf("- 字段列表: %v\n", data.Fields)
//
//	// 打印前几行数据
//	fmt.Printf("\n前5行数据:\n")
//	for i := 0; i < 5 && i < len(data.Rows); i++ {
//		fmt.Printf("  行%d: ", i+1)
//		for _, field := range data.Fields {
//			fmt.Printf("%s=%s ", field, data.Rows[i][field])
//		}
//		if data.CommonSet.IntersectionCount(NewRowSetWithValue(data.TotalRows, i)) > 0 {
//			fmt.Printf("(共有行)")
//		}
//		fmt.Println()
//	}
//}
//
//// NewRowSetWithValue 创建一个包含单个值的RowSet（辅助函数）
//func NewRowSetWithValue(totalRows, value int) *RowSet {
//	set := NewRowSet(totalRows)
//	set.Set(value)
//	return set
//}
