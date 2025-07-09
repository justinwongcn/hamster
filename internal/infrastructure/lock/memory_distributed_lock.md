# memory_distributed_lock.go - 内存分布式锁实现

## 文件概述

`memory_distributed_lock.go`
实现了基于内存的分布式锁，提供了锁的获取、释放、续约等功能。虽然是内存实现，但设计上完全遵循分布式锁的语义，支持UUID标识、过期时间判断、自动续约等特性，并集成了SingleFlight优化。

## 核心功能

### 1. MemoryDistributedLock 结构体

```go
type MemoryDistributedLock struct {
    locks map[string]*memoryLock // 锁存储
    mu    sync.RWMutex           // 读写锁保护
    g     singleflight.Group     // singleflight优化
    stats domainLock.LockStats   // 统计信息
}
```

**主要特性：**

- 基于内存的高性能锁存储
- 支持并发安全的锁操作
- 集成SingleFlight减少重复竞争
- 详细的锁统计信息
- 自动过期锁清理

### 2. memoryLock 锁实例

```go
type memoryLock struct {
    key        string                    // 锁键
    value      string                    // 锁值（UUID）
    expiration time.Duration             // 过期时间
    createdAt  time.Time                 // 创建时间
    unlockChan chan struct{}             // 解锁通知通道
    client     *MemoryDistributedLock    // 锁管理器引用
}
```

**功能特性：**

- UUID唯一标识，防止误解锁
- 基于创建时间的过期判断
- 支持手动和自动续约
- 异步解锁通知机制

## 主要方法

### 1. 构造函数

```go
func NewMemoryDistributedLock() *MemoryDistributedLock
```

**示例：**

```go
lockManager := NewMemoryDistributedLock()
```

### 2. 锁获取操作

#### TryLock - 尝试获取锁

```go
func (mdl *MemoryDistributedLock) TryLock(ctx context.Context, key string, expiration time.Duration) (domainLock.Lock, error)
```

**实现逻辑：**

1. 验证输入参数（键和过期时间）
2. 检查是否已存在锁
3. 检查现有锁是否过期
4. 创建新锁并生成UUID
5. 更新统计信息

**示例：**

```go
lock, err := lockManager.TryLock(ctx, "resource:123", time.Minute)
if err != nil {
    if errors.Is(err, domainLock.ErrFailedToPreemptLock) {
        fmt.Println("锁被其他进程持有")
        return
    }
    return err
}
defer lock.Unlock(ctx)
```

#### Lock - 带重试的获取锁

```go
func (mdl *MemoryDistributedLock) Lock(ctx context.Context, key string, expiration time.Duration, timeout time.Duration, retryStrategy domainLock.RetryStrategy) (domainLock.Lock, error)
```

**实现逻辑：**

1. 首次尝试获取锁
2. 如果失败且是抢锁失败，使用重试策略
3. 在超时时间内循环重试
4. 返回锁实例或超时错误

**示例：**

```go
retryStrategy := NewExponentialBackoffRetryStrategy(100*time.Millisecond, 2.0, 5)
lock, err := lockManager.Lock(ctx, "resource:123", time.Minute, 10*time.Second, retryStrategy)
```

#### SingleflightLock - SingleFlight优化锁

```go
func (mdl *MemoryDistributedLock) SingleflightLock(ctx context.Context, key string, expiration time.Duration, timeout time.Duration, retryStrategy domainLock.RetryStrategy) (domainLock.Lock, error)
```

**优化原理：**

- 同一时间只有一个goroutine去获取特定键的锁
- 其他goroutine等待并共享结果
- 减少锁竞争和系统负载

**示例：**

```go
// 多个goroutine同时调用，只有一个会真正执行锁获取
lock, err := lockManager.SingleflightLock(ctx, "resource:123", time.Minute, 10*time.Second, retryStrategy)
```

### 3. 锁管理操作

#### Refresh - 手动续约

```go
func (ml *memoryLock) Refresh(ctx context.Context) error
```

**实现逻辑：**

1. 检查锁是否仍然存在
2. 验证锁的所有权（UUID匹配）
3. 更新锁的创建时间
4. 更新续约统计

**示例：**

```go
err := lock.Refresh(ctx)
if err != nil {
    if errors.Is(err, domainLock.ErrLockNotHold) {
        fmt.Println("锁已被释放或过期")
    }
    return err
}
```

#### AutoRefresh - 自动续约

```go
func (ml *memoryLock) AutoRefresh(interval time.Duration, timeout time.Duration) error
```

**实现逻辑：**

1. 启动定时器，按间隔执行续约
2. 每次续约都有独立的超时控制
3. 监听解锁信号，及时停止续约
4. 处理续约失败的情况

**示例：**

```go
// 启动自动续约（异步执行）
go func() {
    err := lock.AutoRefresh(30*time.Second, 5*time.Second)
    if err != nil {
        log.Printf("自动续约失败: %v", err)
    }
}()
```

#### Unlock - 释放锁

```go
func (ml *memoryLock) Unlock(ctx context.Context) error
```

**实现逻辑：**

1. 验证锁的所有权
2. 从锁存储中删除锁
3. 发送解锁通知
4. 更新统计信息

**示例：**

```go
err := lock.Unlock(ctx)
if err != nil {
    log.Printf("释放锁失败: %v", err)
}
```

## 重试策略实现

### 1. 固定间隔重试

```go
type FixedIntervalRetryStrategy struct {
    interval time.Duration
    maxRetry int
}

func (s *FixedIntervalRetryStrategy) Iterator() iter.Seq[time.Duration] {
    return func(yield func(time.Duration) bool) {
        for i := 0; i < s.maxRetry; i++ {
            if !yield(s.interval) {
                return
            }
        }
    }
}
```

### 2. 指数退避重试

```go
type ExponentialBackoffRetryStrategy struct {
    initialInterval time.Duration
    multiplier      float64
    maxRetry        int
}

func (s *ExponentialBackoffRetryStrategy) Iterator() iter.Seq[time.Duration] {
    return func(yield func(time.Duration) bool) {
        interval := s.initialInterval
        for i := 0; i < s.maxRetry; i++ {
            if !yield(interval) {
                return
            }
            interval = time.Duration(float64(interval) * s.multiplier)
        }
    }
}
```

### 3. 线性退避重试

```go
type LinearBackoffRetryStrategy struct {
    initialInterval time.Duration
    increment       time.Duration
    maxRetry        int
}

func (s *LinearBackoffRetryStrategy) Iterator() iter.Seq[time.Duration] {
    return func(yield func(time.Duration) bool) {
        interval := s.initialInterval
        for i := 0; i < s.maxRetry; i++ {
            if !yield(interval) {
                return
            }
            interval += s.increment
        }
    }
}
```

## 过期锁清理

### 1. 自动清理机制

```go
func (mdl *MemoryDistributedLock) CleanExpiredLocks() int {
    mdl.mu.Lock()
    defer mdl.mu.Unlock()

    now := time.Now()
    expiredKeys := make([]string, 0)

    // 查找过期的锁
    for key, lock := range mdl.locks {
        lockExpiration, _ := domainLock.NewLockExpiration(lock.expiration)
        if lockExpiration.IsExpired(lock.createdAt, now) {
            expiredKeys = append(expiredKeys, key)
        }
    }

    // 清理过期的锁
    for _, key := range expiredKeys {
        delete(mdl.locks, key)
        mdl.stats = mdl.stats.IncrementExpiredLocks().DecrementActiveLocks()
    }

    return len(expiredKeys)
}
```

### 2. 定期清理任务

```go
func (mdl *MemoryDistributedLock) startCleanupTask() {
    go func() {
        ticker := time.NewTicker(time.Minute) // 每分钟清理一次
        defer ticker.Stop()
        
        for {
            select {
            case <-ticker.C:
                cleaned := mdl.CleanExpiredLocks()
                if cleaned > 0 {
                    log.Printf("清理了 %d 个过期锁", cleaned)
                }
            case <-mdl.stopCh:
                return
            }
        }
    }()
}
```

## 统计信息

### 1. 锁统计收集

```go
type LockStats struct {
    totalLocks    int64  // 总锁数量
    activeLocks   int64  // 活跃锁数量
    failedLocks   int64  // 失败锁数量
    expiredLocks  int64  // 过期锁数量
    refreshCount  int64  // 续约次数
    unlockCount   int64  // 解锁次数
}

func (s LockStats) SuccessRate() float64 {
    if s.totalLocks == 0 {
        return 0
    }
    return float64(s.totalLocks-s.failedLocks) / float64(s.totalLocks)
}
```

### 2. 统计信息获取

```go
func (mdl *MemoryDistributedLock) GetStats() domainLock.LockStats {
    mdl.mu.RLock()
    defer mdl.mu.RUnlock()
    return mdl.stats
}
```

## 使用示例

### 1. 基本锁使用

```go
func processResource(resourceID string) error {
    lockManager := NewMemoryDistributedLock()
    
    // 尝试获取锁
    lockKey := fmt.Sprintf("resource:%s", resourceID)
    lock, err := lockManager.TryLock(ctx, lockKey, time.Minute)
    if err != nil {
        if errors.Is(err, domainLock.ErrFailedToPreemptLock) {
            return fmt.Errorf("资源正在被其他进程处理")
        }
        return err
    }
    defer lock.Unlock(ctx)
    
    // 执行业务逻辑
    return doResourceProcessing(resourceID)
}
```

### 2. 带重试的锁使用

```go
func processWithRetry(resourceID string) error {
    lockManager := NewMemoryDistributedLock()
    
    // 配置重试策略
    retryStrategy := NewExponentialBackoffRetryStrategy(
        100*time.Millisecond, // 初始间隔
        2.0,                  // 倍数因子
        5,                    // 最大重试次数
    )
    
    // 获取锁（带重试）
    lockKey := fmt.Sprintf("resource:%s", resourceID)
    lock, err := lockManager.Lock(ctx, lockKey, time.Minute, 10*time.Second, retryStrategy)
    if err != nil {
        return fmt.Errorf("获取锁失败: %w", err)
    }
    defer lock.Unlock(ctx)
    
    // 启动自动续约
    go func() {
        err := lock.AutoRefresh(30*time.Second, 5*time.Second)
        if err != nil {
            log.Printf("自动续约失败: %v", err)
        }
    }()
    
    // 执行长时间业务逻辑
    return doLongRunningProcess(resourceID)
}
```

### 3. SingleFlight优化使用

```go
func processWithSingleFlight(resourceID string) error {
    lockManager := NewMemoryDistributedLock()
    retryStrategy := NewFixedIntervalRetryStrategy(100*time.Millisecond, 3)
    
    // 多个goroutine同时调用时，只有一个会真正执行锁获取
    lockKey := fmt.Sprintf("resource:%s", resourceID)
    lock, err := lockManager.SingleflightLock(ctx, lockKey, time.Minute, 5*time.Second, retryStrategy)
    if err != nil {
        return err
    }
    defer lock.Unlock(ctx)
    
    return doResourceProcessing(resourceID)
}

// 并发调用示例
func concurrentProcess(resourceID string) {
    var wg sync.WaitGroup
    
    // 启动多个goroutine处理同一资源
    for i := 0; i < 10; i++ {
        wg.Add(1)
        go func(id int) {
            defer wg.Done()
            
            err := processWithSingleFlight(resourceID)
            if err != nil {
                log.Printf("Goroutine %d 处理失败: %v", id, err)
            } else {
                log.Printf("Goroutine %d 处理成功", id)
            }
        }(i)
    }
    
    wg.Wait()
}
```

### 4. 锁状态监控

```go
func monitorLocks(lockManager *MemoryDistributedLock) {
    ticker := time.NewTicker(30 * time.Second)
    defer ticker.Stop()
    
    for {
        select {
        case <-ticker.C:
            stats := lockManager.GetStats()
            
            log.Printf("锁统计信息:")
            log.Printf("  总锁数量: %d", stats.TotalLocks())
            log.Printf("  活跃锁数量: %d", stats.ActiveLocks())
            log.Printf("  失败锁数量: %d", stats.FailedLocks())
            log.Printf("  过期锁数量: %d", stats.ExpiredLocks())
            log.Printf("  续约次数: %d", stats.RefreshCount())
            log.Printf("  解锁次数: %d", stats.UnlockCount())
            log.Printf("  成功率: %.2f%%", stats.SuccessRate()*100)
            
            // 清理过期锁
            cleaned := lockManager.CleanExpiredLocks()
            if cleaned > 0 {
                log.Printf("清理了 %d 个过期锁", cleaned)
            }
        }
    }
}
```

## 注意事项

### 1. 锁粒度控制

```go
// ✅ 推荐：使用细粒度锁
lockKey := fmt.Sprintf("user:%s:profile", userID)

// ❌ 避免：使用粗粒度锁
lockKey := "global_user_lock"
```

### 2. 过期时间设置

```go
// ✅ 推荐：根据业务逻辑设置合理的过期时间
func processOrder(orderID string) error {
    // 订单处理通常需要较长时间
    expiration := 10 * time.Minute
    lock, err := lockManager.TryLock(ctx, fmt.Sprintf("order:%s", orderID), expiration)
    // ...
}

// ❌ 避免：过期时间过短导致业务中断
expiration := time.Second // 太短
```

### 3. 自动续约使用

```go
// ✅ 推荐：为长时间任务启用自动续约
go func() {
    err := lock.AutoRefresh(
        expiration/3,    // 续约间隔为过期时间的1/3
        5*time.Second,   // 续约超时时间
    )
    if err != nil {
        log.Printf("自动续约失败: %v", err)
    }
}()

// ❌ 避免：续约间隔过长导致锁过期
go func() {
    err := lock.AutoRefresh(
        expiration*2,    // 续约间隔大于过期时间
        time.Second,
    )
}()
```

### 4. 错误处理

```go
// ✅ 推荐：区分不同类型的错误
lock, err := lockManager.TryLock(ctx, key, expiration)
if err != nil {
    switch {
    case errors.Is(err, domainLock.ErrFailedToPreemptLock):
        // 锁被占用，可以重试或返回忙碌状态
        return handleLockBusy()
    case errors.Is(err, domainLock.ErrInvalidLockKey):
        // 参数错误，需要修复代码
        return fmt.Errorf("锁键无效: %w", err)
    default:
        // 其他系统错误
        return fmt.Errorf("获取锁失败: %w", err)
    }
}
```
