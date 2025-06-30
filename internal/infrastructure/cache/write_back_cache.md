# write_back_cache.go - 写回缓存实现

## 文件概述

`write_back_cache.go` 实现了写回缓存模式，写入时只更新缓存，通过异步批量写入或定时刷新的方式将脏数据写入持久化存储。该实现提供了高性能的写入操作和灵活的刷新策略，适用于写多读少且能容忍一定数据丢失风险的场景。

## 核心功能

### 1. WriteBackCache 结构体

```go
type WriteBackCache struct {
    cache.Repository                 // 嵌入领域仓储接口
    dirtyKeys        map[string]bool // 脏数据键集合
    dirtyMutex       sync.RWMutex    // 脏数据锁
    flushInterval    time.Duration   // 刷新间隔
    batchSize        int             // 批量大小
    lastFlushTime    time.Time       // 上次刷新时间
    flushMutex       sync.Mutex      // 刷新锁
}
```

**设计特点：**

- 嵌入Repository接口，支持所有标准缓存操作
- 维护脏数据键集合，跟踪需要刷新的数据
- 支持基于时间间隔和批量大小的刷新策略
- 线程安全的并发访问控制
- 自动和手动刷新机制

### 2. 构造函数

```go
func NewWriteBackCache(repository cache.Repository, flushInterval time.Duration, batchSize int) *WriteBackCache
```

**参数说明：**

- `repository`: 底层缓存仓储实现
- `flushInterval`: 刷新间隔时间
- `batchSize`: 触发刷新的脏数据数量阈值

**示例：**

```go
// 创建写回缓存
memoryCache := cache.NewMaxMemoryCache(1024 * 1024) // 1MB
writeBackCache := cache.NewWriteBackCache(
    memoryCache,
    time.Minute,  // 每分钟刷新一次
    100,          // 100个脏数据时触发刷新
)
```

## 主要方法

### 1. 写入操作

#### SetDirty - 设置脏数据

```go
func (w *WriteBackCache) SetDirty(ctx context.Context, key string, val any, expiration time.Duration) error
```

**执行流程：**

1. 写入缓存
2. 标记为脏数据
3. 不立即写入持久化存储

#### Set - 标准写入接口

```go
func (w *WriteBackCache) Set(ctx context.Context, key string, val any, expiration time.Duration) error
```

实际调用SetDirty方法，保持接口一致性。

**示例：**

```go
// 写入数据（只更新缓存）
user := User{ID: "123", Name: "John"}
err := writeBackCache.Set(ctx, "user:123", user, time.Hour)
if err != nil {
    log.Printf("写入缓存失败: %v", err)
    return
}

// 数据已写入缓存并标记为脏数据
fmt.Println("数据写入缓存成功，等待异步刷新到存储")
```

### 2. 刷新操作

#### FlushKey - 刷新单个键

```go
func (w *WriteBackCache) FlushKey(ctx context.Context, key string, storer func(ctx context.Context, key string, val any) error) error
```

**执行逻辑：**

1. 检查键是否为脏数据
2. 从缓存获取值
3. 调用storer函数写入持久化存储
4. 清除脏数据标记

**示例：**

```go
// 定义存储函数
storer := func(ctx context.Context, key string, val any) error {
    return database.Save(key, val)
}

// 强制刷新特定键
err := writeBackCache.FlushKey(ctx, "user:123", storer)
if err != nil {
    log.Printf("刷新键失败: %v", err)
}
```

#### Flush - 刷新所有脏数据

```go
func (w *WriteBackCache) Flush(ctx context.Context, storer func(ctx context.Context, key string, val any) error) error
```

**批量刷新逻辑：**

1. 获取所有脏数据键
2. 批量从缓存读取值
3. 批量写入持久化存储
4. 清理成功写入的脏数据标记
5. 返回组合错误信息

**示例：**

```go
// 手动刷新所有脏数据
err := writeBackCache.Flush(ctx, storer)
if err != nil {
    log.Printf("批量刷新失败: %v", err)
}

fmt.Printf("刷新完成，剩余脏数据: %d\n", writeBackCache.GetDirtyCount())
```

### 3. 自动刷新

#### StartAutoFlush - 启动自动刷新

```go
func (w *WriteBackCache) StartAutoFlush(ctx context.Context, storer func(ctx context.Context, key string, val any) error)
```

**自动刷新特性：**

- 定期检查是否需要刷新
- 基于时间间隔或批量大小触发
- 上下文取消时执行最后一次刷新
- 使用较短的检查间隔确保及时响应

**示例：**

```go
// 启动自动刷新（在goroutine中运行）
ctx, cancel := context.WithCancel(context.Background())
defer cancel()

go writeBackCache.StartAutoFlush(ctx, storer)

// 程序运行期间自动刷新
time.Sleep(5 * time.Minute)

// 停止自动刷新
cancel()
```

#### ShouldFlush - 判断是否需要刷新

```go
func (w *WriteBackCache) ShouldFlush() bool
```

**刷新条件：**

1. 脏数据数量达到批量大小阈值
2. 距离上次刷新时间超过刷新间隔且有脏数据

### 4. 状态查询

#### GetDirtyKeys - 获取脏数据键

```go
func (w *WriteBackCache) GetDirtyKeys() []string
```

#### GetDirtyCount - 获取脏数据数量

```go
func (w *WriteBackCache) GetDirtyCount() int
```

**示例：**

```go
// 监控脏数据状态
dirtyCount := writeBackCache.GetDirtyCount()
dirtyKeys := writeBackCache.GetDirtyKeys()

fmt.Printf("脏数据数量: %d\n", dirtyCount)
fmt.Printf("脏数据键: %v\n", dirtyKeys)

if dirtyCount > 50 {
    fmt.Println("脏数据较多，考虑手动刷新")
}
```

### 5. 删除操作

#### Delete - 删除并清理脏数据标记

```go
func (w *WriteBackCache) Delete(ctx context.Context, key string) error
```

#### LoadAndDelete - 获取并删除

```go
func (w *WriteBackCache) LoadAndDelete(ctx context.Context, key string) (any, error)
```

**删除逻辑：**

- 从底层缓存删除数据
- 清理对应的脏数据标记
- 无论删除是否成功都清理标记

### 6. 淘汰处理

#### OnEvicted - 设置淘汰回调

```go
func (w *WriteBackCache) OnEvicted(fn func(key string, val any))
```

**淘汰处理：**

- 检查被淘汰的数据是否为脏数据
- 脏数据被淘汰时清理标记并记录日志
- 调用原始回调函数

## 使用示例

### 1. 基本写回缓存使用

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
    
    // 创建写回缓存
    writeBackCache := cache.NewWriteBackCache(
        memoryCache,
        30*time.Second, // 30秒刷新间隔
        10,             // 10个脏数据触发刷新
    )
    
    // 定义存储函数
    storer := func(ctx context.Context, key string, val any) error {
        fmt.Printf("写入数据库: %s = %v\n", key, val)
        time.Sleep(10 * time.Millisecond) // 模拟数据库延迟
        return nil
    }
    
    ctx := context.Background()
    
    // 写入数据（只更新缓存）
    for i := 1; i <= 5; i++ {
        user := map[string]interface{}{
            "id":   fmt.Sprintf("%d", i),
            "name": fmt.Sprintf("User%d", i),
        }
        
        err := writeBackCache.Set(ctx, fmt.Sprintf("user:%d", i), user, time.Hour)
        if err != nil {
            log.Printf("写入失败: %v", err)
            continue
        }
        
        fmt.Printf("写入用户%d到缓存\n", i)
    }
    
    fmt.Printf("脏数据数量: %d\n", writeBackCache.GetDirtyCount())
    
    // 手动刷新
    fmt.Println("手动刷新脏数据...")
    err := writeBackCache.Flush(ctx, storer)
    if err != nil {
        log.Printf("刷新失败: %v", err)
    }
    
    fmt.Printf("刷新后脏数据数量: %d\n", writeBackCache.GetDirtyCount())
}
```

### 2. 自动刷新使用

```go
func demonstrateAutoFlush() {
    memoryCache := cache.NewMaxMemoryCache(1024 * 1024)
    writeBackCache := cache.NewWriteBackCache(
        memoryCache,
        time.Minute, // 1分钟刷新间隔
        5,           // 5个脏数据触发刷新
    )
    
    // 定义存储函数
    storer := func(ctx context.Context, key string, val any) error {
        fmt.Printf("自动刷新: %s = %v\n", key, val)
        return nil
    }
    
    // 启动自动刷新
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()
    
    go writeBackCache.StartAutoFlush(ctx, storer)
    
    // 模拟写入操作
    for i := 1; i <= 12; i++ {
        data := fmt.Sprintf("data_%d", i)
        err := writeBackCache.Set(ctx, fmt.Sprintf("key:%d", i), data, time.Hour)
        if err != nil {
            log.Printf("写入失败: %v", err)
            continue
        }
        
        fmt.Printf("写入 key:%d, 脏数据数量: %d\n", i, writeBackCache.GetDirtyCount())
        
        // 每写入一个数据后等待一下
        time.Sleep(200 * time.Millisecond)
        
        // 当脏数据达到5个时会自动刷新
        if i == 5 || i == 10 {
            time.Sleep(100 * time.Millisecond) // 等待自动刷新完成
            fmt.Printf("自动刷新后脏数据数量: %d\n", writeBackCache.GetDirtyCount())
        }
    }
    
    // 停止自动刷新前等待最后一次刷新
    time.Sleep(200 * time.Millisecond)
    cancel()
    
    fmt.Printf("最终脏数据数量: %d\n", writeBackCache.GetDirtyCount())
}
```

### 3. 错误处理和恢复

```go
func demonstrateErrorHandling() {
    memoryCache := cache.NewMaxMemoryCache(1024 * 1024)
    writeBackCache := cache.NewWriteBackCache(
        memoryCache,
        time.Minute,
        5,
    )
    
    // 创建可能失败的存储函数
    storer := func(ctx context.Context, key string, val any) error {
        if strings.Contains(key, "error") {
            return errors.New("模拟数据库错误")
        }
        
        if strings.Contains(key, "timeout") {
            time.Sleep(2 * time.Second)
            return context.DeadlineExceeded
        }
        
        fmt.Printf("成功存储: %s\n", key)
        return nil
    }
    
    ctx := context.Background()
    
    // 写入各种类型的数据
    testData := map[string]string{
        "success:1": "正常数据1",
        "success:2": "正常数据2",
        "error:1":   "错误数据1",
        "success:3": "正常数据3",
        "error:2":   "错误数据2",
        "timeout:1": "超时数据1",
    }
    
    // 写入所有数据
    for key, value := range testData {
        err := writeBackCache.Set(ctx, key, value, time.Hour)
        if err != nil {
            log.Printf("写入缓存失败: %s - %v", key, err)
        }
    }
    
    fmt.Printf("写入完成，脏数据数量: %d\n", writeBackCache.GetDirtyCount())
    fmt.Printf("脏数据键: %v\n", writeBackCache.GetDirtyKeys())
    
    // 尝试刷新
    fmt.Println("开始刷新...")
    err := writeBackCache.Flush(ctx, storer)
    if err != nil {
        fmt.Printf("刷新过程中有错误: %v\n", err)
    }
    
    fmt.Printf("刷新后脏数据数量: %d\n", writeBackCache.GetDirtyCount())
    fmt.Printf("剩余脏数据键: %v\n", writeBackCache.GetDirtyKeys())
    
    // 重试失败的数据
    fmt.Println("重试失败的数据...")
    retryStorer := func(ctx context.Context, key string, val any) error {
        if strings.Contains(key, "error") {
            fmt.Printf("重试成功: %s\n", key)
            return nil // 假设重试成功
        }
        return storer(ctx, key, val)
    }
    
    err = writeBackCache.Flush(ctx, retryStorer)
    if err != nil {
        fmt.Printf("重试后仍有错误: %v\n", err)
    }
    
    fmt.Printf("最终脏数据数量: %d\n", writeBackCache.GetDirtyCount())
}
```

### 4. 性能监控

```go
func demonstratePerformanceMonitoring() {
    memoryCache := cache.NewMaxMemoryCache(1024 * 1024)
    writeBackCache := cache.NewWriteBackCache(
        memoryCache,
        time.Minute,
        50, // 较大的批量大小
    )
    
    // 创建带监控的存储函数
    var (
        totalFlushes    int64
        totalFlushTime  time.Duration
        totalFlushItems int64
        mu              sync.Mutex
    )
    
    storer := func(ctx context.Context, key string, val any) error {
        start := time.Now()
        
        // 模拟数据库写入
        time.Sleep(time.Duration(rand.Intn(10)) * time.Millisecond)
        
        // 记录统计信息
        mu.Lock()
        totalFlushes++
        totalFlushTime += time.Since(start)
        totalFlushItems++
        mu.Unlock()
        
        return nil
    }
    
    ctx := context.Background()
    
    // 启动监控
    go func() {
        ticker := time.NewTicker(5 * time.Second)
        defer ticker.Stop()
        
        for {
            select {
            case <-ctx.Done():
                return
            case <-ticker.C:
                mu.Lock()
                dirtyCount := writeBackCache.GetDirtyCount()
                avgFlushTime := time.Duration(0)
                if totalFlushes > 0 {
                    avgFlushTime = totalFlushTime / time.Duration(totalFlushes)
                }
                mu.Unlock()
                
                fmt.Printf("监控报告: 脏数据=%d, 总刷新=%d, 平均刷新时间=%v\n",
                    dirtyCount, totalFlushes, avgFlushTime)
            }
        }
    }()
    
    // 启动自动刷新
    go writeBackCache.StartAutoFlush(ctx, storer)
    
    // 模拟高频写入
    fmt.Println("开始高频写入测试...")
    start := time.Now()
    
    for i := 0; i < 200; i++ {
        key := fmt.Sprintf("perf:item:%d", i)
        value := map[string]interface{}{
            "id":        i,
            "data":      fmt.Sprintf("data_%d", i),
            "timestamp": time.Now(),
        }
        
        err := writeBackCache.Set(ctx, key, value, time.Hour)
        if err != nil {
            log.Printf("写入失败: %v", err)
        }
        
        // 模拟写入间隔
        if i%10 == 0 {
            time.Sleep(10 * time.Millisecond)
        }
    }
    
    writeTime := time.Since(start)
    
    // 等待所有数据刷新完成
    fmt.Println("等待刷新完成...")
    for writeBackCache.GetDirtyCount() > 0 {
        time.Sleep(100 * time.Millisecond)
    }
    
    totalTime := time.Since(start)
    
    // 输出性能报告
    mu.Lock()
    fmt.Printf("\n性能测试报告:\n")
    fmt.Printf("  写入200个条目耗时: %v\n", writeTime)
    fmt.Printf("  总耗时（包括刷新）: %v\n", totalTime)
    fmt.Printf("  写入吞吐量: %.2f ops/sec\n", 200.0/writeTime.Seconds())
    fmt.Printf("  总刷新次数: %d\n", totalFlushes)
    fmt.Printf("  总刷新条目: %d\n", totalFlushItems)
    fmt.Printf("  平均每次刷新条目: %.2f\n", float64(totalFlushItems)/float64(totalFlushes))
    fmt.Printf("  平均刷新延迟: %v\n", totalFlushTime/time.Duration(totalFlushes))
    mu.Unlock()
}
```

### 5. 淘汰处理

```go
func demonstrateEvictionHandling() {
    // 创建小容量缓存以触发淘汰
    memoryCache := cache.NewMaxMemoryCache(1024) // 1KB，很小的容量
    writeBackCache := cache.NewWriteBackCache(
        memoryCache,
        time.Minute,
        10,
    )
    
    // 设置淘汰回调
    evictedItems := make([]string, 0)
    writeBackCache.OnEvicted(func(key string, val any) {
        evictedItems = append(evictedItems, key)
        fmt.Printf("条目被淘汰: %s = %v\n", key, val)
    })
    
    // 定义存储函数
    storer := func(ctx context.Context, key string, val any) error {
        fmt.Printf("紧急刷新被淘汰的脏数据: %s\n", key)
        return nil
    }
    
    ctx := context.Background()
    
    // 写入大量数据触发淘汰
    fmt.Println("写入大量数据触发淘汰...")
    for i := 1; i <= 20; i++ {
        // 创建较大的数据
        largeData := strings.Repeat(fmt.Sprintf("data_%d_", i), 10)
        
        err := writeBackCache.Set(ctx, fmt.Sprintf("large:item:%d", i), largeData, time.Hour)
        if err != nil {
            log.Printf("写入失败: %v", err)
            continue
        }
        
        fmt.Printf("写入 item:%d, 脏数据数量: %d\n", i, writeBackCache.GetDirtyCount())
        
        // 检查是否有淘汰发生
        if len(evictedItems) > 0 {
            fmt.Printf("检测到淘汰，当前已淘汰: %v\n", evictedItems)
        }
    }
    
    fmt.Printf("最终脏数据数量: %d\n", writeBackCache.GetDirtyCount())
    fmt.Printf("总淘汰条目: %d\n", len(evictedItems))
    
    // 刷新剩余脏数据
    err := writeBackCache.Flush(ctx, storer)
    if err != nil {
        log.Printf("最终刷新失败: %v", err)
    }
}
```

## 性能特性

### 时间复杂度

- **写入操作**: O(1) - 只写缓存，标记脏数据
- **读取操作**: O(1) - 直接从缓存读取
- **刷新操作**: O(n) - n为脏数据数量

### 空间复杂度

- **缓存存储**: O(n) - n为缓存条目数量
- **脏数据标记**: O(m) - m为脏数据数量

### 性能优势

1. **高写入性能**: 写入只更新缓存，延迟极低
2. **批量刷新**: 减少对存储系统的访问频率
3. **异步处理**: 不阻塞业务逻辑执行

## 注意事项

### 1. 数据丢失风险

```go
// ⚠️ 注意：写回缓存有数据丢失风险
// 系统崩溃时未刷新的脏数据会丢失

// 降低风险的策略：
// 1. 设置较短的刷新间隔
writeBackCache := cache.NewWriteBackCache(cache, 30*time.Second, 10)

// 2. 设置较小的批量大小
writeBackCache := cache.NewWriteBackCache(cache, time.Minute, 5)

// 3. 关键数据使用写透缓存
if isCriticalData(key) {
    return writeThroughCache.Set(ctx, key, val, expiration)
}
return writeBackCache.Set(ctx, key, val, expiration)
```

### 2. 内存管理

```go
// ✅ 推荐：监控脏数据数量
if writeBackCache.GetDirtyCount() > maxDirtyCount {
    log.Println("警告: 脏数据过多，强制刷新")
    writeBackCache.Flush(ctx, storer)
}

// ✅ 推荐：设置合理的缓存容量
// 确保缓存不会因为容量不足而频繁淘汰脏数据
```

### 3. 错误处理

```go
// ✅ 推荐：实现重试机制
func createRetryStorer(maxRetries int) func(context.Context, string, any) error {
    return func(ctx context.Context, key string, val any) error {
        var lastErr error
        for i := 0; i < maxRetries; i++ {
            err := database.Save(key, val)
            if err == nil {
                return nil
            }
            lastErr = err
            time.Sleep(time.Duration(i+1) * 100 * time.Millisecond)
        }
        return lastErr
    }
}
```

### 4. 优雅关闭

```go
// ✅ 推荐：程序退出时刷新所有脏数据
func gracefulShutdown(writeBackCache *cache.WriteBackCache, storer func(context.Context, string, any) error) {
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    
    fmt.Println("程序退出，刷新所有脏数据...")
    err := writeBackCache.Flush(ctx, storer)
    if err != nil {
        log.Printf("退出时刷新失败: %v", err)
    } else {
        fmt.Println("所有脏数据已刷新")
    }
}
```

### 5. 监控告警

```go
// ✅ 推荐：设置监控告警
func monitorWriteBackCache(cache *cache.WriteBackCache) {
    go func() {
        ticker := time.NewTicker(time.Minute)
        defer ticker.Stop()
        
        for {
            select {
            case <-ticker.C:
                dirtyCount := cache.GetDirtyCount()
                
                if dirtyCount > 100 {
                    log.Printf("警告: 脏数据过多 (%d)，可能存在刷新问题", dirtyCount)
                }
                
                if cache.ShouldFlush() {
                    log.Printf("提示: 建议执行刷新操作，当前脏数据: %d", dirtyCount)
                }
            }
        }
    }()
}
```
