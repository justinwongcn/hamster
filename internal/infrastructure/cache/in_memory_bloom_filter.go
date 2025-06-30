package cache

import (
	"context"
	"sync"

	domainCache "github.com/justinwongcn/hamster/internal/domain/cache"
)

// InMemoryBloomFilter 内存布隆过滤器实现
// 使用位数组在内存中实现布隆过滤器
// 线程安全，支持并发访问
type InMemoryBloomFilter struct {
	config       domainCache.BloomFilterConfig
	bitArray     []byte
	addedCount   uint64
	mu           sync.RWMutex
}

// NewInMemoryBloomFilter 创建新的内存布隆过滤器
// config: 布隆过滤器配置
// 返回: InMemoryBloomFilter实例
func NewInMemoryBloomFilter(config domainCache.BloomFilterConfig) *InMemoryBloomFilter {
	// 计算需要的字节数
	byteSize := (config.BitArraySize() + 7) / 8
	
	return &InMemoryBloomFilter{
		config:     config,
		bitArray:   make([]byte, byteSize),
		addedCount: 0,
	}
}

// Add 添加元素到布隆过滤器
// ctx: 上下文
// key: 要添加的键
// 返回: 操作错误
func (bf *InMemoryBloomFilter) Add(ctx context.Context, key string) error {
	bf.mu.Lock()
	defer bf.mu.Unlock()
	
	// 创建布隆过滤器键
	bfKey, err := domainCache.NewBloomFilterKey(key)
	if err != nil {
		return err
	}
	
	// 计算哈希值并设置对应的位
	for i := uint64(0); i < bf.config.HashFunctions(); i++ {
		hash := bfKey.Hash(i)
		bitIndex := hash % bf.config.BitArraySize()
		bf.setBit(bitIndex)
	}
	
	bf.addedCount++
	return nil
}

// HasKey 检查键是否可能存在
// ctx: 上下文
// key: 要检查的键
// 返回: 是否可能存在（true表示可能存在，false表示一定不存在）
func (bf *InMemoryBloomFilter) HasKey(ctx context.Context, key string) bool {
	bf.mu.RLock()
	defer bf.mu.RUnlock()
	
	// 创建布隆过滤器键
	bfKey, err := domainCache.NewBloomFilterKey(key)
	if err != nil {
		return false
	}
	
	// 检查所有哈希值对应的位是否都被设置
	for i := uint64(0); i < bf.config.HashFunctions(); i++ {
		hash := bfKey.Hash(i)
		bitIndex := hash % bf.config.BitArraySize()
		if !bf.getBit(bitIndex) {
			return false
		}
	}
	
	return true
}

// Clear 清空布隆过滤器
// ctx: 上下文
// 返回: 操作错误
func (bf *InMemoryBloomFilter) Clear(ctx context.Context) error {
	bf.mu.Lock()
	defer bf.mu.Unlock()
	
	// 重置位数组
	for i := range bf.bitArray {
		bf.bitArray[i] = 0
	}
	
	bf.addedCount = 0
	return nil
}

// Stats 获取布隆过滤器统计信息
// ctx: 上下文
// 返回: 统计信息和错误
func (bf *InMemoryBloomFilter) Stats(ctx context.Context) (domainCache.BloomFilterStats, error) {
	bf.mu.RLock()
	defer bf.mu.RUnlock()
	
	// 计算已设置的位数
	setBits := bf.countSetBits()
	
	// 创建统计信息
	stats := domainCache.NewBloomFilterStats(bf.config, bf.addedCount, setBits)
	return stats, nil
}

// EstimateFalsePositiveRate 估算当前假阳性率
// ctx: 上下文
// 返回: 假阳性率和错误
func (bf *InMemoryBloomFilter) EstimateFalsePositiveRate(ctx context.Context) (float64, error) {
	stats, err := bf.Stats(ctx)
	if err != nil {
		return 0, err
	}
	
	return stats.EstimatedFalsePositiveRate(), nil
}

// setBit 设置指定位置的位为1
// bitIndex: 位索引
func (bf *InMemoryBloomFilter) setBit(bitIndex uint64) {
	byteIndex := bitIndex / 8
	bitOffset := bitIndex % 8
	
	if byteIndex < uint64(len(bf.bitArray)) {
		bf.bitArray[byteIndex] |= (1 << bitOffset)
	}
}

// getBit 获取指定位置的位值
// bitIndex: 位索引
// 返回: 位值（true表示1，false表示0）
func (bf *InMemoryBloomFilter) getBit(bitIndex uint64) bool {
	byteIndex := bitIndex / 8
	bitOffset := bitIndex % 8
	
	if byteIndex >= uint64(len(bf.bitArray)) {
		return false
	}
	
	return (bf.bitArray[byteIndex] & (1 << bitOffset)) != 0
}

// countSetBits 计算已设置的位数
// 返回: 已设置的位数
func (bf *InMemoryBloomFilter) countSetBits() uint64 {
	count := uint64(0)
	
	for _, b := range bf.bitArray {
		// 使用位操作计算字节中设置的位数
		count += uint64(popcount(b))
	}
	
	return count
}

// popcount 计算字节中设置的位数
// b: 字节值
// 返回: 设置的位数
func popcount(b byte) int {
	count := 0
	for b != 0 {
		count++
		b &= b - 1 // 清除最低位的1
	}
	return count
}

// GetBitArray 获取位数组（用于测试和调试）
// 返回: 位数组的副本
func (bf *InMemoryBloomFilter) GetBitArray() []byte {
	bf.mu.RLock()
	defer bf.mu.RUnlock()
	
	result := make([]byte, len(bf.bitArray))
	copy(result, bf.bitArray)
	return result
}

// GetConfig 获取配置信息（用于测试和调试）
// 返回: 布隆过滤器配置
func (bf *InMemoryBloomFilter) GetConfig() domainCache.BloomFilterConfig {
	return bf.config
}

// GetAddedCount 获取已添加元素数量（用于测试和调试）
// 返回: 已添加元素数量
func (bf *InMemoryBloomFilter) GetAddedCount() uint64 {
	bf.mu.RLock()
	defer bf.mu.RUnlock()
	return bf.addedCount
}

// Clone 克隆布隆过滤器
// 返回: 新的布隆过滤器实例
func (bf *InMemoryBloomFilter) Clone() *InMemoryBloomFilter {
	bf.mu.RLock()
	defer bf.mu.RUnlock()
	
	newBF := &InMemoryBloomFilter{
		config:     bf.config,
		bitArray:   make([]byte, len(bf.bitArray)),
		addedCount: bf.addedCount,
	}
	
	copy(newBF.bitArray, bf.bitArray)
	return newBF
}

// Merge 合并另一个布隆过滤器
// other: 要合并的布隆过滤器
// 返回: 操作错误
func (bf *InMemoryBloomFilter) Merge(other *InMemoryBloomFilter) error {
	bf.mu.Lock()
	defer bf.mu.Unlock()
	
	other.mu.RLock()
	defer other.mu.RUnlock()
	
	// 检查配置是否兼容
	if bf.config.BitArraySize() != other.config.BitArraySize() ||
		bf.config.HashFunctions() != other.config.HashFunctions() {
		return domainCache.ErrInvalidBloomFilterParams
	}
	
	// 合并位数组（按位或操作）
	for i := range bf.bitArray {
		bf.bitArray[i] |= other.bitArray[i]
	}
	
	// 更新添加计数（注意：这是近似值，因为可能有重复元素）
	bf.addedCount += other.addedCount
	
	return nil
}

// Reset 重置布隆过滤器到初始状态
// 与Clear类似，但保持配置不变
func (bf *InMemoryBloomFilter) Reset() {
	bf.mu.Lock()
	defer bf.mu.Unlock()
	
	// 重置位数组
	for i := range bf.bitArray {
		bf.bitArray[i] = 0
	}
	
	bf.addedCount = 0
}
