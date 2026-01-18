package main

import "math/bits"

// RowSet 行集合（bitset实现）
type RowSet struct {
	bits      []uint64
	count     int
	totalRows int
}

// NewRowSet 创建空集合
func NewRowSet(totalRows int) *RowSet {
	words := (totalRows + 63) / 64
	return &RowSet{
		bits:      make([]uint64, words),
		totalRows: totalRows,
	}
}

// Set 设置某行为1
func (rs *RowSet) Set(row int) {
	if row < 0 || row >= rs.totalRows {
		return
	}
	word := row / 64
	bit := uint64(1) << (row % 64)
	if rs.bits[word]&bit == 0 {
		rs.bits[word] |= bit
		rs.count++
	}
}

// Clear 清空集合
func (rs *RowSet) Clear() {
	for i := range rs.bits {
		rs.bits[i] = 0
	}
	rs.count = 0
}

// Count 返回集合大小
func (rs *RowSet) Count() int {
	return rs.count
}

// IsEmpty 是否为空
func (rs *RowSet) IsEmpty() bool {
	return rs.count == 0
}

// Clone 深拷贝
func (rs *RowSet) Clone() *RowSet {
	newBits := make([]uint64, len(rs.bits))
	copy(newBits, rs.bits)
	return &RowSet{
		bits:      newBits,
		count:     rs.count,
		totalRows: rs.totalRows,
	}
}

// IsSubsetOf 判断是否子集：this ⊆ other
func (rs *RowSet) IsSubsetOf(other *RowSet) bool {
	if rs.count > other.count {
		return false
	}

	minWords := len(rs.bits)
	if len(other.bits) < minWords {
		minWords = len(other.bits)
	}

	for i := 0; i < minWords; i++ {
		if rs.bits[i]&^other.bits[i] != 0 {
			return false
		}
	}

	// 检查rs额外的高位字是否全0
	for i := minWords; i < len(rs.bits); i++ {
		if rs.bits[i] != 0 {
			return false
		}
	}

	return true
}

// HasIntersection 判断是否有交集
func (rs *RowSet) HasIntersection(other *RowSet) bool {
	if rs.count == 0 || other.count == 0 {
		return false
	}

	minWords := len(rs.bits)
	if len(other.bits) < minWords {
		minWords = len(other.bits)
	}

	for i := 0; i < minWords; i++ {
		if rs.bits[i]&other.bits[i] != 0 {
			return true
		}
	}

	return false
}

// IntersectionCount 计算交集大小
func (rs *RowSet) IntersectionCount(other *RowSet) int {
	count := 0
	minWords := len(rs.bits)
	if len(other.bits) < minWords {
		minWords = len(other.bits)
	}

	for i := 0; i < minWords; i++ {
		count += bits.OnesCount64(rs.bits[i] & other.bits[i])
	}
	return count
}

// UnionWith 合并集合
func (rs *RowSet) UnionWith(other *RowSet) {
	// 确保bits长度一致
	if len(rs.bits) < len(other.bits) {
		newBits := make([]uint64, len(other.bits))
		copy(newBits, rs.bits)
		rs.bits = newBits
	}

	for i := 0; i < len(other.bits); i++ {
		old := rs.bits[i]
		rs.bits[i] |= other.bits[i]
		rs.count += bits.OnesCount64(rs.bits[i]) - bits.OnesCount64(old)
	}
}

// Subtract 差集：this = this - other
func (rs *RowSet) Subtract(other *RowSet) {
	minWords := len(rs.bits)
	if len(other.bits) < minWords {
		minWords = len(other.bits)
	}

	for i := 0; i < minWords; i++ {
		old := rs.bits[i]
		rs.bits[i] &^= other.bits[i]
		rs.count += bits.OnesCount64(rs.bits[i]) - bits.OnesCount64(old)
	}
}
