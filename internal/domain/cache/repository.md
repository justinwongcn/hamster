# repository.go - 缓存仓储接口定义

## 文件概述

`repository.go` 定义了缓存系统的仓储接口，实现了不同的缓存访问模式。遵循仓储模式（Repository
Pattern），将数据访问逻辑抽象化，支持多种缓存策略的实现。

## 核心功能

### 1. 基础仓储接口 (Repository)

定义了缓存的基本操作：

```go
type Repository interface {
    Get(ctx context.Context, key string) (any, error)
    Set(ctx context.Context, key string, val any, expiration time.Duration) error
    Delete(ctx context.Context, key string) error
    LoadAndDelete(ctx context.Context, key string) (any, error)
    Keys(ctx context.Context, pattern string) ([]string, error)
    Exists(ctx context.Context, key string) (bool, error)
    Clear(ctx context.Context) error
    Stats(ctx context.Context) (map[string]any, error)
}
```

**主要方法说明：**

- `Get`: 获取缓存值
- `Set`: 设置缓存值和过期时间
- `Delete`: 删除指定键
- `LoadAndDelete`: 原子性获取并删除
- `Keys`: 根据模式匹配键
- `Exists`: 检查键是否存在
- `Clear`: 清空所有缓存
- `Stats`: 获取统计信息

### 2. 读透仓储接口 (ReadThroughRepository)

扩展基础仓储，支持读透模式：

```go
type ReadThroughRepository interface {
    Repository
    GetWithLoader(ctx context.Context, key string, loader func(ctx context.Context, key string) (any, error)) (any, error)
}
```

**特点：**

- 缓存未命中时自动从数据源加载
- 加载的数据自动写入缓存
- 对调用者透明

### 3. 写透仓储接口 (WriteThroughRepository)

支持写透模式的仓储：

```go
type WriteThroughRepository interface {
    Repository
    SetWithWriter(ctx context.Context, key string, val any, expiration time.Duration, writer func(ctx context.Context, key string, val any) error) error
}
```

**特点：**

- 写入缓存的同时写入持久化存储
- 保证数据一致性
- 写入失败时回滚缓存操作

### 4. 写回仓储接口 (WriteBackRepository)

支持写回模式的仓储：

```go
type WriteBackRepository interface {
    Repository
    MarkDirty(ctx context.Context, key string) error
    FlushDirty(ctx context.Context) error
    FlushKey(ctx context.Context, key string) error
    GetDirtyKeys(ctx context.Context) ([]string, error)
}
```

**特点：**

- 写入时只更新缓存，标记为脏数据
- 异步批量写入持久化存储
- 提供手动刷新机制

## 使用示例

### 基础仓储使用

```go
func ExampleBasicRepository(repo Repository) {
    ctx := context.Background()
    
    // 设置缓存
    err := repo.Set(ctx, "user:123", "John Doe", time.Hour)
    if err != nil {
        log.Printf("设置缓存失败: %v", err)
        return
    }
    
    // 获取缓存
    value, err := repo.Get(ctx, "user:123")
    if err != nil {
        log.Printf("获取缓存失败: %v", err)
        return
    }
    
    fmt.Printf("用户信息: %v\n", value)
}
```

### 读透模式使用

```go
func ExampleReadThrough(repo ReadThroughRepository) {
    ctx := context.Background()
    
    // 定义数据加载器
    loader := func(ctx context.Context, key string) (any, error) {
        // 从数据库加载用户信息
        return loadUserFromDB(key)
    }
    
    // 获取数据（自动处理缓存未命中）
    user, err := repo.GetWithLoader(ctx, "user:123", loader)
    if err != nil {
        log.Printf("获取用户失败: %v", err)
        return
    }
    
    fmt.Printf("用户信息: %v\n", user)
}
```

### 写透模式使用

```go
func ExampleWriteThrough(repo WriteThroughRepository) {
    ctx := context.Background()
    
    // 定义数据写入器
    writer := func(ctx context.Context, key string, val any) error {
        // 写入数据库
        return saveUserToDB(key, val)
    }
    
    // 设置数据（同时写入缓存和数据库）
    err := repo.SetWithWriter(ctx, "user:123", "John Doe", time.Hour, writer)
    if err != nil {
        log.Printf("保存用户失败: %v", err)
        return
    }
    
    fmt.Println("用户信息已保存")
}
```

### 写回模式使用

```go
func ExampleWriteBack(repo WriteBackRepository) {
    ctx := context.Background()
    
    // 设置数据（只写入缓存）
    err := repo.Set(ctx, "user:123", "John Doe", time.Hour)
    if err != nil {
        log.Printf("设置缓存失败: %v", err)
        return
    }
    
    // 标记为脏数据
    err = repo.MarkDirty(ctx, "user:123")
    if err != nil {
        log.Printf("标记脏数据失败: %v", err)
        return
    }
    
    // 批量刷新脏数据
    err = repo.FlushDirty(ctx)
    if err != nil {
        log.Printf("刷新脏数据失败: %v", err)
        return
    }
    
    fmt.Println("脏数据已刷新到持久化存储")
}
```

## 注意事项

### 1. 错误处理

```go
// ✅ 正确：检查特定错误类型
value, err := repo.Get(ctx, "key")
if err != nil {
    if errors.Is(err, ErrKeyNotFound) {
        // 处理键不存在的情况
        return handleKeyNotFound()
    }
    // 处理其他错误
    return err
}
```

### 2. 上下文使用

```go
// ✅ 正确：传递上下文进行超时控制
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

value, err := repo.Get(ctx, "key")
```

### 3. 过期时间设置

```go
// ✅ 正确：使用合适的过期时间
err := repo.Set(ctx, "session:123", sessionData, 30*time.Minute)

// ❌ 错误：过期时间过短可能导致频繁缓存未命中
err := repo.Set(ctx, "user:123", userData, time.Millisecond)
```

### 4. 并发安全

- 仓储接口的实现必须是线程安全的
- 多个goroutine可以同时调用仓储方法
- 实现者需要处理并发访问的同步问题

### 5. 资源清理

```go
// 在适当的时候清理资源
defer func() {
    if err := repo.Clear(ctx); err != nil {
        log.Printf("清理缓存失败: %v", err)
    }
}()
```

## 实现指南

### 1. 实现基础仓储

```go
type MyRepository struct {
    storage map[string]any
    mu      sync.RWMutex
}

func (r *MyRepository) Get(ctx context.Context, key string) (any, error) {
    r.mu.RLock()
    defer r.mu.RUnlock()
    
    value, exists := r.storage[key]
    if !exists {
        return nil, ErrKeyNotFound
    }
    
    return value, nil
}

func (r *MyRepository) Set(ctx context.Context, key string, val any, expiration time.Duration) error {
    r.mu.Lock()
    defer r.mu.Unlock()
    
    r.storage[key] = val
    return nil
}
```

### 2. 扩展特定模式

```go
type MyReadThroughRepository struct {
    *MyRepository
}

func (r *MyReadThroughRepository) GetWithLoader(ctx context.Context, key string, loader func(ctx context.Context, key string) (any, error)) (any, error) {
    // 先尝试从缓存获取
    value, err := r.Get(ctx, key)
    if err == nil {
        return value, nil
    }
    
    // 缓存未命中，使用加载器
    if !errors.Is(err, ErrKeyNotFound) {
        return nil, err
    }
    
    // 从数据源加载
    value, err = loader(ctx, key)
    if err != nil {
        return nil, err
    }
    
    // 写入缓存
    _ = r.Set(ctx, key, value, time.Hour)
    
    return value, nil
}
```
