# types.go - 接口类型定义

## 文件概述

`types.go` 定义了Hamster项目的核心接口类型，为整个系统提供了统一的抽象层。该文件遵循接口隔离原则，定义了简洁而完整的缓存操作接口，为不同的缓存实现提供了标准化的契约。

## 核心接口

### Cache 缓存接口

```go
type Cache interface {
    Set(ctx context.Context, key string, val any, expiration time.Duration) error
    Get(ctx context.Context, key string) (any, error)
    Delete(ctx context.Context, key string) error
    LoadAndDelete(ctx context.Context, key string) (any, error)
    OnEvicted(fn func(key string, val any))
}
```

**设计特点：**

- 提供基本的CRUD操作
- 支持上下文传递
- 灵活的过期时间设置
- 原子的获取并删除操作
- 可配置的淘汰回调

## 方法详解

### 1. Set - 设置缓存值

```go
Set(ctx context.Context, key string, val any, expiration time.Duration) error
```

**参数：**

- `ctx`: 上下文，用于取消操作和传递元数据
- `key`: 缓存键，字符串类型
- `val`: 缓存值，任意类型
- `expiration`: 过期时间，Duration类型

**返回值：**

- `error`: 操作错误，成功时为nil

**用途：**

- 存储键值对到缓存
- 设置数据的过期时间
- 支持覆盖已存在的键

### 2. Get - 获取缓存值

```go
Get(ctx context.Context, key string) (any, error)
```

**参数：**

- `ctx`: 上下文
- `key`: 要获取的缓存键

**返回值：**

- `any`: 缓存值，如果键不存在则为nil
- `error`: 操作错误，键不存在时通常返回特定错误

**用途：**

- 根据键获取对应的缓存值
- 检查键是否存在
- 获取数据用于业务逻辑

### 3. Delete - 删除缓存值

```go
Delete(ctx context.Context, key string) error
```

**参数：**

- `ctx`: 上下文
- `key`: 要删除的缓存键

**返回值：**

- `error`: 操作错误，成功时为nil

**用途：**

- 从缓存中移除指定的键值对
- 清理过期或无效的数据
- 释放缓存空间

### 4. LoadAndDelete - 获取并删除缓存值

```go
LoadAndDelete(ctx context.Context, key string) (any, error)
```

**参数：**

- `ctx`: 上下文
- `key`: 要获取并删除的缓存键

**返回值：**

- `any`: 被删除的缓存值
- `error`: 操作错误

**用途：**

- 原子地获取并删除缓存项
- 实现一次性消费的数据模式
- 避免竞态条件

### 5. OnEvicted - 设置淘汰回调函数

```go
OnEvicted(fn func(key string, val any))
```

**参数：**

- `fn`: 回调函数，当缓存项被淘汰时调用

**用途：**

- 监听缓存项的淘汰事件
- 实现自定义的清理逻辑
- 记录淘汰统计信息

## 使用示例

### 1. 基本缓存操作

```go
package main

import (
    "context"
    "fmt"
    "log"
    "time"
    
    "github.com/justinwongcn/hamster/internal/interfaces"
    "github.com/justinwongcn/hamster/internal/infrastructure/cache"
)

func demonstrateBasicCacheOperations(cache interfaces.Cache) {
    ctx := context.Background()
    
    // 设置缓存
    err := cache.Set(ctx, "user:123", map[string]string{
        "name":  "John Doe",
        "email": "john@example.com",
    }, time.Hour)
    
    if err != nil {
        log.Printf("设置缓存失败: %v", err)
        return
    }
    
    fmt.Println("缓存设置成功")
    
    // 获取缓存
    value, err := cache.Get(ctx, "user:123")
    if err != nil {
        log.Printf("获取缓存失败: %v", err)
        return
    }
    
    if user, ok := value.(map[string]string); ok {
        fmt.Printf("获取到用户: %s (%s)\n", user["name"], user["email"])
    }
    
    // 删除缓存
    err = cache.Delete(ctx, "user:123")
    if err != nil {
        log.Printf("删除缓存失败: %v", err)
        return
    }
    
    fmt.Println("缓存删除成功")
    
    // 验证删除
    _, err = cache.Get(ctx, "user:123")
    if err != nil {
        fmt.Println("确认缓存已删除")
    }
}
```

### 2. LoadAndDelete 原子操作

```go
func demonstrateLoadAndDelete(cache interfaces.Cache) {
    ctx := context.Background()
    
    // 设置一次性令牌
    token := "temp_token_12345"
    err := cache.Set(ctx, "token:"+token, map[string]interface{}{
        "user_id":    123,
        "expires_at": time.Now().Add(time.Minute),
        "scope":      "read_profile",
    }, time.Minute)
    
    if err != nil {
        log.Printf("设置令牌失败: %v", err)
        return
    }
    
    fmt.Println("临时令牌已设置")
    
    // 消费令牌（获取并删除）
    tokenData, err := cache.LoadAndDelete(ctx, "token:"+token)
    if err != nil {
        log.Printf("消费令牌失败: %v", err)
        return
    }
    
    if data, ok := tokenData.(map[string]interface{}); ok {
        fmt.Printf("令牌消费成功，用户ID: %v\n", data["user_id"])
    }
    
    // 尝试再次获取（应该失败）
    _, err = cache.Get(ctx, "token:"+token)
    if err != nil {
        fmt.Println("令牌已被消费，无法再次使用")
    }
}
```

### 3. 淘汰回调处理

```go
func demonstrateEvictionCallback(cache interfaces.Cache) {
    ctx := context.Background()
    
    // 设置淘汰回调
    evictedCount := 0
    cache.OnEvicted(func(key string, val any) {
        evictedCount++
        fmt.Printf("缓存项被淘汰: %s = %v\n", key, val)
        
        // 可以在这里实现自定义逻辑
        // 例如：记录日志、更新统计、清理相关资源等
        if strings.HasPrefix(key, "session:") {
            fmt.Printf("会话 %s 已过期，执行清理操作\n", key)
        }
    })
    
    // 设置一些会过期的数据
    sessions := []string{"session:user1", "session:user2", "session:user3"}
    
    for _, sessionKey := range sessions {
        err := cache.Set(ctx, sessionKey, map[string]interface{}{
            "user_id":    rand.Intn(1000),
            "login_time": time.Now(),
            "ip":         fmt.Sprintf("192.168.1.%d", rand.Intn(255)),
        }, 2*time.Second) // 2秒后过期
        
        if err != nil {
            log.Printf("设置会话失败: %v", err)
        }
    }
    
    fmt.Printf("设置了 %d 个会话，等待过期...\n", len(sessions))
    
    // 等待数据过期和淘汰
    time.Sleep(5 * time.Second)
    
    fmt.Printf("总共淘汰了 %d 个缓存项\n", evictedCount)
}
```

### 4. 上下文使用

```go
func demonstrateContextUsage(cache interfaces.Cache) {
    // 带超时的上下文
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    
    // 带取消的上下文
    cancelCtx, cancelFunc := context.WithCancel(context.Background())
    
    // 模拟长时间操作
    go func() {
        time.Sleep(2 * time.Second)
        fmt.Println("取消操作")
        cancelFunc()
    }()
    
    // 使用带超时的上下文
    err := cache.Set(ctx, "timeout_test", "value", time.Minute)
    if err != nil {
        log.Printf("超时设置失败: %v", err)
    } else {
        fmt.Println("超时设置成功")
    }
    
    // 使用可取消的上下文
    err = cache.Set(cancelCtx, "cancel_test", "value", time.Minute)
    if err != nil {
        log.Printf("取消设置失败: %v", err)
    } else {
        fmt.Println("取消设置成功")
    }
    
    // 带值的上下文
    valueCtx := context.WithValue(context.Background(), "request_id", "req_12345")
    
    err = cache.Set(valueCtx, "value_test", "value", time.Minute)
    if err != nil {
        log.Printf("值上下文设置失败: %v", err)
    } else {
        fmt.Println("值上下文设置成功")
    }
}
```

### 5. 错误处理模式

```go
func demonstrateErrorHandling(cache interfaces.Cache) {
    ctx := context.Background()
    
    // 处理设置错误
    err := cache.Set(ctx, "", "empty_key", time.Minute) // 空键
    if err != nil {
        fmt.Printf("设置空键错误: %v\n", err)
    }
    
    // 处理获取不存在的键
    value, err := cache.Get(ctx, "nonexistent_key")
    if err != nil {
        // 检查是否是"键不存在"错误
        if isKeyNotFoundError(err) {
            fmt.Println("键不存在，这是正常情况")
        } else {
            log.Printf("获取缓存时发生其他错误: %v", err)
        }
    } else {
        fmt.Printf("意外获取到值: %v\n", value)
    }
    
    // 处理删除不存在的键
    err = cache.Delete(ctx, "nonexistent_key")
    if err != nil {
        if isKeyNotFoundError(err) {
            fmt.Println("删除不存在的键，忽略错误")
        } else {
            log.Printf("删除时发生错误: %v", err)
        }
    }
    
    // 处理LoadAndDelete的错误
    value, err = cache.LoadAndDelete(ctx, "nonexistent_key")
    if err != nil {
        if isKeyNotFoundError(err) {
            fmt.Println("LoadAndDelete: 键不存在")
        } else {
            log.Printf("LoadAndDelete时发生错误: %v", err)
        }
    }
}

// 辅助函数：检查是否是键不存在错误
func isKeyNotFoundError(err error) bool {
    // 这里应该根据具体的错误类型进行判断
    // 例如：return errors.Is(err, cache.ErrKeyNotFound)
    return strings.Contains(err.Error(), "not found") || 
           strings.Contains(err.Error(), "键未找到")
}
```

### 6. 批量操作模式

```go
func demonstrateBatchOperations(cache interfaces.Cache) {
    ctx := context.Background()
    
    // 批量设置
    users := map[string]interface{}{
        "user:1": map[string]string{"name": "Alice", "role": "admin"},
        "user:2": map[string]string{"name": "Bob", "role": "user"},
        "user:3": map[string]string{"name": "Charlie", "role": "user"},
    }
    
    fmt.Println("批量设置用户数据...")
    for key, value := range users {
        err := cache.Set(ctx, key, value, time.Hour)
        if err != nil {
            log.Printf("设置 %s 失败: %v", key, err)
        } else {
            fmt.Printf("设置 %s 成功\n", key)
        }
    }
    
    // 批量获取
    fmt.Println("批量获取用户数据...")
    userKeys := []string{"user:1", "user:2", "user:3", "user:4"} // user:4 不存在
    
    for _, key := range userKeys {
        value, err := cache.Get(ctx, key)
        if err != nil {
            if isKeyNotFoundError(err) {
                fmt.Printf("用户 %s 不存在\n", key)
            } else {
                log.Printf("获取 %s 失败: %v", key, err)
            }
        } else {
            if user, ok := value.(map[string]string); ok {
                fmt.Printf("用户 %s: %s (%s)\n", key, user["name"], user["role"])
            }
        }
    }
    
    // 批量删除
    fmt.Println("批量删除用户数据...")
    for _, key := range userKeys[:3] { // 只删除存在的用户
        err := cache.Delete(ctx, key)
        if err != nil {
            log.Printf("删除 %s 失败: %v", key, err)
        } else {
            fmt.Printf("删除 %s 成功\n", key)
        }
    }
}
```

## 接口设计原则

### 1. 简洁性

- 只包含核心的缓存操作
- 避免过度设计和功能膨胀
- 易于理解和实现

### 2. 一致性

- 所有方法都接受context参数
- 统一的错误处理方式
- 一致的命名约定

### 3. 灵活性

- 支持任意类型的值（any类型）
- 可配置的过期时间
- 可选的回调机制

### 4. 可扩展性

- 接口设计允许多种实现
- 支持不同的缓存策略
- 便于添加新功能

## 实现建议

### 1. 错误处理

```go
// ✅ 推荐：定义明确的错误类型
var (
    ErrKeyNotFound = errors.New("键未找到")
    ErrKeyEmpty    = errors.New("键不能为空")
    ErrValueNil    = errors.New("值不能为nil")
)

// ✅ 推荐：在实现中返回具体错误
func (c *MyCache) Get(ctx context.Context, key string) (any, error) {
    if key == "" {
        return nil, ErrKeyEmpty
    }
    
    value, exists := c.data[key]
    if !exists {
        return nil, ErrKeyNotFound
    }
    
    return value, nil
}
```

### 2. 上下文处理

```go
// ✅ 推荐：正确处理上下文
func (c *MyCache) Set(ctx context.Context, key string, val any, expiration time.Duration) error {
    select {
    case <-ctx.Done():
        return ctx.Err()
    default:
    }
    
    // 执行设置操作
    c.data[key] = val
    return nil
}

// ❌ 避免：忽略上下文
func (c *MyCache) Set(ctx context.Context, key string, val any, expiration time.Duration) error {
    // 直接执行操作，没有检查上下文
    c.data[key] = val
    return nil
}
```

### 3. 线程安全

```go
// ✅ 推荐：确保线程安全
type MyCache struct {
    mu   sync.RWMutex
    data map[string]any
}

func (c *MyCache) Get(ctx context.Context, key string) (any, error) {
    c.mu.RLock()
    defer c.mu.RUnlock()
    
    value, exists := c.data[key]
    if !exists {
        return nil, ErrKeyNotFound
    }
    
    return value, nil
}
```

### 4. 过期时间处理

```go
// ✅ 推荐：正确处理过期时间
func (c *MyCache) Set(ctx context.Context, key string, val any, expiration time.Duration) error {
    c.mu.Lock()
    defer c.mu.Unlock()
    
    item := &CacheItem{
        Value:     val,
        ExpiresAt: time.Now().Add(expiration),
    }
    
    c.data[key] = item
    
    // 如果过期时间为0，表示永不过期
    if expiration > 0 {
        c.scheduleExpiration(key, expiration)
    }
    
    return nil
}
```

### 5. 回调函数处理

```go
// ✅ 推荐：安全地调用回调函数
func (c *MyCache) evictItem(key string, value any) {
    if c.onEvicted != nil {
        // 在goroutine中调用回调，避免阻塞
        go func() {
            defer func() {
                if r := recover(); r != nil {
                    log.Printf("淘汰回调函数panic: %v", r)
                }
            }()
            
            c.onEvicted(key, value)
        }()
    }
}
```

## 注意事项

### 1. 接口实现

- 确保所有方法都正确实现
- 处理边界情况和错误条件
- 保持接口行为的一致性

### 2. 性能考虑

- 避免在接口方法中进行重操作
- 合理使用锁和并发控制
- 考虑内存使用和垃圾回收

### 3. 兼容性

- 保持接口的向后兼容性
- 谨慎修改接口定义
- 使用版本控制管理接口变更

### 4. 测试

- 为接口实现编写全面的测试
- 测试各种边界情况
- 验证并发安全性

Cache接口为Hamster项目提供了统一的缓存抽象，支持多种实现方式，是整个缓存系统的核心契约。
