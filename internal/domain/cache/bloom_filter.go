package cache

import (
	"context"
	"errors"
	"fmt"
	"math"
)

var (
	// ErrInvalidBloomFilterParams 无效的布隆过滤器参数错误
	ErrInvalidBloomFilterParams = errors.New("无效的布隆过滤器参数")
	// ErrBloomFilterFull 布隆过滤器已满错误
	ErrBloomFilterFull = errors.New("布隆过滤器已满")
)

// BloomFilter 布隆过滤器领域接口
// 定义布隆过滤器的核心操作，用于快速判断元素是否可能存在
type BloomFilter interface {
	// Add 添加元素到布隆过滤器
	// ctx: 上下文
	// key: 要添加的键
	// 返回: 操作错误
	Add(ctx context.Context, key string) error
	
	// HasKey 检查键是否可能存在
	// ctx: 上下文
	// key: 要检查的键
	// 返回: 是否可能存在（true表示可能存在，false表示一定不存在）
	HasKey(ctx context.Context, key string) bool
	
	// Clear 清空布隆过滤器
	// ctx: 上下文
	// 返回: 操作错误
	Clear(ctx context.Context) error
	
	// Stats 获取布隆过滤器统计信息
	// ctx: 上下文
	// 返回: 统计信息和错误
	Stats(ctx context.Context) (BloomFilterStats, error)
	
	// EstimateFalsePositiveRate 估算当前假阳性率
	// ctx: 上下文
	// 返回: 假阳性率和错误
	EstimateFalsePositiveRate(ctx context.Context) (float64, error)
}

// BloomFilterConfig 布隆过滤器配置值对象
// 封装布隆过滤器的配置参数和验证逻辑
type BloomFilterConfig struct {
	expectedElements uint64  // 预期元素数量
	falsePositiveRate float64 // 期望假阳性率
	bitArraySize     uint64  // 位数组大小
	hashFunctions    uint64  // 哈希函数数量
}

// NewBloomFilterConfig 创建布隆过滤器配置
// expectedElements: 预期元素数量
// falsePositiveRate: 期望假阳性率（0-1之间）
// 返回: BloomFilterConfig实例和错误信息
func NewBloomFilterConfig(expectedElements uint64, falsePositiveRate float64) (BloomFilterConfig, error) {
	if expectedElements == 0 {
		return BloomFilterConfig{}, fmt.Errorf("%w: 预期元素数量不能为0", ErrInvalidBloomFilterParams)
	}
	
	if falsePositiveRate <= 0 || falsePositiveRate >= 1 {
		return BloomFilterConfig{}, fmt.Errorf("%w: 假阳性率必须在0和1之间", ErrInvalidBloomFilterParams)
	}
	
	// 计算最优位数组大小: m = -n * ln(p) / (ln(2)^2)
	bitArraySize := uint64(math.Ceil(-float64(expectedElements) * math.Log(falsePositiveRate) / (math.Log(2) * math.Log(2))))
	
	// 计算最优哈希函数数量: k = (m/n) * ln(2)
	hashFunctions := uint64(math.Ceil(float64(bitArraySize) / float64(expectedElements) * math.Log(2)))
	
	// 确保至少有一个哈希函数
	if hashFunctions == 0 {
		hashFunctions = 1
	}
	
	return BloomFilterConfig{
		expectedElements:  expectedElements,
		falsePositiveRate: falsePositiveRate,
		bitArraySize:      bitArraySize,
		hashFunctions:     hashFunctions,
	}, nil
}

// ExpectedElements 获取预期元素数量
func (c BloomFilterConfig) ExpectedElements() uint64 {
	return c.expectedElements
}

// FalsePositiveRate 获取期望假阳性率
func (c BloomFilterConfig) FalsePositiveRate() float64 {
	return c.falsePositiveRate
}

// BitArraySize 获取位数组大小
func (c BloomFilterConfig) BitArraySize() uint64 {
	return c.bitArraySize
}

// HashFunctions 获取哈希函数数量
func (c BloomFilterConfig) HashFunctions() uint64 {
	return c.hashFunctions
}

// MemoryUsage 计算内存使用量（字节）
func (c BloomFilterConfig) MemoryUsage() uint64 {
	return (c.bitArraySize + 7) / 8 // 向上取整到字节
}

// BloomFilterStats 布隆过滤器统计信息值对象
// 封装布隆过滤器的运行时统计数据
type BloomFilterStats struct {
	config           BloomFilterConfig
	addedElements    uint64  // 已添加元素数量
	setBits          uint64  // 已设置的位数量
	estimatedFPR     float64 // 估算的假阳性率
	memoryUsage      uint64  // 内存使用量
}

// NewBloomFilterStats 创建布隆过滤器统计信息
// config: 布隆过滤器配置
// addedElements: 已添加元素数量
// setBits: 已设置的位数量
// 返回: BloomFilterStats实例
func NewBloomFilterStats(config BloomFilterConfig, addedElements, setBits uint64) BloomFilterStats {
	// 计算估算的假阳性率: (1 - e^(-k*n/m))^k
	// 其中 k = 哈希函数数量, n = 已添加元素数量, m = 位数组大小
	var estimatedFPR float64
	if addedElements > 0 {
		k := float64(config.HashFunctions())
		n := float64(addedElements)
		m := float64(config.BitArraySize())
		estimatedFPR = math.Pow(1-math.Exp(-k*n/m), k)
	}
	
	return BloomFilterStats{
		config:        config,
		addedElements: addedElements,
		setBits:       setBits,
		estimatedFPR:  estimatedFPR,
		memoryUsage:   config.MemoryUsage(),
	}
}

// Config 获取布隆过滤器配置
func (s BloomFilterStats) Config() BloomFilterConfig {
	return s.config
}

// AddedElements 获取已添加元素数量
func (s BloomFilterStats) AddedElements() uint64 {
	return s.addedElements
}

// SetBits 获取已设置的位数量
func (s BloomFilterStats) SetBits() uint64 {
	return s.setBits
}

// EstimatedFalsePositiveRate 获取估算的假阳性率
func (s BloomFilterStats) EstimatedFalsePositiveRate() float64 {
	return s.estimatedFPR
}

// MemoryUsage 获取内存使用量
func (s BloomFilterStats) MemoryUsage() uint64 {
	return s.memoryUsage
}

// LoadFactor 计算负载因子（已设置位数 / 总位数）
func (s BloomFilterStats) LoadFactor() float64 {
	if s.config.BitArraySize() == 0 {
		return 0
	}
	return float64(s.setBits) / float64(s.config.BitArraySize())
}

// IsOverloaded 检查是否过载
// 当已添加元素数量超过预期时返回true
func (s BloomFilterStats) IsOverloaded() bool {
	return s.addedElements > s.config.ExpectedElements()
}

// EfficiencyRatio 计算效率比率
// 实际假阳性率与期望假阳性率的比值
func (s BloomFilterStats) EfficiencyRatio() float64 {
	if s.config.FalsePositiveRate() == 0 {
		return 0
	}
	return s.estimatedFPR / s.config.FalsePositiveRate()
}

// BloomFilterKey 布隆过滤器键值对象
// 封装布隆过滤器中键的处理逻辑
type BloomFilterKey struct {
	value string
}

// NewBloomFilterKey 创建布隆过滤器键
// key: 键值字符串
// 返回: BloomFilterKey实例和错误信息
func NewBloomFilterKey(key string) (BloomFilterKey, error) {
	if key == "" {
		return BloomFilterKey{}, fmt.Errorf("%w: 布隆过滤器键不能为空", ErrInvalidCacheKey)
	}
	
	// 布隆过滤器对键长度的限制可以更宽松
	if len(key) > 1000 {
		return BloomFilterKey{}, fmt.Errorf("%w: 布隆过滤器键长度不能超过1000个字符", ErrInvalidCacheKey)
	}
	
	return BloomFilterKey{value: key}, nil
}

// String 返回键的字符串表示
func (k BloomFilterKey) String() string {
	return k.value
}

// Bytes 返回键的字节表示
func (k BloomFilterKey) Bytes() []byte {
	return []byte(k.value)
}

// Hash 计算键的哈希值
// seed: 哈希种子
// 返回: 哈希值
func (k BloomFilterKey) Hash(seed uint64) uint64 {
	// 使用FNV-1a哈希算法
	hash := uint64(14695981039346656037) // FNV offset basis
	hash ^= seed
	
	for _, b := range k.Bytes() {
		hash ^= uint64(b)
		hash *= 1099511628211 // FNV prime
	}
	
	return hash
}

// Equals 比较两个键是否相等
func (k BloomFilterKey) Equals(other BloomFilterKey) bool {
	return k.value == other.value
}

// BloomFilterRepository 布隆过滤器仓储接口
// 定义布隆过滤器的持久化操作
type BloomFilterRepository interface {
	// Save 保存布隆过滤器状态
	// ctx: 上下文
	// name: 布隆过滤器名称
	// data: 位数组数据
	// config: 配置信息
	// stats: 统计信息
	// 返回: 操作错误
	Save(ctx context.Context, name string, data []byte, config BloomFilterConfig, stats BloomFilterStats) error
	
	// Load 加载布隆过滤器状态
	// ctx: 上下文
	// name: 布隆过滤器名称
	// 返回: 位数组数据、配置信息、统计信息和错误
	Load(ctx context.Context, name string) ([]byte, BloomFilterConfig, BloomFilterStats, error)
	
	// Delete 删除布隆过滤器
	// ctx: 上下文
	// name: 布隆过滤器名称
	// 返回: 操作错误
	Delete(ctx context.Context, name string) error
	
	// Exists 检查布隆过滤器是否存在
	// ctx: 上下文
	// name: 布隆过滤器名称
	// 返回: 是否存在和错误
	Exists(ctx context.Context, name string) (bool, error)
}
