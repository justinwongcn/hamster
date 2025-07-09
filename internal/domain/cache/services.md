# services.go - 缓存领域服务

## 文件概述

`services.go` 定义了缓存领域的核心服务和策略，包括淘汰策略、缓存服务、写回服务和缓存策略。该文件遵循DDD设计原则，将缓存的业务逻辑和策略封装在领域服务中，为不同的缓存模式提供灵活的策略支持。

## 核心功能

### 1. EvictionStrategy 淘汰策略接口

```go
type EvictionStrategy interface {
    SelectForEviction(entries []*Entry) *Entry
    OnAccess(entry *Entry)
    OnAdd(entry *Entry)
    OnRemove(entry *Entry)
}
```

**核心方法：**

- **SelectForEviction**: 从候选条目中选择要淘汰的条目
- **OnAccess**: 条目被访问时的回调处理
- **OnAdd**: 添加新条目时的回调处理
- **OnRemove**: 移除条目时的回调处理

### 2. LRUEvictionStrategy LRU淘汰策略

#### 实现逻辑

```go
type LRUEvictionStrategy struct{}

func (s *LRUEvictionStrategy) SelectForEviction(entries []*Entry) *Entry {
    if len(entries) == 0 {
        return nil
    }
    
    oldest := entries[0]
    for _, entry := range entries[1:] {
        if entry.AccessedAt().Before(oldest.AccessedAt()) {
            oldest = entry
        }
    }
    return oldest
}
```

**特点：**

- 选择最近最少使用的条目进行淘汰
- 基于访问时间进行判断
- 适用于时间局部性强的场景

**示例：**

```go
// 创建LRU策略
lruStrategy := NewLRUEvictionStrategy()

// 模拟条目列表
entries := []*Entry{entry1, entry2, entry3}

// 选择要淘汰的条目
toEvict := lruStrategy.SelectForEviction(entries)
if toEvict != nil {
    fmt.Printf("选择淘汰: %s\n", toEvict.Key().String())
}
```

### 3. FIFOEvictionStrategy FIFO淘汰策略

#### 实现逻辑

```go
type FIFOEvictionStrategy struct{}

func (s *FIFOEvictionStrategy) SelectForEviction(entries []*Entry) *Entry {
    if len(entries) == 0 {
        return nil
    }
    
    oldest := entries[0]
    for _, entry := range entries[1:] {
        if entry.CreatedAt().Before(oldest.CreatedAt()) {
            oldest = entry
        }
    }
    return oldest
}
```

**特点：**

- 选择最先创建的条目进行淘汰
- 基于创建时间进行判断
- 实现简单，适用于顺序访问场景

### 4. CacheService 缓存领域服务

#### 结构定义

```go
type CacheService struct {
    evictionStrategy EvictionStrategy
}
```

#### 核心方法

##### 验证方法

```go
func (s *CacheService) ValidateKey(key string) error
func (s *CacheService) ValidateExpiration(duration time.Duration) error
```

**示例：**

```go
service := NewCacheService(NewLRUEvictionStrategy())

// 验证键
if err := service.ValidateKey("user:123"); err != nil {
    log.Printf("键验证失败: %v", err)
}

// 验证过期时间
if err := service.ValidateExpiration(time.Hour); err != nil {
    log.Printf("过期时间验证失败: %v", err)
}
```

##### 淘汰管理

```go
func (s *CacheService) ShouldEvict(instance *CacheInstance) bool
func (s *CacheService) SelectForEviction(instance *CacheInstance) *Entry
func (s *CacheService) CleanExpiredEntries(instance *CacheInstance) int
```

**示例：**

```go
// 检查是否需要淘汰
if service.ShouldEvict(cacheInstance) {
    // 选择要淘汰的条目
    toEvict := service.SelectForEviction(cacheInstance)
    if toEvict != nil {
        cacheInstance.RemoveEntry(toEvict.Key())
        fmt.Printf("淘汰了条目: %s\n", toEvict.Key().String())
    }
}

// 清理过期条目
cleanedCount := service.CleanExpiredEntries(cacheInstance)
fmt.Printf("清理了 %d 个过期条目\n", cleanedCount)
```

### 5. WriteBackService 写回缓存服务

#### 结构定义

```go
type WriteBackService struct {
    *CacheService
    flushInterval time.Duration
    batchSize     int
}
```

#### 构造函数

```go
func NewWriteBackService(evictionStrategy EvictionStrategy, flushInterval time.Duration, batchSize int) *WriteBackService
```

**示例：**

```go
// 创建写回服务
writeBackService := NewWriteBackService(
    NewLRUEvictionStrategy(),
    time.Minute,  // 每分钟刷新一次
    100,          // 批量大小100
)

fmt.Printf("刷新间隔: %v\n", writeBackService.FlushInterval())
fmt.Printf("批量大小: %d\n", writeBackService.BatchSize())
```

#### 刷新管理

```go
func (s *WriteBackService) ShouldFlush(instance *CacheInstance, lastFlushTime time.Time) bool
func (s *WriteBackService) GetFlushBatch(instance *CacheInstance) []*Entry
```

**刷新条件：**

1. 距离上次刷新时间超过刷新间隔
2. 脏数据数量达到批量大小

**示例：**

```go
lastFlushTime := time.Now().Add(-2 * time.Minute)

// 检查是否需要刷新
if writeBackService.ShouldFlush(cacheInstance, lastFlushTime) {
    // 获取要刷新的批次
    batch := writeBackService.GetFlushBatch(cacheInstance)
    
    fmt.Printf("需要刷新 %d 个脏数据条目\n", len(batch))
    
    // 执行刷新操作
    for _, entry := range batch {
        // 刷新到持久化存储
        err := flushToPersistentStore(entry)
        if err == nil {
            entry.MarkClean()
        }
    }
}
```

#### 刷新操作处理

```go
func (s *WriteBackService) ValidateFlushOperation(entries []*Entry, storer func(ctx context.Context, key string, val any) error) error
func (s *WriteBackService) ProcessFlushResult(instance *CacheInstance, entries []*Entry, errors []error)
```

**示例：**

```go
// 定义存储函数
storer := func(ctx context.Context, key string, val any) error {
    return database.Save(key, val)
}

// 获取要刷新的条目
batch := writeBackService.GetFlushBatch(cacheInstance)

// 验证刷新操作
if err := writeBackService.ValidateFlushOperation(batch, storer); err != nil {
    log.Printf("刷新验证失败: %v", err)
    return
}

// 执行刷新并收集错误
var errors []error
for _, entry := range batch {
    err := storer(ctx, entry.Key().String(), entry.Value().Data())
    errors = append(errors, err)
}

// 处理刷新结果
writeBackService.ProcessFlushResult(cacheInstance, batch, errors)
```

### 6. CachePolicy 缓存策略

#### 结构定义

```go
type CachePolicy struct {
    maxSize          int64
    maxMemory        int64
    defaultTTL       time.Duration
    evictionStrategy EvictionStrategy
    enableWriteBack  bool
    writeBackConfig  *WriteBackConfig
}
```

#### 建造者模式

```go
func NewCachePolicy() *CachePolicy
func (p *CachePolicy) WithMaxSize(maxSize int64) *CachePolicy
func (p *CachePolicy) WithMaxMemory(maxMemory int64) *CachePolicy
func (p *CachePolicy) WithDefaultTTL(ttl time.Duration) *CachePolicy
func (p *CachePolicy) WithEvictionStrategy(strategy EvictionStrategy) *CachePolicy
func (p *CachePolicy) WithWriteBack(config *WriteBackConfig) *CachePolicy
```

**示例：**

```go
// 创建缓存策略
policy := NewCachePolicy().
    WithMaxSize(1000).
    WithMaxMemory(100 * 1024 * 1024). // 100MB
    WithDefaultTTL(time.Hour).
    WithEvictionStrategy(NewLRUEvictionStrategy()).
    WithWriteBack(&WriteBackConfig{
        FlushInterval: time.Minute,
        BatchSize:     50,
        MaxRetries:    3,
        RetryDelay:    time.Second,
    })

fmt.Printf("策略配置:\n")
fmt.Printf("  最大条目数: %d\n", policy.MaxSize())
fmt.Printf("  最大内存: %d bytes\n", policy.MaxMemory())
fmt.Printf("  默认TTL: %v\n", policy.DefaultTTL())
fmt.Printf("  写回模式: %v\n", policy.IsWriteBackEnabled())
```

#### WriteBackConfig 写回配置

```go
type WriteBackConfig struct {
    FlushInterval time.Duration // 刷新间隔
    BatchSize     int           // 批量大小
    MaxRetries    int           // 最大重试次数
    RetryDelay    time.Duration // 重试延迟
}
```

## 使用示例

### 1. 基本缓存服务使用

```go
package main

import (
    "fmt"
    "time"
    
    "github.com/justinwongcn/hamster/internal/domain/cache"
)

func main() {
    // 创建缓存服务
    service := cache.NewCacheService(cache.NewLRUEvictionStrategy())
    
    // 创建缓存实例
    instance := cache.NewCacheInstance("test_cache", 3, 0)
    
    // 添加一些条目
    for i := 1; i <= 4; i++ {
        key, _ := cache.NewCacheKey(fmt.Sprintf("key:%d", i))
        value, _ := cache.NewCacheValue(fmt.Sprintf("value_%d", i), 50)
        expiration, _ := cache.NewExpiration(time.Hour)
        entry := cache.NewEntry(key, value, expiration)
        
        // 检查是否需要淘汰
        if service.ShouldEvict(instance) {
            toEvict := service.SelectForEviction(instance)
            if toEvict != nil {
                instance.RemoveEntry(toEvict.Key())
                fmt.Printf("淘汰了: %s\n", toEvict.Key().String())
            }
        }
        
        instance.SetEntry(entry)
        fmt.Printf("添加了: key:%d\n", i)
    }
    
    fmt.Printf("最终缓存大小: %d\n", instance.Size())
}
```

### 2. 写回缓存服务使用

```go
func demonstrateWriteBack() {
    // 创建写回服务
    writeBackService := cache.NewWriteBackService(
        cache.NewLRUEvictionStrategy(),
        30*time.Second, // 30秒刷新间隔
        5,              // 批量大小5
    )
    
    // 创建缓存实例
    instance := cache.NewCacheInstance("writeback_cache", 0, 0)
    
    // 添加一些脏数据
    for i := 1; i <= 10; i++ {
        key, _ := cache.NewCacheKey(fmt.Sprintf("user:%d", i))
        value, _ := cache.NewCacheValue(fmt.Sprintf("data_%d", i), 30)
        expiration, _ := cache.NewExpiration(time.Hour)
        entry := cache.NewEntry(key, value, expiration)
        
        // 标记为脏数据
        entry.MarkDirty()
        instance.SetEntry(entry)
    }
    
    fmt.Printf("添加了 %d 个脏数据条目\n", len(instance.GetDirtyEntries()))
    
    // 模拟刷新检查
    lastFlushTime := time.Now().Add(-time.Minute) // 1分钟前
    
    if writeBackService.ShouldFlush(instance, lastFlushTime) {
        batch := writeBackService.GetFlushBatch(instance)
        fmt.Printf("需要刷新 %d 个条目\n", len(batch))
        
        // 模拟刷新操作
        var errors []error
        for _, entry := range batch {
            // 模拟存储操作
            fmt.Printf("刷新: %s\n", entry.Key().String())
            errors = append(errors, nil) // 假设都成功
        }
        
        // 处理刷新结果
        writeBackService.ProcessFlushResult(instance, batch, errors)
        
        fmt.Printf("刷新后脏数据条目数: %d\n", len(instance.GetDirtyEntries()))
    }
}
```

### 3. 策略模式使用

```go
func demonstrateStrategies() {
    // 测试不同的淘汰策略
    strategies := map[string]cache.EvictionStrategy{
        "LRU":  cache.NewLRUEvictionStrategy(),
        "FIFO": cache.NewFIFOEvictionStrategy(),
    }
    
    for name, strategy := range strategies {
        fmt.Printf("\n测试 %s 策略:\n", name)
        
        service := cache.NewCacheService(strategy)
        instance := cache.NewCacheInstance(fmt.Sprintf("%s_cache", name), 3, 0)
        
        // 添加条目
        for i := 1; i <= 5; i++ {
            key, _ := cache.NewCacheKey(fmt.Sprintf("item:%d", i))
            value, _ := cache.NewCacheValue(fmt.Sprintf("data_%d", i), 20)
            expiration, _ := cache.NewExpiration(time.Hour)
            entry := cache.NewEntry(key, value, expiration)
            
            // 模拟不同的访问时间
            if name == "LRU" && i <= 3 {
                time.Sleep(time.Millisecond) // 确保访问时间不同
                entry.MarkAccessed()
            }
            
            if service.ShouldEvict(instance) {
                toEvict := service.SelectForEviction(instance)
                if toEvict != nil {
                    instance.RemoveEntry(toEvict.Key())
                    fmt.Printf("  淘汰: %s\n", toEvict.Key().String())
                }
            }
            
            instance.SetEntry(entry)
            fmt.Printf("  添加: item:%d\n", i)
        }
    }
}
```

### 4. 缓存策略配置

```go
func demonstratePolicyConfiguration() {
    // 创建不同的缓存策略
    
    // 基础策略
    basicPolicy := cache.NewCachePolicy().
        WithMaxSize(1000).
        WithMaxMemory(50 * 1024 * 1024). // 50MB
        WithDefaultTTL(30 * time.Minute)
    
    // 写回策略
    writeBackPolicy := cache.NewCachePolicy().
        WithMaxSize(500).
        WithEvictionStrategy(cache.NewLRUEvictionStrategy()).
        WithWriteBack(&cache.WriteBackConfig{
            FlushInterval: time.Minute,
            BatchSize:     20,
            MaxRetries:    3,
            RetryDelay:    time.Second,
        })
    
    // FIFO策略
    fifoPolicy := cache.NewCachePolicy().
        WithMaxSize(2000).
        WithEvictionStrategy(cache.NewFIFOEvictionStrategy()).
        WithDefaultTTL(time.Hour)
    
    policies := map[string]*cache.CachePolicy{
        "基础策略": basicPolicy,
        "写回策略": writeBackPolicy,
        "FIFO策略": fifoPolicy,
    }
    
    for name, policy := range policies {
        fmt.Printf("\n%s配置:\n", name)
        fmt.Printf("  最大条目数: %d\n", policy.MaxSize())
        fmt.Printf("  最大内存: %d MB\n", policy.MaxMemory()/(1024*1024))
        fmt.Printf("  默认TTL: %v\n", policy.DefaultTTL())
        fmt.Printf("  写回模式: %v\n", policy.IsWriteBackEnabled())
        
        if policy.IsWriteBackEnabled() {
            config := policy.GetWriteBackConfig()
            fmt.Printf("  刷新间隔: %v\n", config.FlushInterval)
            fmt.Printf("  批量大小: %d\n", config.BatchSize)
        }
    }
}
```

## 设计原则

### 1. 策略模式

- 将淘汰算法封装为策略
- 支持运行时策略切换
- 便于添加新的淘汰策略

### 2. 领域服务

- 封装复杂的业务逻辑
- 协调多个领域对象
- 提供高层次的业务操作

### 3. 建造者模式

- 提供灵活的配置方式
- 支持链式调用
- 确保配置的完整性

### 4. 单一职责

- 每个服务专注特定功能
- 策略只关心淘汰逻辑
- 配置只管理参数设置

## 注意事项

### 1. 策略选择

```go
// ✅ 推荐：根据访问模式选择策略
// 时间局部性强的场景使用LRU
lruStrategy := cache.NewLRUEvictionStrategy()

// 顺序访问场景使用FIFO
fifoStrategy := cache.NewFIFOEvictionStrategy()
```

### 2. 写回配置

```go
// ✅ 推荐：合理设置写回参数
writeBackConfig := &cache.WriteBackConfig{
    FlushInterval: time.Minute,     // 不要太频繁
    BatchSize:     50,              // 平衡内存和性能
    MaxRetries:    3,               // 适当的重试次数
    RetryDelay:    time.Second,     // 避免重试风暴
}
```

### 3. 内存管理

```go
// ✅ 推荐：设置合理的内存限制
policy := cache.NewCachePolicy().
    WithMaxMemory(availableMemory * 80 / 100) // 使用80%可用内存
```

### 4. 性能监控

```go
// ✅ 推荐：监控淘汰效果
cleanedCount := service.CleanExpiredEntries(instance)
if cleanedCount > instance.Size()/2 {
    log.Println("警告: 大量条目过期，考虑调整TTL")
}
```
