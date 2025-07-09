# build_in_map_cache.go - 内置Map缓存实现

## 文件概述

`build_in_map_cache.go` 实现了基于Go内置map的简单缓存，提供了基本的缓存功能包括设置、获取、删除操作，支持过期时间和自动清理。该实现适用于单机环境下的轻量级缓存需求，具有简单易用、性能良好的特点。

## 核心功能

### 1. BuildInMapCache 结构体

```go
type BuildInMapCache struct {
    data      map[string]*item    // 存储缓存项的映射
    mutex     sync.RWMutex        // 读写锁保证并发安全
    close     chan struct{}       // 关闭通道，用于停止后台清理
    onEvicted func(key string, val any) // 淘汰回调函数
}
```

**设计特点：**

- 基于Go内置map实现，简单高效
- 使用读写锁保证并发安全
- 支持自动过期清理机制
- 提供淘汰事件回调
- 优雅的关闭机制

### 2. item 缓存项结构

```go
type item struct {
    val      any        // 缓存值
    deadline time.Time  // 过期时间
}
```

**设计特点：**

- 存储任意类型的值
- 支持过期时间设置
- 零值表示永不过期

### 3. 配置选项

#### BuildInMapCacheOption 配置函数类型

```go
type BuildInMapCacheOption func(cache *BuildInMapCache)
```

#### BuildInMapCacheWithEvictedCallback 淘汰回调配置

```go
func BuildInMapCacheWithEvictedCallback(fn func(key string, val any)) BuildInMapCacheOption
```

## 主要方法

### 1. 构造函数

#### NewBuildInMapCache - 创建缓存实例

```go
func NewBuildInMapCache(interval time.Duration, opts ...BuildInMapCacheOption) *BuildInMapCache
```

**参数：**

- `interval`: 过期检查间隔时间，0表示不启动自动清理
- `opts`: 可选配置项

**特性：**

- 初始化容量为100的map
- 启动后台清理goroutine（当interval > 0时）
- 支持配置选项模式

**示例：**

```go
// 创建基本缓存，每分钟清理一次过期项
cache := NewBuildInMapCache(time.Minute)

// 创建带淘汰回调的缓存
cache := NewBuildInMapCache(time.Minute, 
    BuildInMapCacheWithEvictedCallback(func(key string, val any) {
        log.Printf("缓存项被淘汰: %s = %v", key, val)
    }))

// 创建不自动清理的缓存
cache := NewBuildInMapCache(0)
```

### 2. 基本操作

#### Set - 设置缓存值

```go
func (b *BuildInMapCache) Set(ctx context.Context, key string, val any, expiration time.Duration) error
```

**参数：**

- `ctx`: 上下文（当前实现中未使用）
- `key`: 缓存键
- `val`: 缓存值，任意类型
- `expiration`: 过期时间，0表示永不过期

**示例：**

```go
// 设置永不过期的缓存
err := cache.Set(ctx, "user:123", user, 0)

// 设置1小时后过期的缓存
err := cache.Set(ctx, "session:abc", session, time.Hour)
```

#### Get - 获取缓存值

```go
func (b *BuildInMapCache) Get(ctx context.Context, key string) (any, error)
```

**特性：**

- 自动检查过期时间
- 过期项会被自动删除
- 使用读锁提高并发性能

**示例：**

```go
value, err := cache.Get(ctx, "user:123")
if err != nil {
    if errors.Is(err, ErrCacheKeyNotFound) {
        fmt.Println("缓存未命中")
    } else {
        log.Printf("获取缓存失败: %v", err)
    }
    return
}

fmt.Printf("获取到缓存值: %v", value)
```

#### Delete - 删除缓存值

```go
func (b *BuildInMapCache) Delete(ctx context.Context, key string) error
```

#### LoadAndDelete - 获取并删除缓存值

```go
func (b *BuildInMapCache) LoadAndDelete(ctx context.Context, key string) (any, error)
```

**原子操作：**

- 在单个锁操作中完成获取和删除
- 避免竞态条件
- 适用于一次性消费的场景

### 3. 生命周期管理

#### Close - 关闭缓存

```go
func (b *BuildInMapCache) Close() error
```

**特性：**

- 停止后台清理goroutine
- 防止重复关闭
- 优雅的资源清理

#### OnEvicted - 设置淘汰回调

```go
func (b *BuildInMapCache) OnEvicted(fn func(key string, val any))
```

## 内部实现

### 1. 过期检查机制

#### deadlineBefore - 检查是否过期

```go
func (i *item) deadlineBefore(t time.Time) bool
```

**逻辑：**

- 零值时间表示永不过期
- 比较deadline和给定时间

### 2. 自动清理机制

**清理策略：**

- 定时器触发清理
- 每次最多检查10000个项目
- 避免长时间占用锁
- 支持优雅关闭

**实现细节：**

```go
// 后台清理goroutine
go func() {
    ticker := time.NewTicker(interval)
    defer ticker.Stop()
    for {
        select {
        case t := <-ticker.C:
            res.mutex.Lock()
            i := 0
            for key, val := range res.data {
                if i > 10000 { // 限制每次检查数量
                    break
                }
                if val.deadlineBefore(t) {
                    res.delete(key)
                }
                i++
            }
            res.mutex.Unlock()
        case <-res.close:
            return
        }
    }
}()
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
)

func main() {
    // 创建缓存，每30秒清理一次过期项
    c := cache.NewBuildInMapCache(30 * time.Second)
    defer c.Close()
    
    ctx := context.Background()
    
    // 设置缓存
    err := c.Set(ctx, "user:123", map[string]string{
        "name":  "John Doe",
        "email": "john@example.com",
    }, time.Minute)
    
    if err != nil {
        log.Printf("设置缓存失败: %v", err)
        return
    }
    
    fmt.Println("缓存设置成功")
    
    // 获取缓存
    value, err := c.Get(ctx, "user:123")
    if err != nil {
        log.Printf("获取缓存失败: %v", err)
        return
    }
    
    if user, ok := value.(map[string]string); ok {
        fmt.Printf("用户信息: %s (%s)\n", user["name"], user["email"])
    }
    
    // 删除缓存
    err = c.Delete(ctx, "user:123")
    if err != nil {
        log.Printf("删除缓存失败: %v", err)
    } else {
        fmt.Println("缓存删除成功")
    }
}
```

### 2. 带淘汰回调的使用

```go
func demonstrateEvictionCallback() {
    // 创建带淘汰回调的缓存
    c := cache.NewBuildInMapCache(time.Second, 
        cache.BuildInMapCacheWithEvictedCallback(func(key string, val any) {
            fmt.Printf("缓存项被淘汰: %s = %v\n", key, val)
        }))
    defer c.Close()
    
    ctx := context.Background()
    
    // 设置短期缓存
    items := map[string]string{
        "temp:1": "临时数据1",
        "temp:2": "临时数据2", 
        "temp:3": "临时数据3",
    }
    
    for key, value := range items {
        err := c.Set(ctx, key, value, 2*time.Second)
        if err != nil {
            log.Printf("设置缓存失败: %v", err)
        }
    }
    
    fmt.Println("设置了3个临时缓存项，等待过期...")
    
    // 等待过期和清理
    time.Sleep(5 * time.Second)
    
    fmt.Println("清理完成")
}
```

### 3. LoadAndDelete 原子操作

```go
func demonstrateLoadAndDelete() {
    c := cache.NewBuildInMapCache(time.Minute)
    defer c.Close()
    
    ctx := context.Background()
    
    // 设置一次性令牌
    tokens := []string{"token1", "token2", "token3"}
    
    for _, token := range tokens {
        err := c.Set(ctx, "token:"+token, map[string]interface{}{
            "user_id": rand.Intn(1000),
            "scope":   "read_profile",
        }, time.Hour)
        
        if err != nil {
            log.Printf("设置令牌失败: %v", err)
        }
    }
    
    fmt.Printf("设置了%d个令牌\n", len(tokens))
    
    // 消费令牌
    for _, token := range tokens {
        value, err := c.LoadAndDelete(ctx, "token:"+token)
        if err != nil {
            log.Printf("消费令牌失败: %v", err)
            continue
        }
        
        if tokenData, ok := value.(map[string]interface{}); ok {
            fmt.Printf("消费令牌 %s，用户ID: %v\n", token, tokenData["user_id"])
        }
    }
    
    // 验证令牌已被删除
    for _, token := range tokens {
        _, err := c.Get(ctx, "token:"+token)
        if err != nil {
            fmt.Printf("令牌 %s 已被消费\n", token)
        }
    }
}
```

### 4. 过期时间测试

```go
func demonstrateExpiration() {
    c := cache.NewBuildInMapCache(500 * time.Millisecond) // 快速清理
    defer c.Close()
    
    ctx := context.Background()
    
    // 设置不同过期时间的缓存
    testCases := []struct {
        key        string
        value      string
        expiration time.Duration
    }{
        {"short", "短期数据", time.Second},
        {"medium", "中期数据", 3 * time.Second},
        {"long", "长期数据", 10 * time.Second},
        {"permanent", "永久数据", 0}, // 永不过期
    }
    
    for _, tc := range testCases {
        err := c.Set(ctx, tc.key, tc.value, tc.expiration)
        if err != nil {
            log.Printf("设置缓存失败: %v", err)
        } else {
            if tc.expiration == 0 {
                fmt.Printf("设置永久缓存: %s\n", tc.key)
            } else {
                fmt.Printf("设置缓存: %s，过期时间: %v\n", tc.key, tc.expiration)
            }
        }
    }
    
    // 定期检查缓存状态
    for i := 0; i < 12; i++ {
        time.Sleep(time.Second)
        fmt.Printf("\n第%d秒检查:\n", i+1)
        
        for _, tc := range testCases {
            value, err := c.Get(ctx, tc.key)
            if err != nil {
                fmt.Printf("  %s: 已过期或不存在\n", tc.key)
            } else {
                fmt.Printf("  %s: %v\n", tc.key, value)
            }
        }
    }
}
```

### 5. 并发访问测试

```go
func demonstrateConcurrentAccess() {
    c := cache.NewBuildInMapCache(time.Minute)
    defer c.Close()
    
    ctx := context.Background()
    
    var wg sync.WaitGroup
    
    // 启动多个写入goroutine
    for i := 0; i < 5; i++ {
        wg.Add(1)
        go func(id int) {
            defer wg.Done()
            
            for j := 0; j < 100; j++ {
                key := fmt.Sprintf("writer_%d_key_%d", id, j)
                value := fmt.Sprintf("value_%d_%d", id, j)
                
                err := c.Set(ctx, key, value, time.Minute)
                if err != nil {
                    log.Printf("写入失败: %v", err)
                }
            }
        }(i)
    }
    
    // 启动多个读取goroutine
    for i := 0; i < 3; i++ {
        wg.Add(1)
        go func(id int) {
            defer wg.Done()
            
            for j := 0; j < 200; j++ {
                key := fmt.Sprintf("writer_%d_key_%d", rand.Intn(5), rand.Intn(100))
                
                value, err := c.Get(ctx, key)
                if err == nil {
                    fmt.Printf("读取器%d获取到: %s = %v\n", id, key, value)
                }
                
                time.Sleep(time.Millisecond)
            }
        }(i)
    }
    
    wg.Wait()
    fmt.Println("并发测试完成")
}
```

### 6. 内存使用监控

```go
func demonstrateMemoryMonitoring() {
    c := cache.NewBuildInMapCache(time.Minute)
    defer c.Close()
    
    ctx := context.Background()
    
    // 监控内存使用
    go func() {
        ticker := time.NewTicker(5 * time.Second)
        defer ticker.Stop()
        
        for {
            select {
            case <-ticker.C:
                var m runtime.MemStats
                runtime.GC()
                runtime.ReadMemStats(&m)
                
                fmt.Printf("内存使用: Alloc=%d KB, Sys=%d KB, NumGC=%d\n",
                    m.Alloc/1024, m.Sys/1024, m.NumGC)
            }
        }
    }()
    
    // 大量数据写入测试
    fmt.Println("开始大量数据写入...")
    
    for i := 0; i < 10000; i++ {
        key := fmt.Sprintf("data:%d", i)
        value := strings.Repeat(fmt.Sprintf("data_%d_", i), 10) // 较大的数据
        
        err := c.Set(ctx, key, value, time.Minute)
        if err != nil {
            log.Printf("设置缓存失败: %v", err)
        }
        
        if i%1000 == 0 {
            fmt.Printf("已写入 %d 条数据\n", i)
        }
    }
    
    fmt.Println("写入完成，等待监控...")
    time.Sleep(30 * time.Second)
}
```

## 性能特性

### 时间复杂度

- **Set操作**: O(1) - 直接map赋值
- **Get操作**: O(1) - 直接map查找
- **Delete操作**: O(1) - 直接map删除
- **清理操作**: O(n) - 遍历所有项目（有限制）

### 空间复杂度

- **存储空间**: O(n) - n为缓存项数量
- **额外开销**: O(1) - 固定的控制结构

### 并发性能

- **读操作**: 支持并发读取（读锁）
- **写操作**: 互斥写入（写锁）
- **清理操作**: 独占访问（写锁）

## 适用场景

### 1. 单机缓存

```go
// ✅ 适合：单机应用的本地缓存
cache := NewBuildInMapCache(time.Minute)
```

### 2. 轻量级缓存

```go
// ✅ 适合：简单的键值对缓存
// 不需要复杂的淘汰策略
cache := NewBuildInMapCache(time.Minute)
```

### 3. 临时数据存储

```go
// ✅ 适合：会话数据、临时令牌等
cache := NewBuildInMapCache(30 * time.Second)
```

## 注意事项

### 1. 内存管理

```go
// ✅ 推荐：设置合理的清理间隔
cache := NewBuildInMapCache(time.Minute) // 1分钟清理一次

// ❌ 避免：清理间隔过短导致性能问题
cache := NewBuildInMapCache(time.Millisecond) // 太频繁

// ❌ 避免：不设置清理导致内存泄漏
cache := NewBuildInMapCache(0) // 需要手动管理过期
```

### 2. 并发控制

```go
// ✅ 推荐：缓存本身是线程安全的
// 可以在多个goroutine中安全使用
go func() {
    cache.Set(ctx, "key1", "value1", time.Minute)
}()

go func() {
    value, _ := cache.Get(ctx, "key1")
}()

// ❌ 避免：不要在外部加锁
var mu sync.Mutex
mu.Lock()
cache.Set(ctx, "key", "value", time.Minute) // 不必要的锁
mu.Unlock()
```

### 3. 资源清理

```go
// ✅ 推荐：程序退出时关闭缓存
cache := NewBuildInMapCache(time.Minute)
defer cache.Close() // 确保资源清理

// ✅ 推荐：检查关闭错误
err := cache.Close()
if err != nil {
    log.Printf("关闭缓存失败: %v", err)
}
```

### 4. 错误处理

```go
// ✅ 推荐：正确处理缓存未命中
value, err := cache.Get(ctx, "key")
if err != nil {
    if errors.Is(err, cache.ErrCacheKeyNotFound) {
        // 缓存未命中，从数据源加载
        value = loadFromDataSource("key")
        cache.Set(ctx, "key", value, time.Hour)
    } else {
        // 其他错误
        log.Printf("缓存操作失败: %v", err)
    }
}

// ❌ 避免：忽略错误
value, _ := cache.Get(ctx, "key") // 可能丢失重要错误信息
```

### 5. 过期时间设置

```go
// ✅ 推荐：根据数据特性设置过期时间
cache.Set(ctx, "user_session", session, 30*time.Minute) // 会话30分钟
cache.Set(ctx, "config", config, time.Hour)             // 配置1小时
cache.Set(ctx, "static_data", data, 0)                  // 静态数据永不过期

// ❌ 避免：所有数据使用相同过期时间
cache.Set(ctx, "session", session, time.Hour)   // 会话时间过长
cache.Set(ctx, "temp_token", token, time.Hour)   // 临时令牌时间过长
```
