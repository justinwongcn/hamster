# in_memory_bloom_filter.go - 内存布隆过滤器实现

## 文件概述

`in_memory_bloom_filter.go` 实现了基于内存的布隆过滤器，用于快速判断元素是否可能存在于集合中。这是防止缓存穿透的核心组件，通过牺牲少量的假阳性来换取极高的查询性能和内存效率。

## 核心功能

### 1. InMemoryBloomFilter 结构体

```go
type InMemoryBloomFilter struct {
    config       domain.BloomFilterConfig  // 布隆过滤器配置
    bitArray     []uint64                  // 位数组存储
    hashFunc     domain.Hash               // 哈希函数
    addedCount   uint64                    // 已添加元素数量
    mu           sync.RWMutex              // 读写锁
}
```

**主要特性：**

- 基于位数组的高效存储
- 支持多个哈希函数
- 线程安全的并发访问
- 可配置的假阳性率
- 详细的统计信息

### 2. 布隆过滤器配置

```go
type BloomFilterConfig struct {
    expectedElements   uint64   // 预期元素数量
    falsePositiveRate float64   // 假阳性率
    bitArraySize      uint64    // 位数组大小
    hashFunctions     uint64    // 哈希函数数量
}
```

**配置计算公式：**

- 位数组大小: `m = -n * ln(p) / (ln(2)^2)`
- 哈希函数数量: `k = (m/n) * ln(2)`

其中：n = 预期元素数量，p = 假阳性率

## 主要方法

### 1. 构造函数

```go
func NewInMemoryBloomFilter(config domain.BloomFilterConfig) *InMemoryBloomFilter
```

**示例：**

```go
// 创建预期1000个元素，1%假阳性率的布隆过滤器
config, err := domain.NewBloomFilterConfig(1000, 0.01)
if err != nil {
    return err
}

bloomFilter := NewInMemoryBloomFilter(config)
```

### 2. 核心操作

#### Add - 添加元素

```go
func (bf *InMemoryBloomFilter) Add(ctx context.Context, key string) error
```

**实现逻辑：**

1. 验证输入参数
2. 计算多个哈希值
3. 设置对应的位
4. 更新统计信息

**示例：**

```go
err := bloomFilter.Add(ctx, "user:123")
if err != nil {
    log.Printf("添加元素失败: %v", err)
}
```

#### HasKey - 检查元素

```go
func (bf *InMemoryBloomFilter) HasKey(ctx context.Context, key string) bool
```

**实现逻辑：**

1. 计算多个哈希值
2. 检查对应的位是否都为1
3. 返回检查结果

**示例：**

```go
exists := bloomFilter.HasKey(ctx, "user:123")
if exists {
    fmt.Println("元素可能存在")
} else {
    fmt.Println("元素一定不存在")
}
```

#### Clear - 清空过滤器

```go
func (bf *InMemoryBloomFilter) Clear(ctx context.Context) error
```

### 3. 统计信息

#### Stats - 获取统计信息

```go
func (bf *InMemoryBloomFilter) Stats(ctx context.Context) (domain.BloomFilterStats, error)
```

**返回信息包括：**

- 配置参数（预期元素数、假阳性率等）
- 运行时统计（已添加元素数、设置位数等）
- 性能指标（估算假阳性率、内存使用量等）

## 哈希函数实现

### 1. 多重哈希

```go
func (bf *InMemoryBloomFilter) getHashes(key string) []uint64 {
    data := []byte(key)
    hashes := make([]uint64, bf.config.HashFunctions())
    
    // 使用双重哈希技术生成多个哈希值
    hash1 := bf.hashFunc(data)
    hash2 := bf.hashFunc(append(data, 0x01))
    
    for i := uint64(0); i < bf.config.HashFunctions(); i++ {
        // 组合哈希: h(i) = hash1 + i * hash2
        combinedHash := uint64(hash1) + i*uint64(hash2)
        hashes[i] = combinedHash % bf.config.BitArraySize()
    }
    
    return hashes
}
```

### 2. 位操作优化

```go
func (bf *InMemoryBloomFilter) setBit(bitIndex uint64) {
    arrayIndex := bitIndex / 64
    bitOffset := bitIndex % 64
    
    bf.bitArray[arrayIndex] |= (1 << bitOffset)
}

func (bf *InMemoryBloomFilter) getBit(bitIndex uint64) bool {
    arrayIndex := bitIndex / 64
    bitOffset := bitIndex % 64
    
    return (bf.bitArray[arrayIndex] & (1 << bitOffset)) != 0
}
```

## 性能优化

### 1. 内存对齐

```go
func NewInMemoryBloomFilter(config domain.BloomFilterConfig) *InMemoryBloomFilter {
    // 确保位数组大小是64的倍数，提高内存访问效率
    bitArraySize := config.BitArraySize()
    arraySize := (bitArraySize + 63) / 64  // 向上取整到64的倍数
    
    return &InMemoryBloomFilter{
        config:     config,
        bitArray:   make([]uint64, arraySize),
        hashFunc:   config.HashFunc(),
        addedCount: 0,
    }
}
```

### 2. 批量操作

```go
func (bf *InMemoryBloomFilter) AddBatch(ctx context.Context, keys []string) error {
    bf.mu.Lock()
    defer bf.mu.Unlock()
    
    for _, key := range keys {
        hashes := bf.getHashes(key)
        for _, hash := range hashes {
            bf.setBit(hash)
        }
        bf.addedCount++
    }
    
    return nil
}

func (bf *InMemoryBloomFilter) HasKeysBatch(ctx context.Context, keys []string) []bool {
    bf.mu.RLock()
    defer bf.mu.RUnlock()
    
    results := make([]bool, len(keys))
    for i, key := range keys {
        results[i] = bf.hasKeyInternal(key)
    }
    
    return results
}
```

### 3. 并发优化

```go
// 读操作使用读锁，允许并发查询
func (bf *InMemoryBloomFilter) HasKey(ctx context.Context, key string) bool {
    bf.mu.RLock()
    defer bf.mu.RUnlock()
    
    return bf.hasKeyInternal(key)
}

// 写操作使用写锁，确保数据一致性
func (bf *InMemoryBloomFilter) Add(ctx context.Context, key string) error {
    bf.mu.Lock()
    defer bf.mu.Unlock()
    
    return bf.addInternal(key)
}
```

## 统计信息实现

### 1. BloomFilterStats 结构

```go
type BloomFilterStatsImpl struct {
    config         domain.BloomFilterConfig
    addedElements  uint64
    setBits        uint64
    memoryUsage    uint64
}

func (s BloomFilterStatsImpl) EstimatedFalsePositiveRate() float64 {
    if s.addedElements == 0 {
        return 0
    }
    
    // 计算实际假阳性率: (1 - e^(-k*n/m))^k
    k := float64(s.config.HashFunctions())
    n := float64(s.addedElements)
    m := float64(s.config.BitArraySize())
    
    return math.Pow(1-math.Exp(-k*n/m), k)
}

func (s BloomFilterStatsImpl) LoadFactor() float64 {
    return float64(s.setBits) / float64(s.config.BitArraySize())
}

func (s BloomFilterStatsImpl) IsOverloaded() bool {
    // 当负载因子超过0.5时认为过载
    return s.LoadFactor() > 0.5
}
```

### 2. 内存使用计算

```go
func (bf *InMemoryBloomFilter) calculateMemoryUsage() uint64 {
    // 位数组内存
    bitArrayMemory := uint64(len(bf.bitArray)) * 8
    
    // 结构体本身的内存
    structMemory := uint64(unsafe.Sizeof(*bf))
    
    return bitArrayMemory + structMemory
}
```

## 使用示例

### 1. 基本使用

```go
// 创建布隆过滤器
config, err := domain.NewBloomFilterConfig(10000, 0.01) // 1万元素，1%假阳性
if err != nil {
    return err
}

bloomFilter := NewInMemoryBloomFilter(config)

// 添加元素
users := []string{"user:1", "user:2", "user:3"}
for _, user := range users {
    err := bloomFilter.Add(ctx, user)
    if err != nil {
        log.Printf("添加失败: %v", err)
    }
}

// 检查元素
if bloomFilter.HasKey(ctx, "user:1") {
    fmt.Println("user:1 可能存在")
}

if !bloomFilter.HasKey(ctx, "user:999") {
    fmt.Println("user:999 一定不存在")
}
```

### 2. 批量操作

```go
// 批量添加
userIDs := make([]string, 1000)
for i := 0; i < 1000; i++ {
    userIDs[i] = fmt.Sprintf("user:%d", i)
}

err := bloomFilter.AddBatch(ctx, userIDs)
if err != nil {
    log.Printf("批量添加失败: %v", err)
}

// 批量检查
checkIDs := []string{"user:1", "user:500", "user:1001"}
results := bloomFilter.HasKeysBatch(ctx, checkIDs)

for i, id := range checkIDs {
    if results[i] {
        fmt.Printf("%s 可能存在\n", id)
    } else {
        fmt.Printf("%s 一定不存在\n", id)
    }
}
```

### 3. 监控和调优

```go
// 获取统计信息
stats, err := bloomFilter.Stats(ctx)
if err != nil {
    log.Printf("获取统计失败: %v", err)
    return
}

// 监控关键指标
fmt.Printf("已添加元素: %d\n", stats.AddedElements())
fmt.Printf("设置位数: %d\n", stats.SetBits())
fmt.Printf("负载因子: %.4f\n", stats.LoadFactor())
fmt.Printf("估算假阳性率: %.4f\n", stats.EstimatedFalsePositiveRate())
fmt.Printf("内存使用: %d bytes\n", stats.MemoryUsage())

// 过载检查
if stats.IsOverloaded() {
    log.Println("警告: 布隆过滤器过载，建议增加容量")
}

// 效率检查
if stats.EfficiencyRatio() < 0.8 {
    log.Println("警告: 布隆过滤器效率较低，建议调整参数")
}
```

### 4. 与缓存集成

```go
// 创建带布隆过滤器的缓存
cache := NewMaxMemoryCache(1024 * 1024)
bloomFilter := NewInMemoryBloomFilter(config)

loadFunc := func(ctx context.Context, key string) (any, error) {
    return loadFromDatabase(key)
}

bloomCache := NewBloomFilterCacheSimple(cache, bloomFilter, loadFunc)

// 使用时自动利用布隆过滤器优化
user, err := bloomCache.Get(ctx, "user:123")
if err != nil {
    if errors.Is(err, ErrKeyNotFound) {
        // 布隆过滤器确定不存在，避免了数据库查询
        fmt.Println("用户不存在，已被布隆过滤器过滤")
    }
    return err
}
```

## 注意事项

### 1. 参数选择

```go
// ✅ 推荐：根据实际需求选择参数
config, err := domain.NewBloomFilterConfig(
    expectedElements,  // 根据实际数据量设置
    0.01,             // 1%假阳性率，平衡内存和准确性
)

// ❌ 避免：假阳性率过低导致内存浪费
config, err := domain.NewBloomFilterConfig(1000, 0.0001) // 0.01%假阳性率
```

### 2. 容量规划

```go
// ✅ 推荐：预留一定的容量余量
actualElements := getCurrentElementCount()
expectedElements := actualElements * 120 / 100  // 预留20%余量

config, err := domain.NewBloomFilterConfig(expectedElements, 0.01)
```

### 3. 重置策略

```go
// 定期重置过载的布隆过滤器
func (bf *InMemoryBloomFilter) checkAndReset() {
    stats, err := bf.Stats(context.Background())
    if err != nil {
        return
    }
    
    if stats.IsOverloaded() {
        log.Println("布隆过滤器过载，执行重置")
        bf.Clear(context.Background())
    }
}

// 定期检查
go func() {
    ticker := time.NewTicker(time.Hour)
    defer ticker.Stop()
    
    for {
        select {
        case <-ticker.C:
            bf.checkAndReset()
        }
    }
}()
```

### 4. 持久化考虑

```go
// 布隆过滤器状态持久化（可选）
func (bf *InMemoryBloomFilter) Save(filename string) error {
    bf.mu.RLock()
    defer bf.mu.RUnlock()
    
    data := struct {
        Config     domain.BloomFilterConfig
        BitArray   []uint64
        AddedCount uint64
    }{
        Config:     bf.config,
        BitArray:   bf.bitArray,
        AddedCount: bf.addedCount,
    }
    
    file, err := os.Create(filename)
    if err != nil {
        return err
    }
    defer file.Close()
    
    encoder := gob.NewEncoder(file)
    return encoder.Encode(data)
}

func LoadInMemoryBloomFilter(filename string) (*InMemoryBloomFilter, error) {
    file, err := os.Open(filename)
    if err != nil {
        return nil, err
    }
    defer file.Close()
    
    var data struct {
        Config     domain.BloomFilterConfig
        BitArray   []uint64
        AddedCount uint64
    }
    
    decoder := gob.NewDecoder(file)
    if err := decoder.Decode(&data); err != nil {
        return nil, err
    }
    
    return &InMemoryBloomFilter{
        config:     data.Config,
        bitArray:   data.BitArray,
        hashFunc:   data.Config.HashFunc(),
        addedCount: data.AddedCount,
    }, nil
}
```
