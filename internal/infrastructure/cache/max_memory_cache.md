# max_memory_cache.go - 最大内存缓存实现

## 文件概述

`max_memory_cache.go` 实现了带内存限制和LRU淘汰策略的基础缓存。这是Hamster缓存系统的核心实现，提供了高性能的内存缓存功能，支持自动内存管理和过期清理。

## 核心功能

### 1. MaxMemoryCache 结构体

```go
type MaxMemoryCache struct {
    maxMemory    int64                    // 最大内存限制（字节）
    usedMemory   int64                    // 当前使用内存（字节）
    data         map[string]*Entry        // 数据存储
    lru          *tools.LRU[string]       // LRU淘汰算法
    mu           sync.RWMutex             // 读写锁
    stats        domain.CacheStats        // 统计信息
    stopCh       chan struct{}            // 停止信号
    cleanupTimer *time.Timer              // 清理定时器
}
```

**主要特性：**

- 内存限制控制，防止OOM
- LRU淘汰策略，自动清理最少使用的数据
- 线程安全的并发访问
- 自动过期清理机制
- 详细的统计信息收集

### 2. Entry 缓存条目

```go
type Entry struct {
    Key        string      // 缓存键
    Value      any         // 缓存值
    Size       int64       // 数据大小
    CreatedAt  time.Time   // 创建时间
    AccessedAt time.Time   // 最后访问时间
    Expiration time.Duration // 过期时间
}
```

**功能方法：**

- `IsExpired(now time.Time) bool` - 检查是否过期
- `UpdateAccessTime(now time.Time)` - 更新访问时间
- `CalculateSize() int64` - 计算数据大小

## 主要方法

### 1. 构造函数

```go
func NewMaxMemoryCache(maxMemory int64) *MaxMemoryCache
```

**参数：**

- `maxMemory`: 最大内存限制（字节）

**示例：**

```go
// 创建1GB内存限制的缓存
cache := NewMaxMemoryCache(1024 * 1024 * 1024)
```

### 2. 基础操作

#### Get - 获取缓存

```go
func (c *MaxMemoryCache) Get(ctx context.Context, key string) (any, error)
```

**实现逻辑：**

1. 验证键的有效性
2. 获取读锁
3. 查找缓存条目
4. 检查是否过期
5. 更新LRU和访问时间
6. 更新统计信息

**示例：**

```go
value, err := cache.Get(ctx, "user:123")
if err != nil {
    if errors.Is(err, ErrKeyNotFound) {
        // 处理键不存在
    }
    return err
}
```

#### Set - 设置缓存

```go
func (c *MaxMemoryCache) Set(ctx context.Context, key string, val any, expiration time.Duration) error
```

**实现逻辑：**

1. 验证输入参数
2. 计算数据大小
3. 获取写锁
4. 检查内存限制
5. 执行LRU淘汰（如需要）
6. 存储数据
7. 更新统计信息

**示例：**

```go
err := cache.Set(ctx, "user:123", userData, time.Hour)
if err != nil {
    log.Printf("设置缓存失败: %v", err)
}
```

#### Delete - 删除缓存

```go
func (c *MaxMemoryCache) Delete(ctx context.Context, key string) error
```

### 3. 高级操作

#### LoadAndDelete - 原子获取并删除

```go
func (c *MaxMemoryCache) LoadAndDelete(ctx context.Context, key string) (any, error)
```

#### Keys - 模式匹配键

```go
func (c *MaxMemoryCache) Keys(ctx context.Context, pattern string) ([]string, error)
```

**支持的模式：**

- `*` - 匹配所有键
- `prefix:*` - 匹配前缀
- `*:suffix` - 匹配后缀

#### Exists - 检查键是否存在

```go
func (c *MaxMemoryCache) Exists(ctx context.Context, key string) (bool, error)
```

### 4. 管理操作

#### Clear - 清空缓存

```go
func (c *MaxMemoryCache) Clear(ctx context.Context) error
```

#### Stats - 获取统计信息

```go
func (c *MaxMemoryCache) Stats(ctx context.Context) (map[string]any, error)
```

**返回的统计信息：**

```go
{
    "hits":         命中次数,
    "misses":       未命中次数,
    "hit_rate":     命中率,
    "sets":         设置次数,
    "deletes":      删除次数,
    "evictions":    淘汰次数,
    "size":         条目数量,
    "memory_usage": 内存使用量,
    "max_memory":   最大内存限制,
}
```

## 内存管理

### 1. 内存计算

```go
func calculateSize(value any) int64 {
    switch v := value.(type) {
    case string:
        return int64(len(v))
    case []byte:
        return int64(len(v))
    case int, int32, int64, float32, float64:
        return 8
    default:
        // 使用反射估算大小
        return estimateSize(v)
    }
}
```

### 2. LRU淘汰策略

```go
func (c *MaxMemoryCache) evictIfNeeded(newSize int64) error {
    for c.usedMemory + newSize > c.maxMemory {
        // 获取最少使用的键
        oldestKey := c.lru.RemoveOldest()
        if oldestKey == "" {
            return ErrCacheFull
        }
        
        // 删除对应的缓存条目
        if entry, exists := c.data[oldestKey]; exists {
            delete(c.data, oldestKey)
            c.usedMemory -= entry.Size
            c.stats = c.stats.IncrementEvictions()
        }
    }
    return nil
}
```

### 3. 自动清理机制

```go
func (c *MaxMemoryCache) startCleanup() {
    c.cleanupTimer = time.AfterFunc(cleanupInterval, func() {
        c.cleanup()
        c.startCleanup() // 重新调度
    })
}

func (c *MaxMemoryCache) cleanup() {
    c.mu.Lock()
    defer c.mu.Unlock()
    
    now := time.Now()
    expiredKeys := make([]string, 0)
    
    // 查找过期的键
    for key, entry := range c.data {
        if entry.IsExpired(now) {
            expiredKeys = append(expiredKeys, key)
        }
    }
    
    // 删除过期的键
    for _, key := range expiredKeys {
        c.deleteInternal(key)
    }
}
```

## 性能优化

### 1. 读写锁优化

```go
// 读操作使用读锁，允许并发读取
func (c *MaxMemoryCache) Get(ctx context.Context, key string) (any, error) {
    c.mu.RLock()
    defer c.mu.RUnlock()
    // 读取逻辑
}

// 写操作使用写锁，确保数据一致性
func (c *MaxMemoryCache) Set(ctx context.Context, key string, val any, expiration time.Duration) error {
    c.mu.Lock()
    defer c.mu.Unlock()
    // 写入逻辑
}
```

### 2. 内存预分配

```go
func NewMaxMemoryCache(maxMemory int64) *MaxMemoryCache {
    // 预分配map容量，减少扩容开销
    initialCapacity := int(maxMemory / averageEntrySize)
    
    return &MaxMemoryCache{
        maxMemory: maxMemory,
        data:      make(map[string]*Entry, initialCapacity),
        lru:       tools.NewLRU[string](initialCapacity),
        // ...
    }
}
```

### 3. 批量操作优化

```go
func (c *MaxMemoryCache) SetBatch(ctx context.Context, items map[string]any, expiration time.Duration) error {
    c.mu.Lock()
    defer c.mu.Unlock()
    
    // 批量计算总大小
    totalSize := int64(0)
    for _, value := range items {
        totalSize += calculateSize(value)
    }
    
    // 一次性检查内存限制
    if err := c.evictIfNeeded(totalSize); err != nil {
        return err
    }
    
    // 批量设置
    for key, value := range items {
        c.setInternal(key, value, expiration)
    }
    
    return nil
}
```

## 使用示例

### 1. 基本使用

```go
// 创建缓存
cache := NewMaxMemoryCache(100 * 1024 * 1024) // 100MB

// 设置数据
err := cache.Set(ctx, "user:123", User{ID: 123, Name: "John"}, time.Hour)
if err != nil {
    log.Printf("设置失败: %v", err)
}

// 获取数据
user, err := cache.Get(ctx, "user:123")
if err != nil {
    if errors.Is(err, ErrKeyNotFound) {
        // 处理不存在的情况
    }
    return err
}

// 删除数据
err = cache.Delete(ctx, "user:123")
```

### 2. 批量操作

```go
// 批量设置
users := map[string]any{
    "user:1": User{ID: 1, Name: "Alice"},
    "user:2": User{ID: 2, Name: "Bob"},
    "user:3": User{ID: 3, Name: "Charlie"},
}

err := cache.SetBatch(ctx, users, time.Hour)
if err != nil {
    log.Printf("批量设置失败: %v", err)
}

// 模式匹配
userKeys, err := cache.Keys(ctx, "user:*")
if err != nil {
    log.Printf("获取键列表失败: %v", err)
}
```

### 3. 监控和统计

```go
// 获取统计信息
stats, err := cache.Stats(ctx)
if err != nil {
    log.Printf("获取统计失败: %v", err)
    return
}

hitRate := stats["hit_rate"].(float64)
memoryUsage := stats["memory_usage"].(int64)
maxMemory := stats["max_memory"].(int64)

log.Printf("命中率: %.2f%%, 内存使用: %d/%d bytes", 
    hitRate*100, memoryUsage, maxMemory)

// 内存使用率告警
if float64(memoryUsage)/float64(maxMemory) > 0.9 {
    log.Println("警告: 缓存内存使用率超过90%")
}
```

## 注意事项

### 1. 内存限制设置

```go
// ✅ 推荐：根据系统内存合理设置
totalMemory := getTotalSystemMemory()
cacheMemory := totalMemory * 60 / 100  // 使用60%系统内存
cache := NewMaxMemoryCache(cacheMemory)

// ❌ 避免：设置过大导致OOM
cache := NewMaxMemoryCache(math.MaxInt64)
```

### 2. 过期时间设置

```go
// ✅ 推荐：根据数据特性设置合理的过期时间
cache.Set(ctx, "user_session", session, 30*time.Minute)
cache.Set(ctx, "user_profile", profile, 24*time.Hour)

// ❌ 避免：过期时间过短导致频繁缓存未命中
cache.Set(ctx, "expensive_data", data, time.Millisecond)
```

### 3. 并发访问

```go
// ✅ 推荐：缓存本身是线程安全的
go func() {
    cache.Set(ctx, "key1", "value1", time.Hour)
}()

go func() {
    value, _ := cache.Get(ctx, "key1")
}()

// ❌ 避免：不要在外部加锁
var mu sync.Mutex
mu.Lock()
cache.Set(ctx, "key", "value", time.Hour) // 不必要的锁
mu.Unlock()
```

### 4. 资源清理

```go
// 程序退出时清理资源
defer func() {
    cache.Close() // 停止清理定时器
    cache.Clear(ctx) // 清空缓存
}()
```
