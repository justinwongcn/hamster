# bloom_filter.go - 布隆过滤器领域模型

## 文件概述

`bloom_filter.go` 定义了布隆过滤器的领域模型，包括核心接口、配置值对象、统计信息值对象和键值对象。该文件遵循DDD设计原则，将布隆过滤器的业务逻辑封装在领域层，为防止缓存穿透提供核心抽象。

## 核心功能

### 1. 错误定义

```go
var (
    ErrInvalidBloomFilterParams = errors.New("无效的布隆过滤器参数")
    ErrBloomFilterFull         = errors.New("布隆过滤器已满")
)
```

定义了布隆过滤器相关的领域错误，便于错误识别和处理。

### 2. BloomFilter 接口

```go
type BloomFilter interface {
    Add(ctx context.Context, key string) error
    HasKey(ctx context.Context, key string) bool
    Clear(ctx context.Context) error
    Stats(ctx context.Context) (BloomFilterStats, error)
    EstimateFalsePositiveRate(ctx context.Context) (float64, error)
}
```

**核心方法：**

- **Add**: 添加元素到布隆过滤器
- **HasKey**: 检查键是否可能存在（false表示一定不存在，true表示可能存在）
- **Clear**: 清空布隆过滤器
- **Stats**: 获取统计信息
- **EstimateFalsePositiveRate**: 估算当前假阳性率

### 3. BloomFilterConfig 配置值对象

#### 结构定义

```go
type BloomFilterConfig struct {
    expectedElements  uint64  // 预期元素数量
    falsePositiveRate float64 // 期望假阳性率
    bitArraySize      uint64  // 位数组大小
    hashFunctions     uint64  // 哈希函数数量
}
```

#### 构造函数

```go
func NewBloomFilterConfig(expectedElements uint64, falsePositiveRate float64) (BloomFilterConfig, error)
```

**参数验证：**

- 预期元素数量不能为0
- 假阳性率必须在0和1之间

**计算公式：**

- 位数组大小: `m = -n * ln(p) / (ln(2)^2)`
- 哈希函数数量: `k = (m/n) * ln(2)`

其中：n = 预期元素数量，p = 假阳性率

**示例：**

```go
// 创建配置：预期1000个元素，1%假阳性率
config, err := NewBloomFilterConfig(1000, 0.01)
if err != nil {
    log.Printf("配置创建失败: %v", err)
    return
}

fmt.Printf("位数组大小: %d\n", config.BitArraySize())
fmt.Printf("哈希函数数量: %d\n", config.HashFunctions())
fmt.Printf("内存使用: %d bytes\n", config.MemoryUsage())
```

#### 访问方法

- `ExpectedElements()`: 获取预期元素数量
- `FalsePositiveRate()`: 获取期望假阳性率
- `BitArraySize()`: 获取位数组大小
- `HashFunctions()`: 获取哈希函数数量
- `MemoryUsage()`: 计算内存使用量（字节）

### 4. BloomFilterStats 统计信息值对象

#### 结构定义

```go
type BloomFilterStats struct {
    config        BloomFilterConfig
    addedElements uint64  // 已添加元素数量
    setBits       uint64  // 已设置的位数量
    estimatedFPR  float64 // 估算的假阳性率
    memoryUsage   uint64  // 内存使用量
}
```

#### 构造函数

```go
func NewBloomFilterStats(config BloomFilterConfig, addedElements, setBits uint64) BloomFilterStats
```

**自动计算：**

- 估算假阳性率: `(1 - e^(-k*n/m))^k`
- 内存使用量: 基于配置计算

**示例：**

```go
stats := NewBloomFilterStats(config, 500, 2000)

fmt.Printf("已添加元素: %d\n", stats.AddedElements())
fmt.Printf("负载因子: %.4f\n", stats.LoadFactor())
fmt.Printf("估算假阳性率: %.4f\n", stats.EstimatedFalsePositiveRate())
fmt.Printf("效率比率: %.4f\n", stats.EfficiencyRatio())

if stats.IsOverloaded() {
    fmt.Println("警告: 布隆过滤器过载")
}
```

#### 分析方法

- `LoadFactor()`: 计算负载因子（已设置位数 / 总位数）
- `IsOverloaded()`: 检查是否过载（已添加元素 > 预期元素）
- `EfficiencyRatio()`: 计算效率比率（实际假阳性率 / 期望假阳性率）

### 5. BloomFilterKey 键值对象

#### 结构定义

```go
type BloomFilterKey struct {
    value string
}
```

#### 构造函数

```go
func NewBloomFilterKey(key string) (BloomFilterKey, error)
```

**验证规则：**

- 键不能为空
- 键长度不能超过1000个字符

**示例：**

```go
key, err := NewBloomFilterKey("user:12345")
if err != nil {
    log.Printf("键创建失败: %v", err)
    return
}

fmt.Printf("键值: %s\n", key.String())
fmt.Printf("字节表示: %v\n", key.Bytes())
fmt.Printf("哈希值: %d\n", key.Hash(0))
```

#### 操作方法

- `String()`: 返回字符串表示
- `Bytes()`: 返回字节表示
- `Hash(seed uint64)`: 计算哈希值（使用FNV-1a算法）
- `Equals(other BloomFilterKey)`: 比较键是否相等

### 6. BloomFilterRepository 仓储接口

```go
type BloomFilterRepository interface {
    Save(ctx context.Context, name string, data []byte, config BloomFilterConfig, stats BloomFilterStats) error
    Load(ctx context.Context, name string) ([]byte, BloomFilterConfig, BloomFilterStats, error)
    Delete(ctx context.Context, name string) error
    Exists(ctx context.Context, name string) (bool, error)
}
```

定义了布隆过滤器的持久化操作，支持状态保存和恢复。

## 使用示例

### 1. 基本配置和使用

```go
package main

import (
    "context"
    "fmt"
    "log"
    
    "github.com/justinwongcn/hamster/internal/domain/cache"
)

func main() {
    // 创建配置
    config, err := cache.NewBloomFilterConfig(10000, 0.01) // 1万元素，1%假阳性
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("配置信息:\n")
    fmt.Printf("  预期元素: %d\n", config.ExpectedElements())
    fmt.Printf("  假阳性率: %.4f\n", config.FalsePositiveRate())
    fmt.Printf("  位数组大小: %d\n", config.BitArraySize())
    fmt.Printf("  哈希函数数量: %d\n", config.HashFunctions())
    fmt.Printf("  内存使用: %d bytes\n", config.MemoryUsage())
}
```

### 2. 键的创建和操作

```go
func demonstrateKey() {
    // 创建键
    key1, err := cache.NewBloomFilterKey("user:12345")
    if err != nil {
        log.Printf("键创建失败: %v", err)
        return
    }
    
    key2, err := cache.NewBloomFilterKey("user:12345")
    if err != nil {
        log.Printf("键创建失败: %v", err)
        return
    }
    
    // 比较键
    if key1.Equals(key2) {
        fmt.Println("键相等")
    }
    
    // 计算哈希值
    hash1 := key1.Hash(0)
    hash2 := key1.Hash(1) // 不同种子产生不同哈希值
    
    fmt.Printf("哈希值1: %d\n", hash1)
    fmt.Printf("哈希值2: %d\n", hash2)
}
```

### 3. 统计信息分析

```go
func analyzeStats() {
    config, _ := cache.NewBloomFilterConfig(1000, 0.01)
    
    // 模拟不同的使用情况
    scenarios := []struct {
        name          string
        addedElements uint64
        setBits       uint64
    }{
        {"正常使用", 500, 2000},
        {"接近满载", 950, 4500},
        {"过载使用", 1200, 5500},
    }
    
    for _, scenario := range scenarios {
        stats := cache.NewBloomFilterStats(config, scenario.addedElements, scenario.setBits)
        
        fmt.Printf("\n%s:\n", scenario.name)
        fmt.Printf("  已添加元素: %d\n", stats.AddedElements())
        fmt.Printf("  负载因子: %.4f\n", stats.LoadFactor())
        fmt.Printf("  估算假阳性率: %.4f\n", stats.EstimatedFalsePositiveRate())
        fmt.Printf("  效率比率: %.4f\n", stats.EfficiencyRatio())
        fmt.Printf("  是否过载: %v\n", stats.IsOverloaded())
    }
}
```

### 4. 配置优化

```go
func optimizeConfig() {
    // 测试不同配置的效果
    testCases := []struct {
        elements uint64
        fpr      float64
    }{
        {1000, 0.1},   // 10%假阳性率
        {1000, 0.01},  // 1%假阳性率
        {1000, 0.001}, // 0.1%假阳性率
        {10000, 0.01}, // 更多元素
    }
    
    fmt.Println("配置优化分析:")
    for _, tc := range testCases {
        config, err := cache.NewBloomFilterConfig(tc.elements, tc.fpr)
        if err != nil {
            continue
        }
        
        fmt.Printf("\n元素: %d, 假阳性率: %.3f%%\n", tc.elements, tc.fpr*100)
        fmt.Printf("  位数组大小: %d\n", config.BitArraySize())
        fmt.Printf("  哈希函数数量: %d\n", config.HashFunctions())
        fmt.Printf("  内存使用: %d bytes (%.2f KB)\n", 
            config.MemoryUsage(), float64(config.MemoryUsage())/1024)
    }
}
```

## 设计原则

### 1. 值对象不变性

- 所有值对象创建后不可修改
- 通过构造函数进行验证
- 提供只读访问方法

### 2. 领域封装

- 将布隆过滤器的数学计算封装在领域层
- 隐藏实现细节，暴露业务概念
- 提供类型安全的操作

### 3. 错误处理

- 定义领域特定的错误类型
- 在构造函数中进行参数验证
- 提供有意义的错误信息

### 4. 接口分离

- 将核心操作和持久化操作分离
- 支持不同的实现策略
- 便于测试和扩展

## 注意事项

### 1. 参数选择

```go
// ✅ 推荐：根据实际需求选择参数
config, err := cache.NewBloomFilterConfig(
    expectedElements,  // 根据实际数据量设置
    0.01,             // 1%假阳性率，平衡内存和准确性
)

// ❌ 避免：假阳性率过低导致内存浪费
config, err := cache.NewBloomFilterConfig(1000, 0.0001) // 0.01%假阳性率
```

### 2. 键的使用

```go
// ✅ 推荐：使用有意义的键名
key, err := cache.NewBloomFilterKey("user:profile:12345")

// ❌ 避免：键名过长
longKey := strings.Repeat("a", 1001)
key, err := cache.NewBloomFilterKey(longKey) // 会返回错误
```

### 3. 统计信息监控

```go
// ✅ 推荐：定期检查统计信息
if stats.IsOverloaded() {
    log.Println("警告: 布隆过滤器过载，建议增加容量")
}

if stats.EfficiencyRatio() > 2.0 {
    log.Println("警告: 假阳性率过高，影响效率")
}
```

### 4. 内存管理

```go
// ✅ 推荐：根据可用内存选择配置
availableMemory := getAvailableMemory()
maxElements := calculateMaxElements(availableMemory, 0.01)
config, err := cache.NewBloomFilterConfig(maxElements, 0.01)
```
