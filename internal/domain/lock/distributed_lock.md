# distributed_lock.go - 分布式锁领域模型

## 文件概述

`distributed_lock.go` 定义了分布式锁的完整领域模型，包括核心接口、值对象和重试策略。该文件遵循DDD设计原则，将分布式锁的业务逻辑封装在领域层，为分布式系统的并发控制提供核心抽象。使用Go
1.23+的迭代器特性实现现代化的重试策略。

## 核心功能

### 1. 错误定义

```go
var (
    ErrFailedToPreemptLock = errors.New("抢锁失败")
    ErrLockNotHold         = errors.New("你没有持有锁")
    ErrLockExpired         = errors.New("锁已过期")
    ErrInvalidLockKey      = errors.New("无效的锁键")
    ErrInvalidExpiration   = errors.New("无效的过期时间")
)
```

定义了分布式锁相关的领域错误，便于错误识别和处理。

### 2. DistributedLock 核心接口

```go
type DistributedLock interface {
    Lock(ctx context.Context, key string, expiration time.Duration, timeout time.Duration, retryStrategy RetryStrategy) (Lock, error)
    TryLock(ctx context.Context, key string, expiration time.Duration) (Lock, error)
    SingleflightLock(ctx context.Context, key string, expiration time.Duration, timeout time.Duration, retryStrategy RetryStrategy) (Lock, error)
}
```

**核心方法：**

- **Lock**: 获取锁（支持重试）
- **TryLock**: 尝试获取锁（不重试）
- **SingleflightLock**: 使用SingleFlight优化的获取锁

### 3. Lock 锁实例接口

```go
type Lock interface {
    Key() string
    Value() string
    Expiration() time.Duration
    CreatedAt() time.Time
    IsExpired(now time.Time) bool
    Refresh(ctx context.Context) error
    AutoRefresh(interval time.Duration, timeout time.Duration) error
    Unlock(ctx context.Context) error
    IsValid(ctx context.Context) (bool, error)
}
```

**锁管理方法：**

- **Key/Value/Expiration**: 获取锁的基本信息
- **CreatedAt**: 获取创建时间
- **IsExpired**: 检查是否过期
- **Refresh**: 手动续约
- **AutoRefresh**: 自动续约
- **Unlock**: 释放锁
- **IsValid**: 检查锁是否有效

### 4. RetryStrategy 重试策略接口

```go
type RetryStrategy interface {
    Iterator() iter.Seq[time.Duration]
}
```

**现代化特性：**

- 使用Go 1.23+的迭代器特性
- 支持函数式的重试间隔定义
- 便于实现各种重试算法

### 5. LockKey 锁键值对象

#### 结构定义

```go
type LockKey struct {
    value string
}
```

#### 构造函数

```go
func NewLockKey(key string) (LockKey, error)
```

**验证规则：**

- 锁键不能为空
- 锁键长度不能超过200个字符

**示例：**

```go
key, err := NewLockKey("resource:user:12345")
if err != nil {
    log.Printf("锁键创建失败: %v", err)
    return
}

fmt.Printf("锁键: %s\n", key.String())
fmt.Printf("是否为空: %v\n", key.IsEmpty())

// 比较锁键
key2, _ := NewLockKey("resource:user:12345")
if key.Equals(key2) {
    fmt.Println("锁键相等")
}
```

#### 操作方法

- `String()`: 返回字符串表示
- `IsEmpty()`: 检查是否为空
- `Equals(other LockKey)`: 比较锁键是否相等

### 6. LockValue 锁值值对象

#### 结构定义

```go
type LockValue struct {
    value string
}
```

#### 构造函数

```go
func NewLockValue(value string) (LockValue, error)
```

**用途：**

- 通常存储UUID作为锁的唯一标识
- 防止误解锁（只有持有正确UUID的进程才能解锁）
- 支持锁的所有权验证

**示例：**

```go
import "github.com/google/uuid"

// 生成UUID作为锁值
lockUUID := uuid.New().String()
lockValue, err := NewLockValue(lockUUID)
if err != nil {
    log.Printf("锁值创建失败: %v", err)
    return
}

fmt.Printf("锁值: %s\n", lockValue.String())

// 验证锁值
if lockValue.Equals(expectedValue) {
    fmt.Println("锁值匹配，可以执行解锁")
}
```

### 7. LockExpiration 锁过期时间值对象

#### 结构定义

```go
type LockExpiration struct {
    duration time.Duration
}
```

#### 构造函数

```go
func NewLockExpiration(duration time.Duration) (LockExpiration, error)
```

**验证规则：**

- 过期时间必须大于0
- 过期时间不能超过24小时

**示例：**

```go
// 创建过期时间
expiration, err := NewLockExpiration(time.Minute)
if err != nil {
    log.Printf("过期时间创建失败: %v", err)
    return
}

createdAt := time.Now()
fmt.Printf("过期时间间隔: %v\n", expiration.Duration())
fmt.Printf("过期时间点: %v\n", expiration.ExpiresAt(createdAt))

// 检查是否过期
now := time.Now()
if expiration.IsExpired(createdAt, now) {
    fmt.Println("锁已过期")
} else {
    remaining := expiration.RemainingTime(createdAt, now)
    fmt.Printf("剩余时间: %v\n", remaining)
}
```

#### 时间计算方法

- `Duration()`: 获取过期时间间隔
- `ExpiresAt(from time.Time)`: 计算过期时间点
- `IsExpired(createdAt, now time.Time)`: 检查是否已过期
- `RemainingTime(createdAt, now time.Time)`: 计算剩余时间

### 8. LockStats 锁统计信息值对象

#### 结构定义

```go
type LockStats struct {
    totalLocks    int64  // 总锁数量
    activeLocks   int64  // 活跃锁数量
    failedLocks   int64  // 失败锁数量
    expiredLocks  int64  // 过期锁数量
    refreshCount  int64  // 续约次数
    unlockCount   int64  // 解锁次数
}
```

#### 统计方法

```go
func (s LockStats) SuccessRate() float64
```

计算锁获取成功率。

**示例：**

```go
stats := NewLockStats()

// 模拟统计更新
stats = stats.IncrementTotalLocks()
stats = stats.IncrementActiveLocks()
stats = stats.IncrementRefreshCount()

fmt.Printf("锁统计信息:\n")
fmt.Printf("  总锁数量: %d\n", stats.TotalLocks())
fmt.Printf("  活跃锁数量: %d\n", stats.ActiveLocks())
fmt.Printf("  失败锁数量: %d\n", stats.FailedLocks())
fmt.Printf("  过期锁数量: %d\n", stats.ExpiredLocks())
fmt.Printf("  续约次数: %d\n", stats.RefreshCount())
fmt.Printf("  解锁次数: %d\n", stats.UnlockCount())
fmt.Printf("  成功率: %.2f%%\n", stats.SuccessRate()*100)
```

#### 更新方法

- `IncrementTotalLocks()`: 增加总锁数量
- `IncrementActiveLocks()`: 增加活跃锁数量
- `DecrementActiveLocks()`: 减少活跃锁数量
- `IncrementFailedLocks()`: 增加失败锁数量
- `IncrementExpiredLocks()`: 增加过期锁数量
- `IncrementRefreshCount()`: 增加续约次数
- `IncrementUnlockCount()`: 增加解锁次数

## 使用示例

### 1. 基本锁键和值对象使用

```go
package main

import (
    "fmt"
    "log"
    "time"
    
    "github.com/google/uuid"
    "github.com/justinwongcn/hamster/internal/domain/lock"
)

func main() {
    // 创建锁键
    key, err := lock.NewLockKey("resource:user:12345")
    if err != nil {
        log.Fatal(err)
    }
    
    // 创建锁值（UUID）
    lockUUID := uuid.New().String()
    value, err := lock.NewLockValue(lockUUID)
    if err != nil {
        log.Fatal(err)
    }
    
    // 创建过期时间
    expiration, err := lock.NewLockExpiration(time.Minute)
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("锁信息:\n")
    fmt.Printf("  键: %s\n", key.String())
    fmt.Printf("  值: %s\n", value.String())
    fmt.Printf("  过期时间: %v\n", expiration.Duration())
}
```

### 2. 过期时间管理

```go
func demonstrateExpiration() {
    // 创建短期锁
    expiration, err := lock.NewLockExpiration(5 * time.Second)
    if err != nil {
        log.Printf("创建过期时间失败: %v", err)
        return
    }
    
    createdAt := time.Now()
    fmt.Printf("锁创建时间: %v\n", createdAt)
    fmt.Printf("锁过期时间: %v\n", expiration.ExpiresAt(createdAt))
    
    // 模拟时间流逝
    for i := 0; i < 7; i++ {
        time.Sleep(time.Second)
        now := time.Now()
        
        if expiration.IsExpired(createdAt, now) {
            fmt.Printf("第%d秒: 锁已过期\n", i+1)
        } else {
            remaining := expiration.RemainingTime(createdAt, now)
            fmt.Printf("第%d秒: 剩余时间 %v\n", i+1, remaining)
        }
    }
}
```

### 3. 统计信息管理

```go
func demonstrateStats() {
    stats := lock.NewLockStats()
    
    // 模拟锁操作统计
    operations := []struct {
        name string
        fn   func(lock.LockStats) lock.LockStats
    }{
        {"获取锁", func(s lock.LockStats) lock.LockStats { 
            return s.IncrementTotalLocks().IncrementActiveLocks() 
        }},
        {"获取锁", func(s lock.LockStats) lock.LockStats { 
            return s.IncrementTotalLocks().IncrementActiveLocks() 
        }},
        {"获取锁失败", func(s lock.LockStats) lock.LockStats { 
            return s.IncrementTotalLocks().IncrementFailedLocks() 
        }},
        {"续约", func(s lock.LockStats) lock.LockStats { 
            return s.IncrementRefreshCount() 
        }},
        {"释放锁", func(s lock.LockStats) lock.LockStats { 
            return s.DecrementActiveLocks().IncrementUnlockCount() 
        }},
        {"锁过期", func(s lock.LockStats) lock.LockStats { 
            return s.DecrementActiveLocks().IncrementExpiredLocks() 
        }},
    }
    
    fmt.Println("锁操作统计:")
    for _, op := range operations {
        stats = op.fn(stats)
        fmt.Printf("执行 %s 后:\n", op.name)
        fmt.Printf("  总锁数: %d, 活跃锁: %d, 失败锁: %d\n", 
            stats.TotalLocks(), stats.ActiveLocks(), stats.FailedLocks())
        fmt.Printf("  成功率: %.2f%%\n", stats.SuccessRate()*100)
    }
}
```

### 4. 锁键验证

```go
func demonstrateKeyValidation() {
    // 测试有效的锁键
    validKeys := []string{
        "user:123",
        "resource:order:456",
        "cache:refresh:789",
    }
    
    fmt.Println("有效锁键测试:")
    for _, keyStr := range validKeys {
        key, err := lock.NewLockKey(keyStr)
        if err != nil {
            fmt.Printf("  %s: 失败 - %v\n", keyStr, err)
        } else {
            fmt.Printf("  %s: 成功\n", key.String())
        }
    }
    
    // 测试无效的锁键
    invalidKeys := []string{
        "",                                    // 空键
        strings.Repeat("a", 201),             // 过长的键
    }
    
    fmt.Println("\n无效锁键测试:")
    for _, keyStr := range invalidKeys {
        _, err := lock.NewLockKey(keyStr)
        if err != nil {
            fmt.Printf("  长度%d的键: 预期失败 - %v\n", len(keyStr), err)
        } else {
            fmt.Printf("  长度%d的键: 意外成功\n", len(keyStr))
        }
    }
}
```

### 5. 过期时间验证

```go
func demonstrateExpirationValidation() {
    // 测试有效的过期时间
    validDurations := []time.Duration{
        time.Second,
        time.Minute,
        time.Hour,
        12 * time.Hour,
    }
    
    fmt.Println("有效过期时间测试:")
    for _, duration := range validDurations {
        expiration, err := lock.NewLockExpiration(duration)
        if err != nil {
            fmt.Printf("  %v: 失败 - %v\n", duration, err)
        } else {
            fmt.Printf("  %v: 成功\n", expiration.Duration())
        }
    }
    
    // 测试无效的过期时间
    invalidDurations := []time.Duration{
        0,                    // 零时间
        -time.Second,         // 负时间
        25 * time.Hour,       // 超过24小时
    }
    
    fmt.Println("\n无效过期时间测试:")
    for _, duration := range invalidDurations {
        _, err := lock.NewLockExpiration(duration)
        if err != nil {
            fmt.Printf("  %v: 预期失败 - %v\n", duration, err)
        } else {
            fmt.Printf("  %v: 意外成功\n", duration)
        }
    }
}
```

### 6. 锁值比较

```go
func demonstrateValueComparison() {
    // 创建两个相同的锁值
    uuid1 := uuid.New().String()
    value1, _ := lock.NewLockValue(uuid1)
    value2, _ := lock.NewLockValue(uuid1)
    
    // 创建不同的锁值
    uuid2 := uuid.New().String()
    value3, _ := lock.NewLockValue(uuid2)
    
    fmt.Println("锁值比较测试:")
    fmt.Printf("value1 == value2: %v\n", value1.Equals(value2)) // true
    fmt.Printf("value1 == value3: %v\n", value1.Equals(value3)) // false
    
    // 模拟锁的所有权验证
    currentLockValue := value1
    attemptUnlockValue := value2
    
    if currentLockValue.Equals(attemptUnlockValue) {
        fmt.Println("锁值匹配，允许解锁")
    } else {
        fmt.Println("锁值不匹配，拒绝解锁")
    }
}
```

## 设计原则

### 1. 值对象不变性

- 所有值对象创建后不可修改
- 通过构造函数进行验证
- 提供只读访问方法

### 2. 领域封装

- 将分布式锁的业务规则封装在领域层
- 隐藏技术实现细节
- 提供类型安全的操作

### 3. 现代化特性

- 使用Go 1.23+的迭代器特性
- 支持函数式编程风格
- 提供灵活的重试策略

### 4. 错误处理

- 定义领域特定的错误类型
- 在构造函数中进行参数验证
- 提供有意义的错误信息

## 注意事项

### 1. 锁键设计

```go
// ✅ 推荐：使用有层次的锁键
key, err := lock.NewLockKey("resource:user:12345")
key, err := lock.NewLockKey("cache:refresh:category:electronics")

// ❌ 避免：锁键过于简单或过长
key, err := lock.NewLockKey("lock") // 太简单
key, err := lock.NewLockKey(strings.Repeat("a", 201)) // 太长
```

### 2. 过期时间设置

```go
// ✅ 推荐：根据业务逻辑设置合理的过期时间
expiration, err := lock.NewLockExpiration(time.Minute)     // 短期操作
expiration, err := lock.NewLockExpiration(10*time.Minute) // 长期操作

// ❌ 避免：过期时间过短或过长
expiration, err := lock.NewLockExpiration(time.Millisecond) // 太短
expiration, err := lock.NewLockExpiration(25*time.Hour)     // 太长
```

### 3. 锁值管理

```go
// ✅ 推荐：使用UUID作为锁值
import "github.com/google/uuid"

lockUUID := uuid.New().String()
value, err := lock.NewLockValue(lockUUID)

// ❌ 避免：使用可预测的锁值
value, err := lock.NewLockValue("simple_value") // 不安全
```

### 4. 统计信息使用

```go
// ✅ 推荐：定期监控锁统计信息
if stats.SuccessRate() < 0.9 {
    log.Println("警告: 锁获取成功率过低")
}

if stats.ActiveLocks() > maxActiveLocks {
    log.Println("警告: 活跃锁数量过多")
}
```

### 5. 过期检查

```go
// ✅ 推荐：在关键操作前检查锁是否过期
now := time.Now()
if expiration.IsExpired(createdAt, now) {
    return lock.ErrLockExpired
}

// 获取剩余时间进行决策
remaining := expiration.RemainingTime(createdAt, now)
if remaining < time.Second {
    log.Println("警告: 锁即将过期")
}
```
