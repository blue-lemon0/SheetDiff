package main

import "math/bits"

// RowSet 行集合（bitset实现）
// 用于高效存储和操作行索引集合
type RowSet struct {
	bits  []uint64 // 位图，每64行存储在一个uint64中
	count int      // 集合中元素个数（1的位数）
	size  int      // 位图总容量（支持的最大行数）
}

// NewRowSet 创建指定容量的空集合
func NewRowSet(capacity int) *RowSet {
	words := (capacity + 63) / 64 // 计算需要的uint64个数
	return &RowSet{
		bits: make([]uint64, words),
		size: capacity,
	}
}

// Add 添加行到集合中
func (rs *RowSet) Add(row int) {
	if row < 0 || row >= rs.size {
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

// Len 返回集合中元素个数
func (rs *RowSet) Len() int {
	return rs.count
}

// Empty 判断集合是否为空
func (rs *RowSet) Empty() bool {
	return rs.count == 0
}

// Clone 创建集合的深拷贝
func (rs *RowSet) Clone() *RowSet {
	newBits := make([]uint64, len(rs.bits))
	copy(newBits, rs.bits)
	return &RowSet{
		bits:  newBits,
		count: rs.count,
		size:  rs.size,
	}
}

// SubsetOf 判断当前集合是否是other的子集
// 数学表达：this ⊆ other
func (rs *RowSet) SubsetOf(other *RowSet) bool {
	// 快速检查：如果当前集合更大，不可能是子集
	if rs.count > other.count {
		return false
	}

	minWords := len(rs.bits)
	if len(other.bits) < minWords {
		minWords = len(other.bits)
	}

	// 检查this的每个1位，other是否也是1
	for i := 0; i < minWords; i++ {
		if rs.bits[i]&^other.bits[i] != 0 {
			return false // this有1而other是0
		}
	}

	// 检查this额外的高位字是否全0
	for i := minWords; i < len(rs.bits); i++ {
		if rs.bits[i] != 0 {
			return false
		}
	}

	return true
}

// Disjoint 判断两个集合是否不相交
// 数学表达：this ∩ other = ∅
func (rs *RowSet) Disjoint(other *RowSet) bool {
	// 快速检查：如果任一集合为空，必然不相交
	if rs.count == 0 || other.count == 0 {
		return true
	}

	minWords := len(rs.bits)
	if len(other.bits) < minWords {
		minWords = len(other.bits)
	}

	// 检查是否有同为1的位
	for i := 0; i < minWords; i++ {
		if rs.bits[i]&other.bits[i] != 0 {
			return false // 有交集
		}
	}

	return true
}

// IntersectCount 计算两个集合的交集大小
// 数学表达：|this ∩ other|
func (rs *RowSet) IntersectCount(other *RowSet) int {
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

// UnionWith 将other集合合并到当前集合（原地操作）
// 数学表达：this = this ∪ other
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

// Subtract 从当前集合中减去other集合（原地操作）
// 数学表达：this = this - other
func (rs *RowSet) Subtract(other *RowSet) {
	minWords := len(rs.bits)
	if len(other.bits) < minWords {
		minWords = len(other.bits)
	}

	for i := 0; i < minWords; i++ {
		old := rs.bits[i]
		rs.bits[i] &^= other.bits[i] // 按位清除操作
		rs.count += bits.OnesCount64(rs.bits[i]) - bits.OnesCount64(old)
	}
}

// Has 检查是否包含某行
func (rs *RowSet) Has(row int) bool {
	if row < 0 || row >= rs.size {
		return false
	}
	word := row / 64
	bit := uint64(1) << (row % 64)
	return (rs.bits[word] & bit) != 0
}

// Capacity 返回集合容量
func (rs *RowSet) Capacity() int {
	return rs.size
}

// Reset 重置为指定容量的空集合（复用内存）
func (rs *RowSet) Reset(capacity int) {
	if capacity != rs.size {
		words := (capacity + 63) / 64
		rs.bits = make([]uint64, words)
		rs.size = capacity
	} else {
		for i := range rs.bits {
			rs.bits[i] = 0
		}
	}
	rs.count = 0
}
