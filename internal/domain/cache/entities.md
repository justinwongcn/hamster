# entities.go - 缓存实体定义

## 文件概述

`entities.go` 定义了缓存领域的核心实体，包括缓存条目（Entry）和缓存实例（CacheInstance）。这些实体封装了缓存的业务逻辑和状态管理，遵循DDD设计原则，为缓存系统提供了丰富的领域模型。

## 核心功能

### 1. Entry 缓存条目实体

#### 结构定义

```go
type Entry struct {
    key         CacheKey    // 缓存键
    value       CacheValue  // 缓存值
    expiration  Expiration  // 过期时间
    createdAt   time.Time   // 创建时间
    accessedAt  time.Time   // 最后访问时间
    accessCount int64       // 访问次数
}
```

**设计特点：**

- 封装缓存条目的完整生命周期信息
- 支持访问统计和时间跟踪
- 提供过期检查和脏数据管理
- 实现不可变的键值对象引用

#### 构造函数

```go
func NewEntry(key CacheKey, value CacheValue, expiration Expiration) *Entry
```

**初始化逻辑：**

1. 设置键、值和过期时间
2. 记录创建时间和初始访问时间
3. 初始化访问计数为0

**示例：**

```go
// 创建缓存键和值
key, _ := NewCacheKey("user:12345")
value, _ := NewCacheValue("John Doe", 100)
expiration, _ := NewExpiration(time.Hour)

// 创建缓存条目
entry := NewEntry(key, value, expiration)

fmt.Printf("条目创建时间: %v\n", entry.CreatedAt())
fmt.Printf("是否过期: %v\n", entry.IsExpired(time.Now()))
```

### 2. Entry 核心方法

#### 访问器方法

- `Key()`: 获取缓存键
- `Value()`: 获取缓存值
- `Expiration()`: 获取过期时间
- `CreatedAt()`: 获取创建时间
- `AccessedAt()`: 获取最后访问时间
- `AccessCount()`: 获取访问次数

#### 状态检查方法

```go
func (e *Entry) IsExpired(now time.Time) bool
func (e *Entry) IsDirty() bool
```

**示例：**

```go
now := time.Now()

// 检查过期状态
if entry.IsExpired(now) {
    fmt.Println("条目已过期")
}

// 检查脏数据状态
if entry.IsDirty() {
    fmt.Println("条目包含未同步的脏数据")
}
```

#### 状态修改方法

```go
func (e *Entry) UpdateValue(newValue CacheValue)
func (e *Entry) MarkAccessed()
func (e *Entry) MarkClean()
func (e *Entry) MarkDirty()
```

**示例：**

```go
// 更新值
newValue, _ := NewCacheValue("Jane Doe", 120)
entry.UpdateValue(newValue)

// 标记访问
entry.MarkAccessed()
fmt.Printf("访问次数: %d\n", entry.AccessCount())

// 标记为脏数据
entry.MarkDirty()
```

#### 时间计算方法

```go
func (e *Entry) Age(now time.Time) time.Duration
func (e *Entry) IdleTime(now time.Time) time.Duration
```

**示例：**

```go
now := time.Now()

age := entry.Age(now)
idleTime := entry.IdleTime(now)

fmt.Printf("条目年龄: %v\n", age)
fmt.Printf("空闲时间: %v\n", idleTime)

// 可用于LRU淘汰策略
if idleTime > time.Hour {
    fmt.Println("条目长时间未访问，可考虑淘汰")
}
```

#### 克隆方法

```go
func (e *Entry) Clone() *Entry
```

创建条目的深拷贝，用于快照或备份操作。

### 3. CacheInstance 缓存实例实体

#### 结构定义

```go
type CacheInstance struct {
    name          string           // 缓存实例名称
    entries       map[string]*Entry // 缓存条目映射
    stats         CacheStats       // 统计信息
    maxSize       int64            // 最大条目数
    maxMemory     int64            // 最大内存使用量
    currentSize   int64            // 当前条目数
    currentMemory int64            // 当前内存使用量
}
```

**设计特点：**

- 作为聚合根管理多个缓存条目
- 维护容量限制和统计信息
- 提供条目的增删改查操作
- 支持过期清理和脏数据管理

#### 构造函数

```go
func NewCacheInstance(name string, maxSize, maxMemory int64) *CacheInstance
```

**参数说明：**

- `name`: 缓存实例名称
- `maxSize`: 最大条目数，0表示无限制
- `maxMemory`: 最大内存使用量，0表示无限制

**示例：**

```go
// 创建缓存实例
cache := NewCacheInstance("user_cache", 1000, 10*1024*1024) // 1000条目，10MB

fmt.Printf("缓存名称: %s\n", cache.Name())
fmt.Printf("最大条目数: %d\n", cache.MaxSize())
fmt.Printf("最大内存: %d bytes\n", cache.MaxMemory())
```

### 4. CacheInstance 核心方法

#### 状态查询方法

```go
func (c *CacheInstance) Size() int64
func (c *CacheInstance) Memory() int64
func (c *CacheInstance) IsFull() bool
func (c *CacheInstance) Stats() CacheStats
```

**示例：**

```go
fmt.Printf("当前条目数: %d/%d\n", cache.Size(), cache.MaxSize())
fmt.Printf("当前内存: %d/%d bytes\n", cache.Memory(), cache.MaxMemory())

if cache.IsFull() {
    fmt.Println("缓存已满，需要执行淘汰策略")
}

stats := cache.Stats()
fmt.Printf("命中率: %.2f%%\n", stats.HitRate()*100)
```

#### 条目操作方法

```go
func (c *CacheInstance) GetEntry(key CacheKey) (*Entry, bool)
func (c *CacheInstance) SetEntry(entry *Entry)
func (c *CacheInstance) RemoveEntry(key CacheKey) (*Entry, bool)
```

**示例：**

```go
// 设置条目
key, _ := NewCacheKey("user:123")
value, _ := NewCacheValue("Alice", 50)
expiration, _ := NewExpiration(time.Hour)
entry := NewEntry(key, value, expiration)

cache.SetEntry(entry)

// 获取条目
if retrievedEntry, exists := cache.GetEntry(key); exists {
    fmt.Printf("获取到用户: %v\n", retrievedEntry.Value().Data())
} else {
    fmt.Println("用户不存在")
}

// 移除条目
if removedEntry, exists := cache.RemoveEntry(key); exists {
    fmt.Printf("移除了用户: %v\n", removedEntry.Value().Data())
}
```

#### 批量操作方法

```go
func (c *CacheInstance) GetDirtyEntries() []*Entry
func (c *CacheInstance) GetExpiredEntries(now time.Time) []*Entry
func (c *CacheInstance) CleanExpiredEntries(now time.Time) int
```

**示例：**

```go
now := time.Now()

// 获取脏数据条目
dirtyEntries := cache.GetDirtyEntries()
fmt.Printf("脏数据条目数: %d\n", len(dirtyEntries))

// 清理过期条目
cleanedCount := cache.CleanExpiredEntries(now)
fmt.Printf("清理了 %d 个过期条目\n", cleanedCount)

// 获取过期条目（不删除）
expiredEntries := cache.GetExpiredEntries(now)
for _, entry := range expiredEntries {
    fmt.Printf("过期条目: %s\n", entry.Key().String())
}
```

## 使用示例

### 1. 基本缓存操作

```go
package main

import (
    "fmt"
    "time"
    
    "github.com/justinwongcn/hamster/internal/domain/cache"
)

func main() {
    // 创建缓存实例
    cacheInstance := cache.NewCacheInstance("user_cache", 100, 1024*1024)
    
    // 创建缓存条目
    key, _ := cache.NewCacheKey("user:12345")
    value, _ := cache.NewCacheValue("John Doe", 100)
    expiration, _ := cache.NewExpiration(time.Hour)
    entry := cache.NewEntry(key, value, expiration)
    
    // 设置条目
    cacheInstance.SetEntry(entry)
    
    // 获取条目
    if retrievedEntry, exists := cacheInstance.GetEntry(key); exists {
        fmt.Printf("用户: %v\n", retrievedEntry.Value().Data())
        fmt.Printf("访问次数: %d\n", retrievedEntry.AccessCount())
    }
    
    // 检查统计信息
    stats := cacheInstance.Stats()
    fmt.Printf("命中次数: %d\n", stats.Hits())
    fmt.Printf("未命中次数: %d\n", stats.Misses())
}
```

### 2. 过期管理

```go
func demonstrateExpiration() {
    cache := cache.NewCacheInstance("temp_cache", 0, 0)
    
    // 创建短期条目
    key, _ := cache.NewCacheKey("temp:data")
    value, _ := cache.NewCacheValue("temporary data", 50)
    expiration, _ := cache.NewExpiration(time.Second) // 1秒过期
    entry := cache.NewEntry(key, value, expiration)
    
    cache.SetEntry(entry)
    
    // 立即检查
    fmt.Printf("创建后是否过期: %v\n", entry.IsExpired(time.Now()))
    
    // 等待过期
    time.Sleep(2 * time.Second)
    
    // 再次检查
    fmt.Printf("2秒后是否过期: %v\n", entry.IsExpired(time.Now()))
    
    // 清理过期条目
    cleanedCount := cache.CleanExpiredEntries(time.Now())
    fmt.Printf("清理了 %d 个过期条目\n", cleanedCount)
}
```

### 3. 脏数据管理

```go
func demonstrateDirtyData() {
    cache := cache.NewCacheInstance("dirty_cache", 0, 0)
    
    // 创建条目
    key, _ := cache.NewCacheKey("user:123")
    value, _ := cache.NewCacheValue("Original Data", 100)
    expiration, _ := cache.NewExpiration(time.Hour)
    entry := cache.NewEntry(key, value, expiration)
    
    cache.SetEntry(entry)
    
    // 标记为脏数据
    entry.MarkDirty()
    fmt.Printf("是否为脏数据: %v\n", entry.IsDirty())
    
    // 获取所有脏数据条目
    dirtyEntries := cache.GetDirtyEntries()
    fmt.Printf("脏数据条目数: %d\n", len(dirtyEntries))
    
    // 模拟同步到持久化存储
    for _, dirtyEntry := range dirtyEntries {
        fmt.Printf("同步脏数据: %s\n", dirtyEntry.Key().String())
        dirtyEntry.MarkClean() // 标记为已同步
    }
}
```

### 4. 容量管理

```go
func demonstrateCapacityManagement() {
    // 创建有限容量的缓存
    cache := cache.NewCacheInstance("limited_cache", 3, 0) // 最多3个条目
    
    // 添加条目直到满载
    for i := 1; i <= 5; i++ {
        key, _ := cache.NewCacheKey(fmt.Sprintf("item:%d", i))
        value, _ := cache.NewCacheValue(fmt.Sprintf("data_%d", i), 50)
        expiration, _ := cache.NewExpiration(time.Hour)
        entry := cache.NewEntry(key, value, expiration)
        
        if cache.IsFull() {
            fmt.Printf("缓存已满，无法添加 item:%d\n", i)
            break
        }
        
        cache.SetEntry(entry)
        fmt.Printf("添加了 item:%d，当前大小: %d\n", i, cache.Size())
    }
    
    fmt.Printf("最终缓存大小: %d/%d\n", cache.Size(), cache.MaxSize())
}
```

### 5. 统计信息监控

```go
func demonstrateStatistics() {
    cache := cache.NewCacheInstance("stats_cache", 0, 0)
    
    // 添加一些条目
    for i := 1; i <= 3; i++ {
        key, _ := cache.NewCacheKey(fmt.Sprintf("key:%d", i))
        value, _ := cache.NewCacheValue(fmt.Sprintf("value_%d", i), 30)
        expiration, _ := cache.NewExpiration(time.Hour)
        entry := cache.NewEntry(key, value, expiration)
        cache.SetEntry(entry)
    }
    
    // 模拟一些访问
    key1, _ := cache.NewCacheKey("key:1")
    key4, _ := cache.NewCacheKey("key:4") // 不存在的键
    
    cache.GetEntry(key1) // 命中
    cache.GetEntry(key1) // 再次命中
    cache.GetEntry(key4) // 未命中
    
    // 查看统计信息
    stats := cache.Stats()
    fmt.Printf("统计信息:\n")
    fmt.Printf("  命中次数: %d\n", stats.Hits())
    fmt.Printf("  未命中次数: %d\n", stats.Misses())
    fmt.Printf("  命中率: %.2f%%\n", stats.HitRate()*100)
    fmt.Printf("  设置次数: %d\n", stats.Sets())
    fmt.Printf("  删除次数: %d\n", stats.Deletes())
}
```

## 设计原则

### 1. 实体身份

- Entry使用CacheKey作为唯一标识
- CacheInstance使用name作为唯一标识
- 支持实体的生命周期管理

### 2. 聚合设计

- CacheInstance作为聚合根
- Entry作为聚合内的实体
- 通过聚合根控制一致性边界

### 3. 不变性保护

- 关键属性通过值对象封装
- 提供受控的状态修改方法
- 防止无效状态的产生

### 4. 业务逻辑封装

- 将缓存相关的业务规则封装在实体内
- 提供有意义的业务方法
- 隐藏技术实现细节

## 注意事项

### 1. 内存管理

```go
// ✅ 推荐：设置合理的容量限制
cache := cache.NewCacheInstance("cache", 1000, 10*1024*1024)

// ❌ 避免：无限制可能导致内存泄漏
cache := cache.NewCacheInstance("cache", 0, 0) // 在高负载下要小心
```

### 2. 过期处理

```go
// ✅ 推荐：定期清理过期条目
go func() {
    ticker := time.NewTicker(time.Minute)
    defer ticker.Stop()
    
    for {
        select {
        case <-ticker.C:
            cache.CleanExpiredEntries(time.Now())
        }
    }
}()
```

### 3. 统计信息

```go
// ✅ 推荐：监控缓存性能
stats := cache.Stats()
if stats.HitRate() < 0.8 {
    log.Println("警告: 缓存命中率过低")
}
```

### 4. 并发安全

```go
// ⚠️ 注意：实体本身不是线程安全的
// 需要在上层进行同步控制
var mu sync.RWMutex

func safeGetEntry(cache *CacheInstance, key CacheKey) (*Entry, bool) {
    mu.RLock()
    defer mu.RUnlock()
    return cache.GetEntry(key)
}
```
