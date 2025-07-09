# Hamster 公共 API 使用指南

Hamster 现在提供了简洁易用的公共 API，让用户可以直接使用库的核心功能，而无需深入了解内部实现细节。

## 快速开始

### 安装

```bash
go get github.com/justinwongcn/hamster
```

### 基本使用

```go
package main

import (
    "context"
    "fmt"
    "time"
    
    "github.com/justinwongcn/hamster"
    "github.com/justinwongcn/hamster/cache"
    "github.com/justinwongcn/hamster/hash"
    "github.com/justinwongcn/hamster/lock"
)

func main() {
    fmt.Printf("Hamster 版本: %s\n", hamster.GetVersion())
    
    // 使用缓存
    cacheService, _ := hamster.NewCache()
    cacheService.Set(context.Background(), "key", "value", time.Hour)
    
    // 使用一致性哈希
    hashService, _ := hamster.NewConsistentHash()
    
    // 使用分布式锁
    lockService, _ := hamster.NewDistributedLock()
}
```

## 缓存服务 (Cache)

### 创建缓存服务

```go
// 使用默认配置
cacheService, err := hamster.NewCache()

// 使用自定义配置
cacheService, err := hamster.NewCache(
    cache.WithMaxMemory(1024*1024), // 1MB
    cache.WithDefaultExpiration(time.Hour),
    cache.WithEvictionPolicy("lru"),
    cache.WithCleanupInterval(10*time.Minute),
)

// 使用配置结构体
config := &cache.Config{
    MaxMemory:         1024 * 1024,
    DefaultExpiration: time.Hour,
    EvictionPolicy:    "lru",
    CleanupInterval:   10 * time.Minute,
}
cacheService, err := cache.NewServiceWithConfig(config)
```

### 基本操作

```go
ctx := context.Background()

// 设置缓存
err := cacheService.Set(ctx, "user:123", userData, time.Hour)

// 获取缓存
value, err := cacheService.Get(ctx, "user:123")

// 删除缓存
err := cacheService.Delete(ctx, "user:123")

// 获取并删除
value, err := cacheService.LoadAndDelete(ctx, "user:123")

// 获取统计信息
stats, err := cacheService.Stats(ctx)
fmt.Printf("命中率: %.2f%%\n", stats.HitRate*100)
```

### 读透缓存

```go
// 创建读透缓存服务
readThroughCache, err := hamster.NewReadThroughCache(
    cache.WithMaxMemory(1024*1024),
)

// 使用加载器获取数据
loader := func(ctx context.Context, key string) (any, error) {
    // 从数据库或其他数据源加载数据
    return loadFromDatabase(key), nil
}

value, err := readThroughCache.GetWithLoader(ctx, "user:123", loader, time.Hour)
```

## 一致性哈希服务 (Hash)

### 创建一致性哈希服务

```go
// 使用默认配置
hashService, err := hamster.NewConsistentHash()

// 使用自定义配置
hashService, err := hamster.NewConsistentHash(
    hash.WithReplicas(150),
    hash.WithSingleflight(true),
)
```

### 节点管理

```go
ctx := context.Background()

// 添加单个节点
peer := hash.Peer{
    ID:      "server1",
    Address: "192.168.1.1:8080",
    Weight:  100,
}
err := hashService.AddPeer(ctx, peer)

// 添加多个节点
peers := []hash.Peer{
    {ID: "server1", Address: "192.168.1.1:8080", Weight: 100},
    {ID: "server2", Address: "192.168.1.2:8080", Weight: 100},
}
err := hashService.AddPeers(ctx, peers)

// 移除节点
err := hashService.RemovePeer(ctx, "server1")
```

### 节点选择

```go
// 选择单个节点
peer, err := hashService.SelectPeer(ctx, "user:123")
fmt.Printf("键 user:123 -> 节点 %s (%s)\n", peer.ID, peer.Address)

// 选择多个节点（用于副本）
peers, err := hashService.SelectPeers(ctx, "user:123", 3)

// 获取统计信息
stats, err := hashService.GetStats(ctx)
fmt.Printf("总节点: %d, 虚拟节点: %d\n", stats.TotalPeers, stats.VirtualNodes)
```

## 分布式锁服务 (Lock)

### 创建分布式锁服务

```go
// 使用默认配置
lockService, err := hamster.NewDistributedLock()

// 使用自定义配置
lockService, err := hamster.NewDistributedLock(
    lock.WithDefaultExpiration(30*time.Second),
    lock.WithDefaultTimeout(5*time.Second),
    lock.WithDefaultRetry(lock.RetryTypeExponential, 3, 100*time.Millisecond),
)
```

### 锁操作

```go
ctx := context.Background()

// 尝试获取锁（不重试）
lockInfo, err := lockService.TryLock(ctx, "resource:123")
if err != nil {
    fmt.Println("获取锁失败")
    return
}

// 获取锁（支持重试）
lockInfo, err := lockService.Lock(ctx, "resource:123")

// 使用自定义选项
options := lock.LockOptions{
    Expiration: 30 * time.Second,
    Timeout:    5 * time.Second,
    RetryType:  lock.RetryTypeExponential,
    RetryCount: 3,
    RetryBase:  100 * time.Millisecond,
}
lockInfo, err := lockService.Lock(ctx, "resource:123", options)

fmt.Printf("获取锁成功: %s\n", lockInfo.Value)
```

## 配置选项

### 缓存配置选项

- `cache.WithMaxMemory(bytes)` - 设置最大内存使用量
- `cache.WithDefaultExpiration(duration)` - 设置默认过期时间
- `cache.WithEvictionPolicy(policy)` - 设置淘汰策略 ("lru", "fifo")
- `cache.WithCleanupInterval(duration)` - 设置清理间隔
- `cache.WithBloomFilter(enable, rate)` - 启用布隆过滤器

### 一致性哈希配置选项

- `hash.WithReplicas(count)` - 设置虚拟节点数量
- `hash.WithHashFunction(fn)` - 设置自定义哈希函数
- `hash.WithSingleflight(enable)` - 启用单飞模式

### 分布式锁配置选项

- `lock.WithDefaultExpiration(duration)` - 设置默认过期时间
- `lock.WithDefaultTimeout(duration)` - 设置默认超时时间
- `lock.WithDefaultRetry(type, count, base)` - 设置默认重试策略
- `lock.WithAutoRefresh(enable, interval)` - 设置自动续约

## 版本信息

```go
version := hamster.GetVersion()
fmt.Printf("当前版本: %s\n", version)
```

## 注意事项

1. **向后兼容性**: 公共 API 一旦发布，将保持向后兼容性
2. **错误处理**: 所有方法都返回错误，请务必检查错误
3. **上下文**: 所有操作都支持 context，可用于超时控制和取消操作
4. **并发安全**: 所有服务都是并发安全的

## 示例代码

完整的示例代码请参考 `examples/` 目录：

- `examples/basic_usage.go` - 基本使用示例
- `examples/simple_demo.go` - 简单演示

## 迁移指南

如果您之前使用的是内部 API，请参考以下迁移指南：

### 从内部 API 迁移

```go
// 旧的方式 (内部 API)
repository := infraCache.NewBuildInMapCache(time.Minute)
appService := appCache.NewApplicationService(repository, nil, nil)

// 新的方式 (公共 API)
cacheService, err := hamster.NewCache(
    cache.WithCleanupInterval(time.Minute),
)
```

公共 API 提供了更简洁、更易用的接口，同时隐藏了内部实现的复杂性。
