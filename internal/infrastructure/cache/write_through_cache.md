# write_through_cache.go - 写透缓存实现

## 文件概述

`write_through_cache.go` 实现了写透缓存模式，当写入缓存时同时写入到持久化存储，确保缓存和存储的数据一致性。该实现提供了标准写透缓存和带限流功能的变体，为需要强一致性的应用场景提供了可靠的缓存解决方案。

## 核心功能

### 1. WriteThroughCache 标准写透缓存

```go
type WriteThroughCache struct {
    domainCache.Repository                                // 嵌入领域仓储接口
    StoreFunc func(ctx context.Context, key string, val any) error // 持久化存储函数
}
```

**设计特点：**

- 嵌入Repository接口，支持所有标准缓存操作
- 写入时同步更新缓存和持久化存储
- 确保数据一致性，持久化失败时不更新缓存
- 简单可靠的强一致性保证

### 2. RateLimitWriteThroughCache 限流写透缓存

```go
type RateLimitWriteThroughCache struct {
    domainCache.Repository                                // 嵌入领域仓储接口
    StoreFunc func(ctx context.Context, key string, val any) error // 持久化存储函数
}
```

**限流特性：**

- 通过上下文检查限流状态
- 被限流时跳过持久化存储，只更新缓存
- 适用于需要保护后端存储的场景
- 在高负载时提供降级策略

## 主要方法

### 1. WriteThroughCache 核心方法

#### Set - 写透设置

```go
func (w *WriteThroughCache) Set(ctx context.Context, key string, val any, expiration time.Duration) error
```

**执行流程：**

1. 先调用StoreFunc写入持久化存储
2. 如果持久化成功，再写入缓存
3. 如果持久化失败，不写入缓存，直接返回错误
4. 确保缓存和存储的强一致性

**示例：**

```go
// 创建写透缓存
cache := &WriteThroughCache{
    Repository: memoryCache,
    StoreFunc: func(ctx context.Context, key string, val any) error {
        // 写入数据库
        return database.Save(key, val)
    },
}

// 写入数据
user := User{ID: "123", Name: "John"}
err := cache.Set(ctx, "user:123", user, time.Hour)
if err != nil {
    log.Printf("写入失败: %v", err)
    // 此时缓存和数据库都没有更新
    return
}

// 成功写入，缓存和数据库都已更新
fmt.Println("数据写入成功")
```

### 2. RateLimitWriteThroughCache 核心方法

#### Set - 带限流的写透设置

```go
func (r *RateLimitWriteThroughCache) Set(ctx context.Context, key string, val any, expiration time.Duration) error
```

**限流逻辑：**

1. 检查上下文中的限流标记
2. 未被限流时：先写持久化存储，再写缓存
3. 被限流时：跳过持久化存储，只写缓存
4. 无论是否限流都会更新缓存

**示例：**

```go
// 创建限流写透缓存
rateLimitCache := &RateLimitWriteThroughCache{
    Repository: memoryCache,
    StoreFunc: func(ctx context.Context, key string, val any) error {
        return database.Save(key, val)
    },
}

// 正常写入
user := User{ID: "123", Name: "John"}
err := rateLimitCache.Set(ctx, "user:123", user, time.Hour)
// 会同时更新缓存和数据库

// 限流写入
limitedCtx := context.WithValue(ctx, "limited", true)
err = rateLimitCache.Set(limitedCtx, "user:456", user, time.Hour)
// 只更新缓存，不写数据库
```

## 使用示例

### 1. 基本写透缓存使用

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
    
    // 创建写透缓存
    writeThroughCache := &cache.WriteThroughCache{
        Repository: memoryCache,
        StoreFunc:  createDatabaseStorer(),
    }
    
    ctx := context.Background()
    
    // 写入数据
    user := map[string]interface{}{
        "id":   "123",
        "name": "John Doe",
        "age":  30,
    }
    
    fmt.Println("写入用户数据...")
    err := writeThroughCache.Set(ctx, "user:123", user, time.Hour)
    if err != nil {
        log.Printf("写入失败: %v", err)
        return
    }
    
    fmt.Println("写入成功，数据已同步到缓存和数据库")
    
    // 验证数据
    cachedUser, err := writeThroughCache.Get(ctx, "user:123")
    if err != nil {
        log.Printf("获取缓存数据失败: %v", err)
        return
    }
    
    fmt.Printf("缓存中的用户数据: %v\n", cachedUser)
}

func createDatabaseStorer() func(context.Context, string, any) error {
    return func(ctx context.Context, key string, val any) error {
        fmt.Printf("写入数据库: %s = %v\n", key, val)
        
        // 模拟数据库写入延迟
        time.Sleep(50 * time.Millisecond)
        
        // 模拟偶发的数据库错误
        if key == "error:key" {
            return errors.New("数据库写入失败")
        }
        
        return nil
    }
}
```

### 2. 限流写透缓存使用

```go
func demonstrateRateLimitWriteThrough() {
    memoryCache := cache.NewMaxMemoryCache(1024 * 1024)
    
    rateLimitCache := &cache.RateLimitWriteThroughCache{
        Repository: memoryCache,
        StoreFunc:  createDatabaseStorer(),
    }
    
    ctx := context.Background()
    
    user1 := map[string]interface{}{"id": "456", "name": "Alice"}
    user2 := map[string]interface{}{"id": "789", "name": "Bob"}
    
    // 正常写入
    fmt.Println("正常写入用户1...")
    err := rateLimitCache.Set(ctx, "user:456", user1, time.Hour)
    if err != nil {
        log.Printf("写入失败: %v", err)
    } else {
        fmt.Println("用户1写入成功（缓存+数据库）")
    }
    
    // 限流写入
    fmt.Println("限流写入用户2...")
    limitedCtx := context.WithValue(ctx, "limited", true)
    err = rateLimitCache.Set(limitedCtx, "user:789", user2, time.Hour)
    if err != nil {
        log.Printf("写入失败: %v", err)
    } else {
        fmt.Println("用户2写入成功（仅缓存）")
    }
    
    // 验证缓存中的数据
    fmt.Println("验证缓存数据...")
    for _, key := range []string{"user:456", "user:789"} {
        if user, err := rateLimitCache.Get(ctx, key); err == nil {
            fmt.Printf("缓存中的%s: %v\n", key, user)
        }
    }
}
```

### 3. 错误处理和事务性

```go
func demonstrateTransactionalBehavior() {
    memoryCache := cache.NewMaxMemoryCache(1024 * 1024)
    
    writeThroughCache := &cache.WriteThroughCache{
        Repository: memoryCache,
        StoreFunc:  createFaultyStorer(),
    }
    
    ctx := context.Background()
    
    testCases := []struct {
        key   string
        value interface{}
        desc  string
    }{
        {"success:key", "success value", "成功写入"},
        {"error:key", "error value", "数据库错误"},
        {"timeout:key", "timeout value", "数据库超时"},
    }
    
    for _, tc := range testCases {
        fmt.Printf("测试: %s\n", tc.desc)
        
        // 记录写入前的缓存状态
        beforeWrite, _ := writeThroughCache.Get(ctx, tc.key)
        fmt.Printf("  写入前缓存: %v\n", beforeWrite)
        
        // 尝试写入
        err := writeThroughCache.Set(ctx, tc.key, tc.value, time.Hour)
        
        // 记录写入后的缓存状态
        afterWrite, _ := writeThroughCache.Get(ctx, tc.key)
        fmt.Printf("  写入后缓存: %v\n", afterWrite)
        
        if err != nil {
            fmt.Printf("  结果: 写入失败 - %v\n", err)
            fmt.Printf("  一致性: 缓存未更新，保持一致性\n")
        } else {
            fmt.Printf("  结果: 写入成功\n")
            fmt.Printf("  一致性: 缓存和数据库都已更新\n")
        }
        fmt.Println()
    }
}

func createFaultyStorer() func(context.Context, string, any) error {
    return func(ctx context.Context, key string, val any) error {
        fmt.Printf("  尝试写入数据库: %s\n", key)
        
        switch {
        case strings.HasPrefix(key, "success:"):
            time.Sleep(10 * time.Millisecond)
            fmt.Printf("  数据库写入成功: %s\n", key)
            return nil
        case strings.HasPrefix(key, "error:"):
            fmt.Printf("  数据库写入失败: %s\n", key)
            return errors.New("数据库连接错误")
        case strings.HasPrefix(key, "timeout:"):
            fmt.Printf("  数据库写入超时: %s\n", key)
            return context.DeadlineExceeded
        default:
            return nil
        }
    }
}
```

### 4. 批量操作和性能测试

```go
func demonstrateBatchOperations() {
    memoryCache := cache.NewMaxMemoryCache(1024 * 1024)
    
    writeThroughCache := &cache.WriteThroughCache{
        Repository: memoryCache,
        StoreFunc:  createBatchStorer(),
    }
    
    ctx := context.Background()
    
    // 批量写入测试
    fmt.Println("批量写入测试...")
    start := time.Now()
    
    var wg sync.WaitGroup
    errors := make(chan error, 100)
    
    // 并发写入100个条目
    for i := 0; i < 100; i++ {
        wg.Add(1)
        go func(id int) {
            defer wg.Done()
            
            key := fmt.Sprintf("batch:item:%d", id)
            value := map[string]interface{}{
                "id":    id,
                "data":  fmt.Sprintf("data_%d", id),
                "timestamp": time.Now(),
            }
            
            err := writeThroughCache.Set(ctx, key, value, time.Hour)
            if err != nil {
                errors <- fmt.Errorf("写入%s失败: %w", key, err)
            }
        }(i)
    }
    
    wg.Wait()
    close(errors)
    
    duration := time.Since(start)
    
    // 统计结果
    errorCount := 0
    for err := range errors {
        log.Printf("错误: %v", err)
        errorCount++
    }
    
    successCount := 100 - errorCount
    
    fmt.Printf("批量写入结果:\n")
    fmt.Printf("  总耗时: %v\n", duration)
    fmt.Printf("  成功写入: %d\n", successCount)
    fmt.Printf("  失败写入: %d\n", errorCount)
    fmt.Printf("  平均延迟: %v\n", duration/100)
    fmt.Printf("  吞吐量: %.2f ops/sec\n", float64(successCount)/duration.Seconds())
}

func createBatchStorer() func(context.Context, string, any) error {
    var mu sync.Mutex
    writeCount := 0
    
    return func(ctx context.Context, key string, val any) error {
        mu.Lock()
        writeCount++
        currentCount := writeCount
        mu.Unlock()
        
        // 模拟数据库写入延迟
        time.Sleep(time.Duration(rand.Intn(20)) * time.Millisecond)
        
        // 模拟偶发错误（5%失败率）
        if rand.Float64() < 0.05 {
            return fmt.Errorf("数据库写入失败 #%d", currentCount)
        }
        
        return nil
    }
}
```

### 5. 监控和指标收集

```go
func demonstrateMonitoring() {
    memoryCache := cache.NewMaxMemoryCache(1024 * 1024)
    
    // 创建带监控的写透缓存
    monitoredCache := &cache.WriteThroughCache{
        Repository: memoryCache,
        StoreFunc:  createMonitoredStorer(),
    }
    
    ctx := context.Background()
    
    // 模拟各种写入操作
    operations := []struct {
        key   string
        value interface{}
    }{
        {"user:1", map[string]string{"name": "Alice"}},
        {"user:2", map[string]string{"name": "Bob"}},
        {"product:1", map[string]interface{}{"name": "Laptop", "price": 999.99}},
        {"error:item", "this will fail"},
        {"user:3", map[string]string{"name": "Charlie"}},
    }
    
    fmt.Println("开始监控测试...")
    
    for i, op := range operations {
        fmt.Printf("操作 %d: 写入 %s\n", i+1, op.key)
        
        start := time.Now()
        err := monitoredCache.Set(ctx, op.key, op.value, time.Hour)
        duration := time.Since(start)
        
        if err != nil {
            fmt.Printf("  失败: %v (耗时: %v)\n", err, duration)
        } else {
            fmt.Printf("  成功 (耗时: %v)\n", duration)
        }
    }
    
    // 获取监控统计
    fmt.Println("\n监控统计:")
    stats := getMonitoringStats()
    fmt.Printf("  总写入次数: %d\n", stats.TotalWrites)
    fmt.Printf("  成功写入: %d\n", stats.SuccessfulWrites)
    fmt.Printf("  失败写入: %d\n", stats.FailedWrites)
    fmt.Printf("  成功率: %.2f%%\n", float64(stats.SuccessfulWrites)/float64(stats.TotalWrites)*100)
    fmt.Printf("  平均延迟: %v\n", stats.AverageLatency)
}

type MonitoringStats struct {
    TotalWrites      int
    SuccessfulWrites int
    FailedWrites     int
    AverageLatency   time.Duration
}

var (
    monitoringMu    sync.Mutex
    totalWrites     int
    successfulWrites int
    failedWrites    int
    totalLatency    time.Duration
)

func createMonitoredStorer() func(context.Context, string, any) error {
    return func(ctx context.Context, key string, val any) error {
        start := time.Now()
        
        // 模拟数据库操作
        time.Sleep(time.Duration(rand.Intn(50)) * time.Millisecond)
        
        var err error
        if strings.HasPrefix(key, "error:") {
            err = errors.New("模拟数据库错误")
        }
        
        // 记录监控数据
        latency := time.Since(start)
        monitoringMu.Lock()
        totalWrites++
        totalLatency += latency
        if err != nil {
            failedWrites++
        } else {
            successfulWrites++
        }
        monitoringMu.Unlock()
        
        return err
    }
}

func getMonitoringStats() MonitoringStats {
    monitoringMu.Lock()
    defer monitoringMu.Unlock()
    
    var avgLatency time.Duration
    if totalWrites > 0 {
        avgLatency = totalLatency / time.Duration(totalWrites)
    }
    
    return MonitoringStats{
        TotalWrites:      totalWrites,
        SuccessfulWrites: successfulWrites,
        FailedWrites:     failedWrites,
        AverageLatency:   avgLatency,
    }
}
```

## 性能特性

### 时间复杂度

- **Set操作**: O(1) + O(store) - 缓存写入 + 持久化存储时间
- **其他操作**: O(1) - 直接委托给底层Repository

### 空间复杂度

- **缓存存储**: O(n) - n为缓存条目数量
- **额外开销**: O(1) - 只有StoreFunc函数引用

### 性能特点

1. **强一致性**: 确保缓存和存储的数据一致性
2. **写入延迟**: 受持久化存储性能影响
3. **事务性**: 持久化失败时不更新缓存

## 注意事项

### 1. StoreFunc设计

```go
// ✅ 推荐：StoreFunc应该是原子的
storeFunc := func(ctx context.Context, key string, val any) error {
    // 原子的数据库操作
    return database.SaveInTransaction(key, val)
}

// ❌ 避免：StoreFunc有多个步骤且可能部分失败
storeFunc := func(ctx context.Context, key string, val any) error {
    database.Save(key, val)        // 可能成功
    return index.Update(key, val)  // 可能失败，导致不一致
}
```

### 2. 错误处理策略

```go
// ✅ 推荐：区分不同类型的存储错误
storeFunc := func(ctx context.Context, key string, val any) error {
    err := database.Save(key, val)
    if err != nil {
        if isTemporaryError(err) {
            return fmt.Errorf("临时存储错误: %w", err)
        }
        return fmt.Errorf("永久存储错误: %w", err)
    }
    return nil
}
```

### 3. 性能优化

```go
// ✅ 推荐：使用连接池和批量操作
storeFunc := func(ctx context.Context, key string, val any) error {
    // 使用连接池
    conn := dbPool.Get()
    defer dbPool.Put(conn)
    
    return conn.Save(key, val)
}

// 考虑批量写入优化
type BatchWriteThroughCache struct {
    *WriteThroughCache
    batchSize int
    batchTimeout time.Duration
}
```

### 4. 限流配置

```go
// ✅ 推荐：在适当的中间件中设置限流
func writeRateLimitMiddleware(next http.Handler) http.Handler {
    limiter := rate.NewLimiter(100, 10) // 100 QPS，突发10
    
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if !limiter.Allow() {
            ctx := context.WithValue(r.Context(), "limited", true)
            r = r.WithContext(ctx)
        }
        next.ServeHTTP(w, r)
    })
}
```

### 5. 监控指标

```go
// ✅ 推荐：监控关键指标
// - 写入成功率
// - 写入延迟分布
// - 存储错误率
// - 缓存一致性检查

func monitorWriteThrough(cache *WriteThroughCache) {
    go func() {
        ticker := time.NewTicker(time.Minute)
        defer ticker.Stop()
        
        for {
            select {
            case <-ticker.C:
                stats := collectWriteThroughStats()
                if stats.ErrorRate > 0.05 {
                    log.Printf("警告: 写透缓存错误率过高: %.2f%%", stats.ErrorRate*100)
                }
                if stats.AverageLatency > time.Second {
                    log.Printf("警告: 写透缓存延迟过高: %v", stats.AverageLatency)
                }
            }
        }
    }()
}
```
