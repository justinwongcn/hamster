# Infrastructure Cache 缓存基础设施包

缓存基础设施包提供了完整的缓存系统实现，包括多种缓存策略、淘汰算法、高级缓存模式和布隆过滤器等组件。该包是Hamster项目缓存系统的核心实现层，为上层应用提供高性能、可扩展的缓存服务。

## 📁 包结构

```
cache/
├── 基础缓存实现
│   ├── max_memory_cache.go          # 最大内存缓存实现
│   ├── build_in_map_cache.go        # 内置Map缓存实现
│   └── eviction_policy.go           # 淘汰策略接口定义
│
├── 淘汰策略实现
│   ├── lru_policy.go                # LRU淘汰策略
│   ├── fifo_policy.go               # FIFO淘汰策略
│   └── random_policy.go             # 随机淘汰策略
│
├── 高级缓存模式
│   ├── read_through_cache.go        # 读透缓存
│   ├── write_through_cache.go       # 写透缓存
│   └── write_back_cache.go          # 写回缓存
│
├── 布隆过滤器
│   ├── in_memory_bloom_filter.go    # 内存布隆过滤器
│   └── bloom_filter_cache.go        # 布隆过滤器缓存
│
└── 测试文件
    ├── benchmark_test.go             # 性能基准测试
    └── *_test.go                     # 各组件单元测试
```

## 🚀 主要功能

### 1. 基础缓存实现

#### MaxMemoryCache - 最大内存缓存
- **内存限制**: 支持最大内存容量限制
- **多种淘汰策略**: LRU、FIFO、Random等
- **并发安全**: 读写锁保证线程安全
- **性能监控**: 内置命中率统计

#### BuildInMapCache - 内置Map缓存
- **基于Go Map**: 简单高效的实现
- **自动过期**: 后台定时清理过期项
- **配置灵活**: 支持淘汰回调等选项
- **轻量级**: 适合简单缓存需求

### 2. 淘汰策略

#### LRU策略 (Least Recently Used)
- **最近最少使用**: 淘汰最久未访问的数据
- **双向链表**: O(1)时间复杂度操作
- **适用场景**: 具有时间局部性的访问模式

#### FIFO策略 (First In First Out)
- **先进先出**: 按插入顺序淘汰数据
- **单向链表**: 简单高效的实现
- **适用场景**: 顺序访问模式

#### Random策略
- **随机淘汰**: 随机选择数据进行淘汰
- **公平性**: 每个数据被淘汰概率相等
- **适用场景**: 无特殊访问模式要求

### 3. 高级缓存模式

#### ReadThroughCache - 读透缓存
- **自动加载**: 缓存未命中时自动从数据源加载
- **SingleFlight**: 防止缓存击穿
- **限流支持**: 可选的限流机制

#### WriteThroughCache - 写透缓存
- **强一致性**: 同时写入缓存和存储
- **事务性**: 存储失败时不更新缓存
- **限流降级**: 高负载时的降级策略

#### WriteBackCache - 写回缓存
- **高性能写入**: 只写缓存，异步刷新
- **批量刷新**: 基于时间和数量的刷新策略
- **脏数据管理**: 完整的脏数据跟踪

### 4. 布隆过滤器

#### InMemoryBloomFilter - 内存布隆过滤器
- **高效过滤**: 快速判断元素是否可能存在
- **可配置**: 支持自定义假阳性率和容量
- **统计信息**: 详细的使用统计

#### BloomFilterCache - 布隆过滤器缓存
- **防缓存穿透**: 有效过滤不存在的键
- **自动管理**: 可选的自动添加机制
- **性能优化**: 结合SingleFlight优化

## 🔧 快速上手

### 基础缓存使用

```go
import "github.com/justinwongcn/hamster/internal/infrastructure/cache"

// 创建最大内存缓存
memCache := cache.NewMaxMemoryCache(1024 * 1024) // 1MB

// 设置缓存
err := memCache.Set(ctx, "user:123", userData, time.Hour)

// 获取缓存
value, err := memCache.Get(ctx, "user:123")

// 删除缓存
err = memCache.Delete(ctx, "user:123")
```

### 淘汰策略使用

```go
// 创建LRU策略
lruPolicy := cache.NewLRUPolicy(100) // 容量100

// 创建带策略的缓存
memCache := cache.NewMaxMemoryCache(1024*1024, 
    cache.MaxMemoryCacheWithEvictionPolicy(lruPolicy))
```

### 高级缓存模式

```go
// 读透缓存
readThrough := &cache.ReadThroughCache{
    Repository: memCache,
    LoadFunc: func(ctx context.Context, key string) (any, error) {
        return database.Load(key)
    },
    Expiration: time.Hour,
}

// 写透缓存
writeThrough := &cache.WriteThroughCache{
    Repository: memCache,
    StoreFunc: func(ctx context.Context, key string, val any) error {
        return database.Save(key, val)
    },
}
```

### 布隆过滤器

```go
// 创建布隆过滤器
config, _ := domain.NewBloomFilterConfig(10000, 0.01)
bloomFilter := cache.NewInMemoryBloomFilter(config)

// 布隆过滤器缓存
bloomCache := cache.NewBloomFilterCacheSimple(
    memCache, bloomFilter, loadFunc)
```

## 🎯 架构设计

### 1. 分层架构
- **接口层**: 统一的缓存接口定义
- **策略层**: 可插拔的淘汰策略
- **实现层**: 具体的缓存实现
- **优化层**: 性能优化和高级功能

### 2. 设计模式
- **策略模式**: 淘汰策略的可插拔设计
- **装饰器模式**: 缓存功能的层次增强
- **工厂模式**: 缓存实例的创建管理
- **观察者模式**: 淘汰事件的回调机制

### 3. 并发安全
- **读写锁**: 保证并发访问安全
- **原子操作**: 关键计数器的原子更新
- **无锁设计**: 部分组件采用无锁算法

## 📊 性能特性

### 时间复杂度
- **基础操作**: O(1) - Get/Set/Delete
- **LRU策略**: O(1) - 所有操作
- **FIFO策略**: O(1) - 所有操作
- **Random策略**: O(1) - 所有操作

### 空间复杂度
- **缓存存储**: O(n) - n为缓存项数量
- **策略开销**: O(n) - 策略相关的数据结构
- **布隆过滤器**: O(m) - m为位数组大小

### 性能基准
```bash
# 运行性能基准测试
go test -bench=. -benchmem ./internal/infrastructure/cache/

# 典型性能指标
BenchmarkMaxMemoryCache_Set-8     1000000    1200 ns/op    128 B/op    2 allocs/op
BenchmarkMaxMemoryCache_Get-8     2000000     800 ns/op     64 B/op    1 allocs/op
BenchmarkLRUPolicy_Access-8       5000000     300 ns/op     32 B/op    1 allocs/op
```

## 🔍 监控和调试

### 统计信息
```go
// 获取缓存统计
stats := memCache.GetStats()
fmt.Printf("命中率: %.2f%%", stats.HitRate()*100)
fmt.Printf("内存使用: %d bytes", stats.MemoryUsage())

// 布隆过滤器统计
bloomStats := bloomFilter.GetStats()
fmt.Printf("假阳性率: %.4f", bloomStats.EstimatedFalsePositiveRate())
```

### 性能监控
```go
// 设置监控回调
memCache.OnEvicted(func(key string, val any) {
    log.Printf("缓存项被淘汰: %s", key)
})

// 定期监控
go func() {
    ticker := time.NewTicker(time.Minute)
    for range ticker.C {
        stats := memCache.GetStats()
        if stats.HitRate() < 0.8 {
            log.Printf("警告: 缓存命中率过低: %.2f%%", stats.HitRate()*100)
        }
    }
}()
```

## ⚠️ 最佳实践

### 1. 缓存策略选择
```go
// 时间局部性强 -> LRU
lruCache := cache.NewMaxMemoryCache(size, 
    cache.MaxMemoryCacheWithEvictionPolicy(cache.NewLRUPolicy(capacity)))

// 顺序访问 -> FIFO  
fifoCache := cache.NewMaxMemoryCache(size,
    cache.MaxMemoryCacheWithEvictionPolicy(cache.NewFIFOPolicy(capacity)))

// 无特殊模式 -> Random
randomCache := cache.NewMaxMemoryCache(size,
    cache.MaxMemoryCacheWithEvictionPolicy(cache.NewRandomPolicy(capacity)))
```

### 2. 内存管理
```go
// ✅ 推荐：设置合理的内存限制
cache := cache.NewMaxMemoryCache(100 * 1024 * 1024) // 100MB

// ✅ 推荐：监控内存使用
if cache.GetStats().MemoryUsage() > maxMemory*0.9 {
    log.Println("警告: 缓存内存使用接近上限")
}
```

### 3. 过期时间设置
```go
// ✅ 推荐：根据数据特性设置过期时间
cache.Set(ctx, "user_session", session, 30*time.Minute)  // 会话30分钟
cache.Set(ctx, "config", config, time.Hour)              // 配置1小时
cache.Set(ctx, "static_data", data, 24*time.Hour)        // 静态数据24小时
```

### 4. 错误处理
```go
// ✅ 推荐：正确处理缓存错误
value, err := cache.Get(ctx, key)
if err != nil {
    if errors.Is(err, cache.ErrKeyNotFound) {
        // 缓存未命中，从数据源加载
        value = loadFromDataSource(key)
        cache.Set(ctx, key, value, expiration)
    } else {
        // 其他错误，记录日志
        log.Printf("缓存操作失败: %v", err)
    }
}
```

## 🧪 测试指南

### 单元测试
```bash
# 运行所有测试
go test ./internal/infrastructure/cache/

# 运行特定组件测试
go test -run TestMaxMemoryCache ./internal/infrastructure/cache/

# 查看测试覆盖率
go test -cover ./internal/infrastructure/cache/
```

### 基准测试
```bash
# 运行性能基准测试
go test -bench=. ./internal/infrastructure/cache/

# 运行内存分析
go test -bench=. -memprofile=mem.prof ./internal/infrastructure/cache/

# 运行CPU分析
go test -bench=. -cpuprofile=cpu.prof ./internal/infrastructure/cache/
```

### 压力测试
```bash
# 并发压力测试
go test -race ./internal/infrastructure/cache/

# 长时间运行测试
go test -timeout=30m ./internal/infrastructure/cache/
```

## 🔄 扩展指南

### 添加新的淘汰策略
```go
// 1. 实现EvictionPolicy接口
type MyPolicy struct {
    // 策略状态
}

func (p *MyPolicy) KeyAccessed(ctx context.Context, key string) error {
    // 实现访问逻辑
}

func (p *MyPolicy) Evict(ctx context.Context) (string, error) {
    // 实现淘汰逻辑
}

// 2. 注册到工厂
func NewMyPolicy(capacity int) EvictionPolicy {
    return &MyPolicy{...}
}
```

### 添加新的缓存模式
```go
// 1. 实现Repository接口
type MyCache struct {
    // 缓存状态
}

func (c *MyCache) Get(ctx context.Context, key string) (any, error) {
    // 实现获取逻辑
}

func (c *MyCache) Set(ctx context.Context, key string, val any, expiration time.Duration) error {
    // 实现设置逻辑
}

// 2. 添加特殊功能
func (c *MyCache) MySpecialMethod() {
    // 特殊功能实现
}
```

## 📈 性能优化

### 1. 内存优化
- 使用对象池减少GC压力
- 合理设置缓存容量
- 定期清理过期数据

### 2. 并发优化
- 读写锁分离
- 减少锁持有时间
- 使用无锁数据结构

### 3. 算法优化
- 选择合适的淘汰策略
- 优化数据结构
- 减少内存分配

Infrastructure Cache包为Hamster项目提供了完整、高性能的缓存解决方案，支持多种使用场景和性能要求。
