# service.go - 分布式锁应用服务

## 文件概述

`service.go` 实现了分布式锁应用服务层，遵循DDD架构模式，协调领域服务和基础设施层，实现具体的分布式锁业务用例。该文件提供了完整的分布式锁操作接口，包括获取锁、释放锁、续约锁、状态检查等功能，并内置了多种重试策略。

## 核心功能

### 1. DistributedLockApplicationService 应用服务

```go
type DistributedLockApplicationService struct {
    distributedLock domainLock.DistributedLock
}
```

**设计特点：**

- 协调领域服务和基础设施层
- 实现分布式锁的业务用例
- 提供输入验证和错误处理
- 支持多种重试策略

### 2. 数据传输对象 (DTOs)

#### LockCommand - 加锁命令

```go
type LockCommand struct {
    Key        string        `json:"key"`
    Expiration time.Duration `json:"expiration"`
    Timeout    time.Duration `json:"timeout"`
    RetryType  string        `json:"retry_type"` // "fixed", "exponential", "linear"
    RetryCount int           `json:"retry_count"`
    RetryBase  time.Duration `json:"retry_base"`
}
```

#### LockResult - 锁结果

```go
type LockResult struct {
    Key       string    `json:"key"`
    Value     string    `json:"value"`
    CreatedAt time.Time `json:"created_at"`
    ExpiresAt time.Time `json:"expires_at"`
    IsValid   bool      `json:"is_valid"`
}
```

#### RefreshCommand - 续约命令

```go
type RefreshCommand struct {
    Key string `json:"key"`
}
```

#### AutoRefreshCommand - 自动续约命令

```go
type AutoRefreshCommand struct {
    Key      string        `json:"key"`
    Interval time.Duration `json:"interval"`
    Timeout  time.Duration `json:"timeout"`
}
```

## 主要方法

### 1. 锁获取操作

#### TryLock - 尝试获取锁（不重试）

```go
func (s *DistributedLockApplicationService) TryLock(ctx context.Context, cmd LockCommand) (*LockResult, error)
```

**用例**: 用户想要快速尝试获取锁，如果失败立即返回

**示例：**

```go
service := NewDistributedLockApplicationService(distributedLock)

cmd := LockCommand{
    Key:        "resource:123",
    Expiration: time.Minute,
    Timeout:    time.Second,
}

result, err := service.TryLock(ctx, cmd)
if err != nil {
    log.Printf("获取锁失败: %v", err)
    return
}

fmt.Printf("获取锁成功: %s, 过期时间: %v\n", result.Key, result.ExpiresAt)
```

#### Lock - 获取锁（支持重试）

```go
func (s *DistributedLockApplicationService) Lock(ctx context.Context, cmd LockCommand) (*LockResult, error)
```

**用例**: 用户想要获取锁，如果失败则按策略重试

**示例：**

```go
cmd := LockCommand{
    Key:        "resource:456",
    Expiration: time.Minute,
    Timeout:    10 * time.Second,
    RetryType:  "exponential",
    RetryCount: 5,
    RetryBase:  100 * time.Millisecond,
}

result, err := service.Lock(ctx, cmd)
if err != nil {
    log.Printf("获取锁失败: %v", err)
    return
}

fmt.Printf("获取锁成功: %s\n", result.Key)
```

#### SingleflightLock - 使用SingleFlight优化的获取锁

```go
func (s *DistributedLockApplicationService) SingleflightLock(ctx context.Context, cmd LockCommand) (*LockResult, error)
```

**用例**: 用户想要获取锁，本地goroutine先竞争，减少对分布式锁的压力

### 2. 锁续约操作

#### RefreshLock - 手动续约锁

```go
func (s *DistributedLockApplicationService) RefreshLock(ctx context.Context, cmd RefreshCommand, lock domainLock.Lock) error
```

**用例**: 用户想要延长锁的有效期

**示例：**

```go
refreshCmd := RefreshCommand{Key: "resource:123"}
err := service.RefreshLock(ctx, refreshCmd, lock)
if err != nil {
    log.Printf("续约失败: %v", err)
}
```

#### StartAutoRefresh - 启动自动续约

```go
func (s *DistributedLockApplicationService) StartAutoRefresh(cmd AutoRefreshCommand, lock domainLock.Lock) error
```

**用例**: 用户想要自动续约锁，避免锁过期

**示例：**

```go
autoRefreshCmd := AutoRefreshCommand{
    Key:      "resource:123",
    Interval: 30 * time.Second,
    Timeout:  5 * time.Second,
}

err := service.StartAutoRefresh(autoRefreshCmd, lock)
if err != nil {
    log.Printf("启动自动续约失败: %v", err)
}
```

### 3. 锁释放和状态检查

#### UnlockLock - 释放锁

```go
func (s *DistributedLockApplicationService) UnlockLock(ctx context.Context, cmd UnlockCommand, lock domainLock.Lock) error
```

#### CheckLockStatus - 检查锁状态

```go
func (s *DistributedLockApplicationService) CheckLockStatus(ctx context.Context, query LockQuery, lock domainLock.Lock) (*LockResult, error)
```

## 重试策略

### 1. NoRetryStrategy - 不重试策略

```go
type NoRetryStrategy struct{}
```

不产生任何重试间隔，立即失败。

### 2. FixedIntervalRetryStrategy - 固定间隔重试策略

```go
type FixedIntervalRetryStrategy struct {
    interval time.Duration
    maxRetry int
}
```

每次重试使用相同的时间间隔。

### 3. ExponentialBackoffRetryStrategy - 指数退避重试策略

```go
type ExponentialBackoffRetryStrategy struct {
    initialInterval time.Duration
    multiplier      float64
    maxRetry        int
}
```

每次重试的间隔按指数增长。

### 4. LinearBackoffRetryStrategy - 线性退避重试策略

```go
type LinearBackoffRetryStrategy struct {
    initialInterval time.Duration
    increment       time.Duration
    maxRetry        int
}
```

每次重试的间隔线性增长。

## 使用示例

### 1. 基本锁操作

```go
package main

import (
    "context"
    "fmt"
    "log"
    "time"
    
    "github.com/justinwongcn/hamster/internal/application/lock"
    domainLock "github.com/justinwongcn/hamster/internal/domain/lock"
    infraLock "github.com/justinwongcn/hamster/internal/infrastructure/lock"
)

func main() {
    // 创建分布式锁实现
    distributedLock := infraLock.NewMemoryDistributedLock()
    
    // 创建应用服务
    service := lock.NewDistributedLockApplicationService(distributedLock)
    
    ctx := context.Background()
    
    // 尝试获取锁
    cmd := lock.LockCommand{
        Key:        "critical_resource",
        Expiration: time.Minute,
        Timeout:    5 * time.Second,
        RetryType:  "fixed",
        RetryCount: 3,
        RetryBase:  time.Second,
    }
    
    result, err := service.Lock(ctx, cmd)
    if err != nil {
        log.Printf("获取锁失败: %v", err)
        return
    }
    
    fmt.Printf("获取锁成功: %s, 值: %s\n", result.Key, result.Value)
    fmt.Printf("创建时间: %v, 过期时间: %v\n", result.CreatedAt, result.ExpiresAt)
    
    // 执行业务逻辑
    fmt.Println("执行关键业务逻辑...")
    time.Sleep(2 * time.Second)
    
    // 释放锁
    unlockCmd := lock.UnlockCommand{Key: "critical_resource"}
    err = service.UnlockLock(ctx, unlockCmd, result.Lock)
    if err != nil {
        log.Printf("释放锁失败: %v", err)
    } else {
        fmt.Println("锁释放成功")
    }
}
```

### 2. 不同重试策略演示

```go
func demonstrateRetryStrategies() {
    service := lock.NewDistributedLockApplicationService(distributedLock)
    ctx := context.Background()
    
    // 固定间隔重试
    fmt.Println("测试固定间隔重试...")
    fixedCmd := lock.LockCommand{
        Key:        "resource_fixed",
        Expiration: time.Minute,
        Timeout:    10 * time.Second,
        RetryType:  "fixed",
        RetryCount: 3,
        RetryBase:  time.Second,
    }
    
    start := time.Now()
    result, err := service.Lock(ctx, fixedCmd)
    duration := time.Since(start)
    
    if err == nil {
        fmt.Printf("固定间隔重试成功，耗时: %v\n", duration)
        service.UnlockLock(ctx, lock.UnlockCommand{Key: "resource_fixed"}, result.Lock)
    } else {
        fmt.Printf("固定间隔重试失败: %v，耗时: %v\n", err, duration)
    }
    
    // 指数退避重试
    fmt.Println("测试指数退避重试...")
    expCmd := lock.LockCommand{
        Key:        "resource_exp",
        Expiration: time.Minute,
        Timeout:    10 * time.Second,
        RetryType:  "exponential",
        RetryCount: 4,
        RetryBase:  100 * time.Millisecond,
    }
    
    start = time.Now()
    result, err = service.Lock(ctx, expCmd)
    duration = time.Since(start)
    
    if err == nil {
        fmt.Printf("指数退避重试成功，耗时: %v\n", duration)
        service.UnlockLock(ctx, lock.UnlockCommand{Key: "resource_exp"}, result.Lock)
    } else {
        fmt.Printf("指数退避重试失败: %v，耗时: %v\n", err, duration)
    }
    
    // 线性退避重试
    fmt.Println("测试线性退避重试...")
    linearCmd := lock.LockCommand{
        Key:        "resource_linear",
        Expiration: time.Minute,
        Timeout:    10 * time.Second,
        RetryType:  "linear",
        RetryCount: 3,
        RetryBase:  500 * time.Millisecond,
    }
    
    start = time.Now()
    result, err = service.Lock(ctx, linearCmd)
    duration = time.Since(start)
    
    if err == nil {
        fmt.Printf("线性退避重试成功，耗时: %v\n", duration)
        service.UnlockLock(ctx, lock.UnlockCommand{Key: "resource_linear"}, result.Lock)
    } else {
        fmt.Printf("线性退避重试失败: %v，耗时: %v\n", err, duration)
    }
}
```

### 3. 锁续约演示

```go
func demonstrateLockRefresh() {
    service := lock.NewDistributedLockApplicationService(distributedLock)
    ctx := context.Background()
    
    // 获取锁
    cmd := lock.LockCommand{
        Key:        "refresh_resource",
        Expiration: 10 * time.Second, // 较短的过期时间
        Timeout:    5 * time.Second,
    }
    
    result, err := service.Lock(ctx, cmd)
    if err != nil {
        log.Printf("获取锁失败: %v", err)
        return
    }
    
    fmt.Printf("获取锁成功，初始过期时间: %v\n", result.ExpiresAt)
    
    // 手动续约
    time.Sleep(5 * time.Second)
    refreshCmd := lock.RefreshCommand{Key: "refresh_resource"}
    err = service.RefreshLock(ctx, refreshCmd, result.Lock)
    if err != nil {
        log.Printf("手动续约失败: %v", err)
    } else {
        fmt.Println("手动续约成功")
        
        // 检查锁状态
        query := lock.LockQuery{Key: "refresh_resource"}
        status, err := service.CheckLockStatus(ctx, query, result.Lock)
        if err == nil {
            fmt.Printf("续约后状态: 有效=%v, 新过期时间=%v\n", 
                status.IsValid, status.ExpiresAt)
        }
    }
    
    // 启动自动续约
    autoRefreshCmd := lock.AutoRefreshCommand{
        Key:      "refresh_resource",
        Interval: 3 * time.Second,
        Timeout:  time.Second,
    }
    
    err = service.StartAutoRefresh(autoRefreshCmd, result.Lock)
    if err != nil {
        log.Printf("启动自动续约失败: %v", err)
    } else {
        fmt.Println("自动续约已启动")
        
        // 等待一段时间观察自动续约效果
        time.Sleep(15 * time.Second)
        
        // 检查锁是否仍然有效
        status, err := service.CheckLockStatus(ctx, query, result.Lock)
        if err == nil {
            fmt.Printf("自动续约后状态: 有效=%v\n", status.IsValid)
        }
    }
    
    // 释放锁
    unlockCmd := lock.UnlockCommand{Key: "refresh_resource"}
    service.UnlockLock(ctx, unlockCmd, result.Lock)
}
```

### 4. SingleFlight锁演示

```go
func demonstrateSingleflightLock() {
    service := lock.NewDistributedLockApplicationService(distributedLock)
    ctx := context.Background()
    
    var wg sync.WaitGroup
    results := make(chan *lock.LockResult, 10)
    errors := make(chan error, 10)
    
    // 启动多个goroutine同时尝试获取同一个锁
    for i := 0; i < 10; i++ {
        wg.Add(1)
        go func(id int) {
            defer wg.Done()
            
            cmd := lock.LockCommand{
                Key:        "singleflight_resource",
                Expiration: time.Minute,
                Timeout:    5 * time.Second,
                RetryType:  "fixed",
                RetryCount: 2,
                RetryBase:  time.Second,
            }
            
            start := time.Now()
            result, err := service.SingleflightLock(ctx, cmd)
            duration := time.Since(start)
            
            if err != nil {
                errors <- fmt.Errorf("goroutine %d 失败: %w (耗时: %v)", id, err, duration)
            } else {
                results <- result
                fmt.Printf("Goroutine %d 获取锁成功，耗时: %v\n", id, duration)
            }
        }(i)
    }
    
    wg.Wait()
    close(results)
    close(errors)
    
    // 统计结果
    successCount := 0
    for result := range results {
        successCount++
        if successCount == 1 {
            // 只有第一个成功的需要释放锁
            unlockCmd := lock.UnlockCommand{Key: "singleflight_resource"}
            service.UnlockLock(ctx, unlockCmd, result.Lock)
        }
    }
    
    errorCount := 0
    for err := range errors {
        errorCount++
        fmt.Printf("错误: %v\n", err)
    }
    
    fmt.Printf("SingleFlight结果: 成功=%d, 失败=%d\n", successCount, errorCount)
}
```

### 5. 并发锁竞争演示

```go
func demonstrateConcurrentLocking() {
    service := lock.NewDistributedLockApplicationService(distributedLock)
    ctx := context.Background()
    
    var wg sync.WaitGroup
    lockCount := 0
    var mu sync.Mutex
    
    // 启动多个goroutine竞争不同的锁
    for i := 0; i < 5; i++ {
        wg.Add(1)
        go func(id int) {
            defer wg.Done()
            
            resourceKey := fmt.Sprintf("concurrent_resource_%d", id%3) // 3个不同的资源
            
            cmd := lock.LockCommand{
                Key:        resourceKey,
                Expiration: 5 * time.Second,
                Timeout:    3 * time.Second,
                RetryType:  "exponential",
                RetryCount: 3,
                RetryBase:  100 * time.Millisecond,
            }
            
            result, err := service.Lock(ctx, cmd)
            if err != nil {
                fmt.Printf("Goroutine %d 获取锁 %s 失败: %v\n", id, resourceKey, err)
                return
            }
            
            mu.Lock()
            lockCount++
            mu.Unlock()
            
            fmt.Printf("Goroutine %d 获取锁 %s 成功\n", id, resourceKey)
            
            // 模拟业务处理
            time.Sleep(time.Duration(rand.Intn(2000)) * time.Millisecond)
            
            // 释放锁
            unlockCmd := lock.UnlockCommand{Key: resourceKey}
            err = service.UnlockLock(ctx, unlockCmd, result.Lock)
            if err != nil {
                fmt.Printf("Goroutine %d 释放锁 %s 失败: %v\n", id, resourceKey, err)
            } else {
                fmt.Printf("Goroutine %d 释放锁 %s 成功\n", id, resourceKey)
            }
        }(i)
    }
    
    wg.Wait()
    
    mu.Lock()
    fmt.Printf("总共成功获取锁: %d 次\n", lockCount)
    mu.Unlock()
}
```

## 设计原则

### 1. 应用服务职责

- 协调领域服务和基础设施
- 实现具体的分布式锁业务用例
- 提供输入验证和错误处理
- 转换数据传输对象

### 2. 重试策略设计

- 支持多种重试策略
- 使用迭代器模式实现
- 可配置的重试参数
- 优雅的退避算法

### 3. 错误处理

- 统一的错误包装和传播
- 有意义的错误消息
- 区分业务错误和技术错误

### 4. 并发安全

- 支持SingleFlight优化
- 线程安全的操作
- 避免锁竞争

## 注意事项

### 1. 输入验证

```go
// ✅ 推荐：在应用服务层进行输入验证
func (s *DistributedLockApplicationService) validateLockCommand(cmd LockCommand) error {
    if cmd.Key == "" {
        return fmt.Errorf("锁键不能为空")
    }
    if cmd.Expiration <= 0 {
        return fmt.Errorf("过期时间必须大于0")
    }
    // ... 其他验证
}

// ❌ 避免：跳过输入验证
func (s *DistributedLockApplicationService) Lock(ctx context.Context, cmd LockCommand) (*LockResult, error) {
    // 直接调用底层服务，没有验证
    return s.distributedLock.Lock(ctx, cmd.Key, cmd.Expiration, cmd.Timeout, nil)
}
```

### 2. 重试策略选择

```go
// ✅ 推荐：根据场景选择合适的重试策略
// 高频操作：使用固定间隔，避免过度退避
cmd.RetryType = "fixed"
cmd.RetryBase = 100 * time.Millisecond

// 低频操作：使用指数退避，减少系统压力
cmd.RetryType = "exponential"
cmd.RetryBase = 500 * time.Millisecond

// ❌ 避免：不合理的重试配置
cmd.RetryCount = 100 // 重试次数过多
cmd.RetryBase = 10 * time.Second // 重试间隔过长
```

### 3. 锁的生命周期管理

```go
// ✅ 推荐：确保锁被正确释放
result, err := service.Lock(ctx, cmd)
if err != nil {
    return err
}
defer func() {
    unlockCmd := lock.UnlockCommand{Key: cmd.Key}
    service.UnlockLock(ctx, unlockCmd, result.Lock)
}()

// 执行业务逻辑
// ...

// ❌ 避免：忘记释放锁
result, err := service.Lock(ctx, cmd)
if err != nil {
    return err
}
// 执行业务逻辑后没有释放锁
```

### 4. 自动续约使用

```go
// ✅ 推荐：合理设置自动续约参数
autoRefreshCmd := lock.AutoRefreshCommand{
    Key:      lockKey,
    Interval: lockExpiration / 3, // 续约间隔为过期时间的1/3
    Timeout:  time.Second,        // 续约超时时间要短
}

// ❌ 避免：不合理的续约配置
autoRefreshCmd := lock.AutoRefreshCommand{
    Interval: lockExpiration,     // 续约间隔等于过期时间，可能导致锁过期
    Timeout:  10 * time.Second,   // 续约超时时间过长
}
```

### 5. 错误处理

```go
// ✅ 推荐：区分不同类型的错误
result, err := service.Lock(ctx, cmd)
if err != nil {
    if errors.Is(err, domainLock.ErrLockTimeout) {
        // 锁超时，可以重试或降级处理
    } else if errors.Is(err, domainLock.ErrLockAlreadyHeld) {
        // 锁已被持有，等待或选择其他策略
    } else {
        // 其他错误，记录日志并返回
        log.Printf("获取锁失败: %v", err)
        return err
    }
}
```
