# Hamster 分布式缓存系统

![License](https://img.shields.io/badge/license-MIT-blue.svg)
![Go Version](https://img.shields.io/badge/go-%3E%3D1.18-blue.svg)
![Build Status](https://img.shields.io/badge/build-passing-brightgreen.svg)

Hamster 是一个高性能、可扩展的分布式缓存系统，基于领域驱动设计（DDD）构建，提供了缓存、分布式锁、一致性哈希等核心功能，支持多种缓存模式和布隆过滤器优化，适用于各种高性能和高可用性应用场景。

## 📋 核心特性

- **多种缓存模式**：支持读透、写透、写回等多种缓存模式
- **分布式锁**：提供高性能分布式锁机制，支持自动续约和多种重试策略
- **一致性哈希**：实现高效的数据分片和负载均衡
- **布隆过滤器**：防止缓存穿透，优化读取性能
- **内存管理**：支持内存限制和LRU淘汰策略
- **领域驱动设计**：清晰的分层架构，便于扩展和维护
- **并发优化**：线程安全实现，适合高并发场景
- **可观测性**：内置性能指标监控

## 🔧 快速开始

### 安装

```bash
go get -u github.com/justinwongcn/hamster
```

### 基础缓存使用

```go
package main

import (
    "context"
    "fmt"
    "time"

    "github.com/justinwongcn/hamster/internal/infrastructure/cache"
)

func main() {
    // 创建一个内存缓存实例，最大内存限制为1MB
    memoryCache := cache.NewMaxMemoryCache(1 * 1024 * 1024)

    ctx := context.Background()

    // 设置缓存
    err := memoryCache.Set(ctx, "user:123", "John Doe", time.Hour)
    if err != nil {
        fmt.Printf("设置缓存失败: %v\n", err)
        return
    }

    // 获取缓存
    value, err := memoryCache.Get(ctx, "user:123")
    if err != nil {
        fmt.Printf("获取缓存失败: %v\n", err)
        return
    }

    fmt.Printf("用户信息: %v\n", value)

    // 删除缓存
    err = memoryCache.Delete(ctx, "user:123")
    if err != nil {
        fmt.Printf("删除缓存失败: %v\n", err)
        return
    }
}
```

### 读透缓存使用

```go
// 创建读透缓存
readThrough := cache.NewReadThroughCache(memoryCache)

// 定义数据加载器函数
loader := func(ctx context.Context, key string) (any, error) {
    // 从数据库或其他数据源加载数据
    return fmt.Sprintf("Database value for %s", key), nil
}

// 使用读透缓存获取数据
value, err := readThrough.GetWithLoader(ctx, "user:456", loader, time.Hour)
if err != nil {
    fmt.Printf("获取数据失败: %v\n", err)
    return
}

fmt.Printf("用户信息: %v\n", value)
```

### 分布式锁使用

```go
// 创建分布式锁管理器
lockManager := lock.NewMemoryDistributedLock()

// 创建重试策略
retryStrategy := lock.NewExponentialBackoffRetryStrategy(100*time.Millisecond, 2.0, 5)

// 获取锁
lock, err := lockManager.Lock(ctx, "resource:123", time.Minute, 5*time.Second, retryStrategy)
if err != nil {
    fmt.Printf("获取锁失败: %v\n", err)
    return
}

// 自动续约
go func() {
    _ = lock.AutoRefresh(30*time.Second, 5*time.Second)
}()

// 使用资源...

// 释放锁
defer lock.Unlock(ctx)
```

### 一致性哈希使用

```go
// 创建一致性哈希映射
hashMap := consistent_hash.NewConsistentHashMap(150, nil)

// 添加节点
hashMap.Add("server1", "server2", "server3")

// 获取节点
server, err := hashMap.Get("user:123")
if err != nil {
    fmt.Printf("获取节点失败: %v\n", err)
    return
}

fmt.Printf("用户分配到服务器: %s\n", server)
```

### 布隆过滤器使用

```go
// 创建布隆过滤器配置
config, err := cache.NewBloomFilterConfig(1000, 0.01)
if err != nil {
    fmt.Printf("创建布隆过滤器配置失败: %v\n", err)
    return
}

// 创建内存布隆过滤器
bloomFilter := cache.NewInMemoryBloomFilter(config)

// 创建带布隆过滤器的缓存
bloomCache := cache.NewBloomFilterCacheSimple(memoryCache, bloomFilter, loader)

// 添加元素
err = bloomFilter.Add(ctx, "user:123")
if err != nil {
    fmt.Printf("添加元素失败: %v\n", err)
    return
}

// 检查元素
exists := bloomFilter.HasKey(ctx, "user:123")
fmt.Printf("元素存在: %v\n", exists)
```

## 🏗️ 项目架构

Hamster 采用领域驱动设计（DDD）架构，分为四个主要层次：

### 领域层 (Domain)

领域层是系统的核心，包含业务逻辑、实体、值对象和领域服务：
- 不依赖外部框架或基础设施
- 通过接口定义与外部的交互契约
- 包含所有业务规则和验证逻辑

### 应用层 (Application)

应用层协调领域对象和基础设施，实现具体的业务用例：
- 处理事务管理、权限控制
- 数据转换（DTO）
- 协调多个领域服务

### 基础设施层 (Infrastructure)

基础设施层提供领域层接口的具体实现：
- 实现领域层定义的接口
- 处理外部系统交互
- 提供技术服务（内存管理、网络通信等）

### 接口层 (Interface)

接口层定义了系统与外部交互的标准：
- 提供统一的接口类型
- 保证不同实现之间的兼容性
- 简化接口契约

## 📈 性能优化

Hamster 实现了多种性能优化策略：

1. **内存管理优化**
   - 合理的内存限制设置
   - 对象大小优化
   - 使用对象池减少GC压力

2. **缓存策略优化**
   - 多种缓存模式适应不同场景
   - 优化过期时间设置
   - 布隆过滤器防止缓存穿透

3. **并发优化**
   - 分段锁减少竞争
   - 使用读写锁优化读操作
   - 原子操作避免竞态条件

4. **一致性哈希优化**
   - 虚拟节点数量优化
   - 高性能哈希函数
   - 节点选择优化

## 🤝 如何贡献

1. Fork 项目
2. 创建特性分支 (`git checkout -b feature/amazing-feature`)
3. 提交您的更改 (`git commit -m 'Add some amazing feature'`)
4. 推送到分支 (`git push origin feature/amazing-feature`)
5. 创建 Pull Request

## 📄 许可证

该项目基于 MIT 许可证 - 详见 [LICENSE](LICENSE) 文件

## 📊 性能指标

Hamster 关注以下关键性能指标：

1. **吞吐量 (Throughput)**
   - 每秒处理的请求数 (QPS)
   - 缓存命中率 (Hit Rate)
   - 数据传输速率 (MB/s)

2. **延迟 (Latency)**
   - 平均响应时间
   - P95/P99 响应时间
   - 锁获取时间

3. **资源使用率**
   - 内存使用率
   - CPU 使用率
   - 网络带宽使用率

4. **可用性指标**
   - 系统正常运行时间
   - 错误率
   - 恢复时间
