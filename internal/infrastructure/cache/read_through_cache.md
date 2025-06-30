# read_through_cache.go - 读透缓存实现

## 文件概述

`read_through_cache.go` 实现了读透缓存模式，当缓存未命中时自动从数据源加载数据并更新缓存。该实现使用SingleFlight防止缓存击穿，并提供了带限流功能的变体，为高并发场景下的数据访问提供了高效的缓存解决方案。

## 核心功能

### 1. ReadThroughCache 标准读透缓存

```go
type ReadThroughCache struct {
    domainCache.Repository                                    // 嵌入领域仓储接口
    LoadFunc   func(ctx context.Context, key string) (any, error) // 数据加载函数
    Expiration time.Duration                                      // 缓存过期时间
    logFunc    func(format string, args ...any)                  // 日志函数
    g          singleflight.Group                                 // 防止缓存击穿
}
```

**设计特点：**

- 嵌入Repository接口，支持所有标准缓存操作
- 自动数据加载和缓存更新
- 使用SingleFlight防止缓存击穿
- 支持自定义日志记录
- 优雅的错误处理机制

### 2. RateLimitReadThroughCache 限流读透缓存

```go
type RateLimitReadThroughCache struct {
    domainCache.Repository                                    // 嵌入领域仓储接口
    LoadFunc   func(ctx context.Context, key string) (any, error) // 数据加载函数
    Expiration time.Duration                                      // 缓存过期时间
    g          singleflight.Group                                 // 防止缓存击穿
}
```

**限流特性：**

- 通过上下文检查限流状态
- 被限流时不会触发数据加载
- 适用于需要保护后端服务的场景

## 主要方法

### 1. ReadThroughCache 核心方法

#### Get - 读透获取

```go
func (r *ReadThroughCache) Get(ctx context.Context, key string) (any, error)
```

**执行流程：**

1. 先从缓存获取数据
2. 如果缓存命中，直接返回
3. 如果缓存未命中，调用handleCacheMiss处理
4. 使用SingleFlight防止并发加载同一键

**示例：**

```go
// 创建读透缓存
cache := &ReadThroughCache{
    Repository: memoryCache,
    LoadFunc: func(ctx context.Context, key string) (any, error) {
        // 从数据库加载数据
        return database.GetUser(key)
    },
    Expiration: time.Hour,
}

// 设置日志函数
cache.SetLogFunc(log.Printf)

// 获取数据
user, err := cache.Get(ctx, "user:123")
if err != nil {
    log.Printf("获取用户失败: %v", err)
    return
}

fmt.Printf("用户信息: %v\n", user)
```

#### handleCacheMiss - 处理缓存未命中

```go
func (r *ReadThroughCache) handleCacheMiss(ctx context.Context, key string) (any, error)
```

**处理逻辑：**

1. 使用SingleFlight确保同一键只有一个加载操作
2. 调用LoadFunc从数据源加载数据
3. 更新缓存（即使失败也返回加载的数据）
4. 记录详细的日志信息

#### SetLogFunc - 设置日志函数

```go
func (r *ReadThroughCache) SetLogFunc(logFunc func(format string, args ...any))
```

### 2. RateLimitReadThroughCache 核心方法

#### Get - 带限流的读透获取

```go
func (r *RateLimitReadThroughCache) Get(ctx context.Context, key string) (any, error)
```

**限流检查：**

- 通过`ctx.Value("limited")`检查是否被限流
- 被限流时直接返回缓存结果，不触发数据加载
- 未被限流时执行正常的读透逻辑

**示例：**

```go
// 创建限流读透缓存
rateLimitCache := &RateLimitReadThroughCache{
    Repository: memoryCache,
    LoadFunc: func(ctx context.Context, key string) (any, error) {
        return database.GetUser(key)
    },
    Expiration: time.Hour,
}

// 正常访问
user, err := rateLimitCache.Get(ctx, "user:123")

// 限流访问
limitedCtx := context.WithValue(ctx, "limited", true)
user, err = rateLimitCache.Get(limitedCtx, "user:123") // 不会触发LoadFunc
```

## 使用示例

### 1. 基本读透缓存使用

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
    // 创建底层缓存
    memoryCache := cache.NewMaxMemoryCache(1024 * 1024) // 1MB
    
    // 创建读透缓存
    readThroughCache := &cache.ReadThroughCache{
        Repository: memoryCache,
        LoadFunc:   createUserLoader(),
        Expiration: time.Hour,
    }
    
    // 设置日志
    readThroughCache.SetLogFunc(log.Printf)
    
    ctx := context.Background()
    
    // 第一次获取（会触发加载）
    fmt.Println("第一次获取用户...")
    user, err := readThroughCache.Get(ctx, "user:123")
    if err != nil {
        log.Printf("获取用户失败: %v", err)
        return
    }
    fmt.Printf("用户信息: %v\n", user)
    
    // 第二次获取（从缓存获取）
    fmt.Println("第二次获取用户...")
    user, err = readThroughCache.Get(ctx, "user:123")
    if err != nil {
        log.Printf("获取用户失败: %v", err)
        return
    }
    fmt.Printf("用户信息: %v\n", user)
}

func createUserLoader() func(context.Context, string) (any, error) {
    return func(ctx context.Context, key string) (any, error) {
        fmt.Printf("从数据库加载: %s\n", key)
        
        // 模拟数据库查询延迟
        time.Sleep(100 * time.Millisecond)
        
        // 模拟数据
        if key == "user:123" {
            return map[string]interface{}{
                "id":   "123",
                "name": "John Doe",
                "age":  30,
            }, nil
        }
        
        return nil, cache.ErrKeyNotFound
    }
}
```

### 2. 限流读透缓存使用

```go
func demonstrateRateLimitCache() {
    memoryCache := cache.NewMaxMemoryCache(1024 * 1024)
    
    rateLimitCache := &cache.RateLimitReadThroughCache{
        Repository: memoryCache,
        LoadFunc:   createUserLoader(),
        Expiration: time.Hour,
    }
    
    ctx := context.Background()
    
    // 正常访问
    fmt.Println("正常访问...")
    user, err := rateLimitCache.Get(ctx, "user:456")
    if err != nil {
        log.Printf("获取用户失败: %v", err)
    } else {
        fmt.Printf("正常获取用户: %v\n", user)
    }
    
    // 模拟限流场景
    fmt.Println("限流访问...")
    limitedCtx := context.WithValue(ctx, "limited", true)
    
    // 尝试获取新用户（被限流，不会触发LoadFunc）
    user, err = rateLimitCache.Get(limitedCtx, "user:789")
    if err != nil {
        if errors.Is(err, cache.ErrKeyNotFound) {
            fmt.Println("限流状态下，新用户未找到（未触发加载）")
        } else {
            log.Printf("获取用户失败: %v", err)
        }
    }
    
    // 获取已缓存的用户（限流状态下仍可获取）
    user, err = rateLimitCache.Get(limitedCtx, "user:456")
    if err == nil {
        fmt.Printf("限流状态下获取已缓存用户: %v\n", user)
    }
}
```

### 3. 并发访问测试

```go
func demonstrateConcurrentAccess() {
    memoryCache := cache.NewMaxMemoryCache(1024 * 1024)
    
    readThroughCache := &cache.ReadThroughCache{
        Repository: memoryCache,
        LoadFunc:   createSlowLoader(), // 慢加载函数
        Expiration: time.Hour,
    }
    
    readThroughCache.SetLogFunc(func(format string, args ...any) {
        log.Printf("[ReadThrough] "+format, args...)
    })
    
    ctx := context.Background()
    key := "slow:data"
    
    // 启动多个goroutine并发访问同一键
    var wg sync.WaitGroup
    results := make(chan string, 5)
    
    fmt.Println("启动5个并发请求...")
    for i := 0; i < 5; i++ {
        wg.Add(1)
        go func(id int) {
            defer wg.Done()
            
            start := time.Now()
            value, err := readThroughCache.Get(ctx, key)
            duration := time.Since(start)
            
            if err != nil {
                results <- fmt.Sprintf("Goroutine %d: 错误 - %v (耗时: %v)", id, err, duration)
            } else {
                results <- fmt.Sprintf("Goroutine %d: 成功 - %v (耗时: %v)", id, value, duration)
            }
        }(i)
    }
    
    wg.Wait()
    close(results)
    
    // 输出结果
    for result := range results {
        fmt.Println(result)
    }
}

func createSlowLoader() func(context.Context, string) (any, error) {
    return func(ctx context.Context, key string) (any, error) {
        fmt.Printf("开始慢加载: %s\n", key)
        
        // 模拟慢查询
        time.Sleep(2 * time.Second)
        
        fmt.Printf("完成慢加载: %s\n", key)
        return fmt.Sprintf("data for %s", key), nil
    }
}
```

### 4. 错误处理和恢复

```go
func demonstrateErrorHandling() {
    memoryCache := cache.NewMaxMemoryCache(1024 * 1024)
    
    readThroughCache := &cache.ReadThroughCache{
        Repository: memoryCache,
        LoadFunc:   createFaultyLoader(),
        Expiration: time.Hour,
    }
    
    readThroughCache.SetLogFunc(log.Printf)
    
    ctx := context.Background()
    
    testCases := []string{
        "success:key",
        "error:key",
        "timeout:key",
        "cache_error:key",
    }
    
    for _, key := range testCases {
        fmt.Printf("测试键: %s\n", key)
        
        value, err := readThroughCache.Get(ctx, key)
        if err != nil {
            switch {
            case errors.Is(err, cache.ErrKeyNotFound):
                fmt.Printf("  结果: 键不存在\n")
            case errors.Is(err, cache.ErrFailedToRefreshCache):
                fmt.Printf("  结果: 缓存刷新失败，但数据已加载: %v\n", value)
            default:
                fmt.Printf("  结果: 其他错误 - %v\n", err)
            }
        } else {
            fmt.Printf("  结果: 成功获取 - %v\n", value)
        }
        fmt.Println()
    }
}

func createFaultyLoader() func(context.Context, string) (any, error) {
    return func(ctx context.Context, key string) (any, error) {
        switch {
        case strings.HasPrefix(key, "success:"):
            return fmt.Sprintf("data for %s", key), nil
        case strings.HasPrefix(key, "error:"):
            return nil, errors.New("模拟加载错误")
        case strings.HasPrefix(key, "timeout:"):
            time.Sleep(5 * time.Second)
            return nil, context.DeadlineExceeded
        case strings.HasPrefix(key, "cache_error:"):
            // 这种情况下，数据加载成功但缓存设置可能失败
            return "loaded data", nil
        default:
            return nil, cache.ErrKeyNotFound
        }
    }
}
```

### 5. 性能监控

```go
func demonstratePerformanceMonitoring() {
    memoryCache := cache.NewMaxMemoryCache(1024 * 1024)
    
    // 创建带性能监控的读透缓存
    readThroughCache := &cache.ReadThroughCache{
        Repository: memoryCache,
        LoadFunc:   createMonitoredLoader(),
        Expiration: time.Hour,
    }
    
    // 设置详细日志
    readThroughCache.SetLogFunc(func(format string, args ...any) {
        log.Printf("[Performance] "+format, args...)
    })
    
    ctx := context.Background()
    
    // 性能测试
    fmt.Println("开始性能测试...")
    
    start := time.Now()
    cacheHits := 0
    cacheMisses := 0
    
    // 测试100次访问
    for i := 0; i < 100; i++ {
        key := fmt.Sprintf("perf:key:%d", i%10) // 10个不同的键，重复访问
        
        accessStart := time.Now()
        _, err := readThroughCache.Get(ctx, key)
        accessDuration := time.Since(accessStart)
        
        if err == nil {
            if accessDuration < 10*time.Millisecond {
                cacheHits++ // 快速响应，可能是缓存命中
            } else {
                cacheMisses++ // 慢响应，可能是缓存未命中
            }
        }
    }
    
    totalDuration := time.Since(start)
    
    fmt.Printf("性能测试结果:\n")
    fmt.Printf("  总耗时: %v\n", totalDuration)
    fmt.Printf("  平均延迟: %v\n", totalDuration/100)
    fmt.Printf("  估算缓存命中: %d\n", cacheHits)
    fmt.Printf("  估算缓存未命中: %d\n", cacheMisses)
    fmt.Printf("  估算命中率: %.2f%%\n", float64(cacheHits)/float64(cacheHits+cacheMisses)*100)
}

func createMonitoredLoader() func(context.Context, string) (any, error) {
    loadCount := 0
    return func(ctx context.Context, key string) (any, error) {
        loadCount++
        fmt.Printf("数据加载 #%d: %s\n", loadCount, key)
        
        // 模拟数据库查询
        time.Sleep(50 * time.Millisecond)
        
        return fmt.Sprintf("data for %s (load #%d)", key, loadCount), nil
    }
}
```

## 性能特性

### 时间复杂度

- **缓存命中**: O(1) - 直接从缓存获取
- **缓存未命中**: O(1) + O(load) - 缓存查找 + 数据加载时间
- **并发访问**: O(1) - SingleFlight确保同一键只加载一次

### 空间复杂度

- **缓存存储**: O(n) - n为缓存条目数量
- **SingleFlight**: O(k) - k为正在加载的键数量

### 性能优势

1. **防止缓存击穿**: SingleFlight确保同一键只有一个加载操作
2. **自动缓存管理**: 无需手动管理缓存更新
3. **优雅降级**: 即使缓存更新失败也返回加载的数据

## 注意事项

### 1. LoadFunc设计

```go
// ✅ 推荐：LoadFunc应该是幂等的
loadFunc := func(ctx context.Context, key string) (any, error) {
    // 幂等的数据库查询
    return database.GetByID(key)
}

// ❌ 避免：LoadFunc有副作用
loadFunc := func(ctx context.Context, key string) (any, error) {
    // 有副作用的操作，可能导致问题
    database.IncrementCounter(key)
    return database.GetByID(key)
}
```

### 2. 过期时间设置

```go
// ✅ 推荐：根据数据特性设置合理的过期时间
cache := &ReadThroughCache{
    Repository: memoryCache,
    LoadFunc:   loadFunc,
    Expiration: time.Hour, // 根据数据更新频率设置
}

// ❌ 避免：过期时间过短导致频繁加载
cache.Expiration = time.Second // 太短
```

### 3. 错误处理

```go
// ✅ 推荐：区分不同类型的错误
value, err := cache.Get(ctx, key)
if err != nil {
    switch {
    case errors.Is(err, cache.ErrKeyNotFound):
        // 数据不存在
    case errors.Is(err, cache.ErrFailedToRefreshCache):
        // 数据加载成功但缓存更新失败
        // 可以使用返回的value
    default:
        // 其他错误
    }
}
```

### 4. 限流使用

```go
// ✅ 推荐：在适当的中间件中设置限流标记
func rateLimitMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if isRateLimited(r) {
            ctx := context.WithValue(r.Context(), "limited", true)
            r = r.WithContext(ctx)
        }
        next.ServeHTTP(w, r)
    })
}
```

### 5. 日志记录

```go
// ✅ 推荐：设置结构化日志
cache.SetLogFunc(func(format string, args ...any) {
    log.Printf("[ReadThroughCache] "+format, args...)
})

// 监控关键指标
// - 缓存命中率
// - 数据加载延迟
// - 缓存更新失败率
```
