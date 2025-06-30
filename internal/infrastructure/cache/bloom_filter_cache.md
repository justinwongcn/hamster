# bloom_filter_cache.go - 布隆过滤器缓存实现

## 文件概述

`bloom_filter_cache.go` 实现了带布隆过滤器的读透缓存，通过布隆过滤器预先过滤不存在的键，有效防止缓存穿透。该实现结合了读透缓存模式和布隆过滤器的优势，在保证数据一致性的同时显著减少对底层数据源的无效查询。

## 核心功能

### 1. BloomFilterCache 结构体

```go
type BloomFilterCache struct {
    domainCache.Repository                    // 嵌入领域仓储接口
    bloomFilter            domainCache.BloomFilter // 布隆过滤器
    loadFunc               func(ctx context.Context, key string) (any, error) // 数据加载函数
    expiration             time.Duration      // 缓存过期时间
    autoAddToBloom         bool               // 是否自动将成功加载的键添加到布隆过滤器
    logFunc                func(format string, args ...any) // 日志函数
    g                      singleflight.Group // 防止缓存击穿
}
```

**设计特点：**

- 嵌入Repository接口，支持所有标准缓存操作
- 集成布隆过滤器，防止缓存穿透
- 使用SingleFlight防止缓存击穿
- 支持自动和手动管理布隆过滤器
- 提供详细的日志记录功能

### 2. BloomFilterCacheConfig 配置结构

```go
type BloomFilterCacheConfig struct {
    Repository     domainCache.Repository                                    // 底层缓存仓储
    BloomFilter    domainCache.BloomFilter                                   // 布隆过滤器
    LoadFunc       func(ctx context.Context, key string) (any, error)       // 数据加载函数
    Expiration     time.Duration                                             // 缓存过期时间
    AutoAddToBloom bool                                                      // 是否自动将成功加载的键添加到布隆过滤器
    LogFunc        func(format string, args ...any)                         // 日志函数
}
```

**配置选项：**

- **Repository**: 底层缓存存储实现
- **BloomFilter**: 布隆过滤器实例
- **LoadFunc**: 缓存未命中时的数据加载函数
- **Expiration**: 默认缓存过期时间
- **AutoAddToBloom**: 是否自动维护布隆过滤器
- **LogFunc**: 可选的日志记录函数

## 主要方法

### 1. 构造函数

#### NewBloomFilterCache

```go
func NewBloomFilterCache(config BloomFilterCacheConfig) *BloomFilterCache
```

使用完整配置创建布隆过滤器缓存实例。

#### NewBloomFilterCacheSimple

```go
func NewBloomFilterCacheSimple(
    repository domainCache.Repository,
    bloomFilter domainCache.BloomFilter,
    loadFunc func(ctx context.Context, key string) (any, error),
) *BloomFilterCache
```

使用简化参数创建实例，采用默认配置。

**示例：**

```go
// 完整配置方式
config := BloomFilterCacheConfig{
    Repository:     memoryCache,
    BloomFilter:    bloomFilter,
    LoadFunc:       loadFromDatabase,
    Expiration:     time.Hour,
    AutoAddToBloom: true,
    LogFunc:        log.Printf,
}
cache := NewBloomFilterCache(config)

// 简化方式
cache := NewBloomFilterCacheSimple(memoryCache, bloomFilter, loadFromDatabase)
```

### 2. 核心缓存操作

#### Get - 带布隆过滤器的读透获取

```go
func (bfc *BloomFilterCache) Get(ctx context.Context, key string) (any, error)
```

**执行流程：**

1. 先从缓存获取数据
2. 如果缓存命中，直接返回
3. 如果缓存未命中，检查布隆过滤器
4. 如果布隆过滤器返回false，直接返回键不存在
5. 如果布隆过滤器返回true，使用SingleFlight加载数据
6. 加载成功后更新缓存，并可选地添加到布隆过滤器

**示例：**

```go
// 定义数据加载函数
loadFunc := func(ctx context.Context, key string) (any, error) {
    return database.GetUser(key)
}

cache := NewBloomFilterCacheSimple(memoryCache, bloomFilter, loadFunc)

// 获取数据
user, err := cache.Get(ctx, "user:123")
if err != nil {
    if errors.Is(err, ErrKeyNotFound) {
        fmt.Println("用户不存在（被布隆过滤器过滤或数据库中确实不存在）")
    } else {
        log.Printf("获取用户失败: %v", err)
    }
    return
}

fmt.Printf("用户信息: %v\n", user)
```

#### Set - 设置缓存并更新布隆过滤器

```go
func (bfc *BloomFilterCache) Set(ctx context.Context, key string, val any, expiration time.Duration) error
```

**执行逻辑：**

1. 设置缓存数据
2. 如果启用自动添加，将键添加到布隆过滤器

**示例：**

```go
user := User{ID: "123", Name: "John"}
err := cache.Set(ctx, "user:123", user, time.Hour)
if err != nil {
    log.Printf("设置缓存失败: %v", err)
}
```

#### Delete - 删除缓存

```go
func (bfc *BloomFilterCache) Delete(ctx context.Context, key string) error
```

**注意事项：**

- 只删除缓存中的数据
- 布隆过滤器不支持删除操作，键会继续存在于布隆过滤器中
- 这是布隆过滤器的固有特性，不会影响正确性

### 3. 布隆过滤器管理

#### GetBloomFilterStats - 获取统计信息

```go
func (bfc *BloomFilterCache) GetBloomFilterStats(ctx context.Context) (domainCache.BloomFilterStats, error)
```

#### ClearBloomFilter - 清空布隆过滤器

```go
func (bfc *BloomFilterCache) ClearBloomFilter(ctx context.Context) error
```

#### AddKeyToBloomFilter - 手动添加键

```go
func (bfc *BloomFilterCache) AddKeyToBloomFilter(ctx context.Context, key string) error
```

#### HasKeyInBloomFilter - 检查键是否存在

```go
func (bfc *BloomFilterCache) HasKeyInBloomFilter(ctx context.Context, key string) bool
```

**示例：**

```go
// 获取布隆过滤器统计信息
stats, err := cache.GetBloomFilterStats(ctx)
if err == nil {
    fmt.Printf("布隆过滤器统计:\n")
    fmt.Printf("  已添加元素: %d\n", stats.AddedElements())
    fmt.Printf("  假阳性率: %.4f\n", stats.EstimatedFalsePositiveRate())
    fmt.Printf("  负载因子: %.4f\n", stats.LoadFactor())
}

// 手动添加键到布隆过滤器
err = cache.AddKeyToBloomFilter(ctx, "user:456")
if err != nil {
    log.Printf("添加键到布隆过滤器失败: %v", err)
}

// 检查键是否在布隆过滤器中
exists := cache.HasKeyInBloomFilter(ctx, "user:456")
fmt.Printf("键是否在布隆过滤器中: %v\n", exists)
```

### 4. 配置管理

#### 自动添加配置

```go
func (bfc *BloomFilterCache) SetAutoAddToBloom(autoAdd bool)
func (bfc *BloomFilterCache) IsAutoAddToBloomEnabled() bool
```

#### 过期时间配置

```go
func (bfc *BloomFilterCache) GetExpiration() time.Duration
func (bfc *BloomFilterCache) SetExpiration(expiration time.Duration)
```

#### 加载函数配置

```go
func (bfc *BloomFilterCache) SetLoadFunc(loadFunc func(ctx context.Context, key string) (any, error))
func (bfc *BloomFilterCache) GetLoadFunc() func(ctx context.Context, key string) (any, error)
```

#### 日志函数配置

```go
func (bfc *BloomFilterCache) SetLogFunc(logFunc func(format string, args ...any))
```

## 使用示例

### 1. 基本使用

```go
package main

import (
    "context"
    "fmt"
    "log"
    "time"
    
    "github.com/justinwongcn/hamster/internal/infrastructure/cache"
    "github.com/justinwongcn/hamster/internal/domain/cache"
)

func main() {
    // 创建底层缓存
    memoryCache := cache.NewMaxMemoryCache(1024 * 1024) // 1MB
    
    // 创建布隆过滤器
    bloomConfig, _ := domain.NewBloomFilterConfig(10000, 0.01) // 1万元素，1%假阳性
    bloomFilter := cache.NewInMemoryBloomFilter(bloomConfig)
    
    // 定义数据加载函数
    loadFunc := func(ctx context.Context, key string) (any, error) {
        // 模拟从数据库加载
        if key == "user:123" {
            return map[string]string{"id": "123", "name": "John"}, nil
        }
        return nil, cache.ErrKeyNotFound
    }
    
    // 创建布隆过滤器缓存
    bloomCache := cache.NewBloomFilterCacheSimple(memoryCache, bloomFilter, loadFunc)
    
    // 设置日志函数
    bloomCache.SetLogFunc(log.Printf)
    
    ctx := context.Background()
    
    // 第一次获取（会触发加载）
    user, err := bloomCache.Get(ctx, "user:123")
    if err == nil {
        fmt.Printf("第一次获取用户: %v\n", user)
    }
    
    // 第二次获取（从缓存获取）
    user, err = bloomCache.Get(ctx, "user:123")
    if err == nil {
        fmt.Printf("第二次获取用户: %v\n", user)
    }
    
    // 获取不存在的用户（被布隆过滤器过滤）
    _, err = bloomCache.Get(ctx, "user:999")
    if err != nil {
        fmt.Printf("获取不存在用户的错误: %v\n", err)
    }
}
```

### 2. 高级配置使用

```go
func demonstrateAdvancedConfig() {
    // 创建自定义配置
    config := cache.BloomFilterCacheConfig{
        Repository:     cache.NewMaxMemoryCache(2 * 1024 * 1024), // 2MB
        BloomFilter:    createBloomFilter(),
        LoadFunc:       createLoadFunc(),
        Expiration:     30 * time.Minute, // 30分钟过期
        AutoAddToBloom: true,             // 自动添加到布隆过滤器
        LogFunc:        createLogger(),   // 自定义日志函数
    }
    
    bloomCache := cache.NewBloomFilterCache(config)
    
    ctx := context.Background()
    
    // 测试缓存操作
    testKeys := []string{"product:1", "product:2", "product:999"}
    
    for _, key := range testKeys {
        value, err := bloomCache.Get(ctx, key)
        if err != nil {
            fmt.Printf("键 %s: 错误 - %v\n", key, err)
        } else {
            fmt.Printf("键 %s: 值 - %v\n", key, value)
        }
    }
    
    // 查看布隆过滤器统计
    stats, err := bloomCache.GetBloomFilterStats(ctx)
    if err == nil {
        fmt.Printf("\n布隆过滤器统计:\n")
        fmt.Printf("  已添加元素: %d\n", stats.AddedElements())
        fmt.Printf("  假阳性率: %.4f\n", stats.EstimatedFalsePositiveRate())
    }
}

func createBloomFilter() domain.BloomFilter {
    config, _ := domain.NewBloomFilterConfig(50000, 0.01)
    return cache.NewInMemoryBloomFilter(config)
}

func createLoadFunc() func(context.Context, string) (any, error) {
    return func(ctx context.Context, key string) (any, error) {
        // 模拟数据库查询延迟
        time.Sleep(10 * time.Millisecond)
        
        // 模拟数据存在性检查
        if strings.HasPrefix(key, "product:") {
            id := strings.TrimPrefix(key, "product:")
            if id != "999" { // 999不存在
                return map[string]string{
                    "id":   id,
                    "name": fmt.Sprintf("Product %s", id),
                }, nil
            }
        }
        
        return nil, cache.ErrKeyNotFound
    }
}

func createLogger() func(string, ...any) {
    return func(format string, args ...any) {
        log.Printf("[BloomFilterCache] "+format, args...)
    }
}
```

### 3. 性能测试和监控

```go
func demonstratePerformanceMonitoring() {
    bloomCache := createBloomFilterCache()
    ctx := context.Background()
    
    // 预热布隆过滤器
    fmt.Println("预热布隆过滤器...")
    for i := 1; i <= 1000; i++ {
        key := fmt.Sprintf("user:%d", i)
        bloomCache.AddKeyToBloomFilter(ctx, key)
    }
    
    // 性能测试
    fmt.Println("开始性能测试...")
    
    start := time.Now()
    hitCount := 0
    missCount := 0
    filteredCount := 0
    
    // 测试1000次随机访问
    for i := 0; i < 1000; i++ {
        key := fmt.Sprintf("user:%d", rand.Intn(2000)) // 0-1999，一半存在一半不存在
        
        _, err := bloomCache.Get(ctx, key)
        if err == nil {
            hitCount++
        } else if errors.Is(err, cache.ErrKeyNotFound) {
            // 检查是否被布隆过滤器过滤
            if !bloomCache.HasKeyInBloomFilter(ctx, key) {
                filteredCount++
            } else {
                missCount++
            }
        }
    }
    
    duration := time.Since(start)
    
    fmt.Printf("性能测试结果:\n")
    fmt.Printf("  总耗时: %v\n", duration)
    fmt.Printf("  平均延迟: %v\n", duration/1000)
    fmt.Printf("  缓存命中: %d\n", hitCount)
    fmt.Printf("  缓存未命中: %d\n", missCount)
    fmt.Printf("  布隆过滤器过滤: %d\n", filteredCount)
    fmt.Printf("  过滤效率: %.2f%%\n", float64(filteredCount)/float64(filteredCount+missCount)*100)
}
```

### 4. 错误处理和恢复

```go
func demonstrateErrorHandling() {
    bloomCache := createBloomFilterCache()
    ctx := context.Background()
    
    // 设置一个可能失败的加载函数
    bloomCache.SetLoadFunc(func(ctx context.Context, key string) (any, error) {
        if strings.Contains(key, "error") {
            return nil, errors.New("模拟数据库错误")
        }
        
        if strings.Contains(key, "timeout") {
            time.Sleep(2 * time.Second) // 模拟超时
            return nil, context.DeadlineExceeded
        }
        
        return fmt.Sprintf("data for %s", key), nil
    })
    
    testCases := []string{
        "normal:key",
        "error:key",
        "timeout:key",
        "nonexistent:key",
    }
    
    for _, key := range testCases {
        fmt.Printf("测试键: %s\n", key)
        
        // 设置超时上下文
        timeoutCtx, cancel := context.WithTimeout(ctx, time.Second)
        
        value, err := bloomCache.Get(timeoutCtx, key)
        if err != nil {
            switch {
            case errors.Is(err, cache.ErrKeyNotFound):
                fmt.Printf("  结果: 键不存在\n")
            case errors.Is(err, context.DeadlineExceeded):
                fmt.Printf("  结果: 操作超时\n")
            case errors.Is(err, cache.ErrFailedToRefreshCache):
                fmt.Printf("  结果: 缓存刷新失败，但数据已加载\n")
            default:
                fmt.Printf("  结果: 其他错误 - %v\n", err)
            }
        } else {
            fmt.Printf("  结果: 成功获取 - %v\n", value)
        }
        
        cancel()
        fmt.Println()
    }
}
```

### 5. 动态配置管理

```go
func demonstrateDynamicConfiguration() {
    bloomCache := createBloomFilterCache()
    ctx := context.Background()
    
    fmt.Printf("初始配置:\n")
    fmt.Printf("  自动添加到布隆过滤器: %v\n", bloomCache.IsAutoAddToBloomEnabled())
    fmt.Printf("  缓存过期时间: %v\n", bloomCache.GetExpiration())
    
    // 动态调整配置
    fmt.Println("\n调整配置...")
    bloomCache.SetAutoAddToBloom(false)
    bloomCache.SetExpiration(5 * time.Minute)
    
    fmt.Printf("调整后配置:\n")
    fmt.Printf("  自动添加到布隆过滤器: %v\n", bloomCache.IsAutoAddToBloomEnabled())
    fmt.Printf("  缓存过期时间: %v\n", bloomCache.GetExpiration())
    
    // 测试配置效果
    key := "test:dynamic:config"
    
    // 手动添加到布隆过滤器
    bloomCache.AddKeyToBloomFilter(ctx, key)
    
    // 设置缓存（不会自动添加到布隆过滤器，因为已禁用）
    bloomCache.Set(ctx, key, "test value", time.Minute)
    
    // 验证
    value, err := bloomCache.Get(ctx, key)
    if err == nil {
        fmt.Printf("获取到值: %v\n", value)
    }
    
    // 清理布隆过滤器测试
    fmt.Println("\n清理布隆过滤器...")
    err = bloomCache.ClearBloomFilter(ctx)
    if err == nil {
        fmt.Println("布隆过滤器已清空")
        
        // 验证清空效果
        exists := bloomCache.HasKeyInBloomFilter(ctx, key)
        fmt.Printf("键是否还在布隆过滤器中: %v\n", exists)
    }
}
```

## 性能特性

### 时间复杂度

- **Get操作**: O(1) - 缓存查找 + O(k) - 布隆过滤器检查
- **Set操作**: O(1) - 缓存设置 + O(k) - 布隆过滤器添加
- **Delete操作**: O(1) - 缓存删除

### 空间复杂度

- **缓存存储**: O(n) - n为缓存条目数量
- **布隆过滤器**: O(m) - m为位数组大小

### 性能优势

1. **防止缓存穿透**: 布隆过滤器有效过滤不存在的键
2. **防止缓存击穿**: SingleFlight确保同一键只有一个加载操作
3. **减少数据库压力**: 显著减少对底层数据源的查询

## 注意事项

### 1. 布隆过滤器特性

```go
// ✅ 理解：布隆过滤器的特性
// - false表示键一定不存在
// - true表示键可能存在（有假阳性）
// - 不支持删除操作

// 正确的使用方式
if !bloomCache.HasKeyInBloomFilter(ctx, key) {
    // 键一定不存在，可以直接返回
    return nil, ErrKeyNotFound
}
// 键可能存在，需要进一步检查
```

### 2. 自动添加配置

```go
// ✅ 推荐：根据使用场景选择是否自动添加
// 读多写少的场景：启用自动添加
bloomCache.SetAutoAddToBloom(true)

// 写多读少的场景：可能需要手动管理
bloomCache.SetAutoAddToBloom(false)
```

### 3. 过期时间设置

```go
// ✅ 推荐：设置合理的过期时间
bloomCache.SetExpiration(time.Hour) // 根据数据特性设置

// ❌ 避免：过期时间过短导致频繁加载
bloomCache.SetExpiration(time.Second) // 太短
```

### 4. 错误处理

```go
// ✅ 推荐：区分不同类型的错误
value, err := bloomCache.Get(ctx, key)
if err != nil {
    switch {
    case errors.Is(err, ErrKeyNotFound):
        // 键不存在（被布隆过滤器过滤或确实不存在）
    case errors.Is(err, ErrFailedToRefreshCache):
        // 数据加载成功但缓存更新失败
        // 可以使用返回的value，但要注意缓存不一致
    default:
        // 其他错误
    }
}
```

### 5. 监控和维护

```go
// ✅ 推荐：定期监控布隆过滤器状态
go func() {
    ticker := time.NewTicker(time.Minute)
    defer ticker.Stop()
    
    for {
        select {
        case <-ticker.C:
            stats, err := bloomCache.GetBloomFilterStats(ctx)
            if err == nil {
                if stats.IsOverloaded() {
                    log.Println("警告: 布隆过滤器过载")
                }
                
                if stats.EstimatedFalsePositiveRate() > 0.05 {
                    log.Println("警告: 假阳性率过高")
                }
            }
        }
    }
}()
```
