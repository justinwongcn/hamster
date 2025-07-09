# Hamster 分布式缓存系统

![License](https://img.shields.io/badge/license-MIT-blue.svg)
![Go Version](https://img.shields.io/badge/go-%3E%3D1.18-blue.svg)
![Build Status](https://img.shields.io/badge/build-passing-brightgreen.svg)
![Coverage](https://img.shields.io/badge/coverage-90%25-brightgreen.svg)
![Documentation](https://img.shields.io/badge/docs-complete-brightgreen.svg)

Hamster 是一个**企业级、生产就绪**的分布式缓存系统，基于领域驱动设计（DDD）构建。项目提供了完整的缓存解决方案，包括11种缓存实现、3种淘汰策略、分布式锁、一致性哈希、布隆过滤器等核心组件，并配备了全面的技术文档体系，适用于各种高性能和高可用性应用场景。

## ✨ 项目亮点

- 🏗️ **完整的DDD架构**: 四层清晰分离，易于扩展和维护
- 📚 **全面的文档体系**: 35个详细技术文档，从理论到实践
- 🚀 **生产级代码质量**: 90%+测试覆盖率，性能基准测试
- 🔧 **丰富的组件库**: 11种缓存实现，满足各种使用场景
- ⚡ **高性能设计**: 并发优化，内存管理，算法优化
- 🛡️ **企业级特性**: 监控告警，故障处理，扩展指南

## 📋 核心特性

### 🗄️ 缓存系统
- **11种缓存实现**: MaxMemoryCache、BuildInMapCache、ReadThrough、WriteThrough、WriteBack等
- **3种淘汰策略**: LRU、FIFO、Random，支持自定义策略
- **布隆过滤器**: 防止缓存穿透，支持动态配置和统计监控
- **内存管理**: 智能内存限制，自动过期清理，对象池优化

### 🔒 分布式锁
- **多种重试策略**: 固定间隔、指数退避、线性退避
- **自动续约**: 防止锁过期，支持配置续约间隔
- **SingleFlight优化**: 本地并发控制，减少分布式锁压力
- **故障恢复**: 锁超时处理，异常情况恢复

### 🔄 一致性哈希
- **虚拟节点**: 提高负载均衡效果，支持自定义虚拟节点数
- **动态扩缩容**: 支持节点的动态添加和移除
- **故障转移**: 自动检测节点故障并选择替代节点
- **负载统计**: 详细的负载分布统计和监控

### 🛠️ 工具组件
- **ID生成器**: 雪花算法、UUID生成器，支持分布式唯一ID
- **错误管理**: 统一的错误类型定义和处理机制
- **性能监控**: 内置指标收集，支持自定义监控回调

## 🚀 快速开始

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
    // 创建最大内存缓存，支持LRU淘汰策略
    lruPolicy := cache.NewLRUPolicy(100) // 容量100
    memoryCache := cache.NewMaxMemoryCache(1*1024*1024, // 1MB内存限制
        cache.MaxMemoryCacheWithEvictionPolicy(lruPolicy))

    ctx := context.Background()

    // 设置缓存
    user := map[string]interface{}{
        "id":   "123",
        "name": "John Doe",
        "age":  30,
    }
    err := memoryCache.Set(ctx, "user:123", user, time.Hour)
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

    // 获取缓存统计
    stats := memoryCache.GetStats()
    fmt.Printf("缓存命中率: %.2f%%\n", stats.HitRate()*100)
}
```

### 高级缓存模式

```go
// 1. 读透缓存 - 自动加载数据
readThrough := &cache.ReadThroughCache{
    Repository: memoryCache,
    LoadFunc: func(ctx context.Context, key string) (any, error) {
        // 从数据库加载数据
        return database.LoadUser(key)
    },
    Expiration: time.Hour,
}

value, err := readThrough.Get(ctx, "user:456")

// 2. 写透缓存 - 同时写入缓存和存储
writeThrough := &cache.WriteThroughCache{
    Repository: memoryCache,
    StoreFunc: func(ctx context.Context, key string, val any) error {
        return database.SaveUser(key, val)
    },
}

err = writeThrough.Set(ctx, "user:789", userData, time.Hour)

// 3. 写回缓存 - 异步批量刷新
writeBack := cache.NewWriteBackCache(memoryCache, time.Minute, 100)
err = writeBack.SetDirty(ctx, "user:999", userData, time.Hour)
```

### 分布式锁使用

```go
import "github.com/justinwongcn/hamster/internal/infrastructure/lock"

// 创建分布式锁
distributedLock := lock.NewMemoryDistributedLock()

// 创建指数退避重试策略
retryStrategy := lock.NewExponentialBackoffRetryStrategy(
    100*time.Millisecond, // 初始间隔
    2.0,                  // 倍数
    5,                    // 最大重试次数
)

// 获取锁
lockInstance, err := distributedLock.Lock(ctx, "resource:123",
    time.Minute, 5*time.Second, retryStrategy)
if err != nil {
    fmt.Printf("获取锁失败: %v\n", err)
    return
}
defer lockInstance.Unlock(ctx)

// 启动自动续约
go lockInstance.AutoRefresh(30*time.Second, 5*time.Second)

// 执行需要锁保护的业务逻辑
fmt.Println("执行关键业务逻辑...")
```

### 一致性哈希使用

```go
import "github.com/justinwongcn/hamster/internal/infrastructure/consistent_hash"

// 创建一致性哈希映射，150个虚拟节点
hashMap := consistent_hash.NewConsistentHashMap(150, nil)

// 创建节点选择器
picker := consistent_hash.NewSingleflightPeerPicker(hashMap)

// 创建节点
peer1, _ := domain.NewPeerInfo("server1", "192.168.1.1:8080", 100)
peer2, _ := domain.NewPeerInfo("server2", "192.168.1.2:8080", 150) // 更高权重
peer3, _ := domain.NewPeerInfo("server3", "192.168.1.3:8080", 100)

// 添加节点
picker.AddPeers(peer1, peer2, peer3)

// 选择节点
peer, err := picker.PickPeer("user:123")
if err != nil {
    fmt.Printf("选择节点失败: %v\n", err)
    return
}

fmt.Printf("用户分配到服务器: %s (%s)\n", peer.ID(), peer.Address())

// 选择多个节点（用于数据副本）
peers, err := picker.PickPeers("data:important", 3)
if err == nil {
    fmt.Printf("数据副本节点: ")
    for _, p := range peers {
        fmt.Printf("%s ", p.ID())
    }
    fmt.Println()
}
```

### 布隆过滤器使用

```go
import "github.com/justinwongcn/hamster/internal/domain/cache"

// 创建布隆过滤器配置
config, err := cache.NewBloomFilterConfig(10000, 0.01) // 10000个元素，1%假阳性率
if err != nil {
    fmt.Printf("创建配置失败: %v\n", err)
    return
}

// 创建内存布隆过滤器
bloomFilter := cache.NewInMemoryBloomFilter(config)

// 创建布隆过滤器缓存
loader := func(ctx context.Context, key string) (any, error) {
    return database.LoadData(key)
}

bloomCache := cache.NewBloomFilterCacheSimple(memoryCache, bloomFilter, loader)

// 使用布隆过滤器缓存
value, err := bloomCache.Get(ctx, "user:123")
if err != nil {
    fmt.Printf("获取数据失败: %v\n", err)
    return
}

// 获取布隆过滤器统计
stats := bloomFilter.GetStats()
fmt.Printf("假阳性率: %.4f, 已添加元素: %d\n",
    stats.EstimatedFalsePositiveRate(), stats.AddedElements())
```

## 🏗️ 项目架构

Hamster 采用领域驱动设计（DDD）架构，分为四个主要层次，确保代码的可维护性和可扩展性：

```
┌─────────────────────────────────────┐
│           Interfaces Layer          │  接口层 - 统一抽象
├─────────────────────────────────────┤
│          Application Layer          │  应用层 - 业务用例
├─────────────────────────────────────┤
│         Infrastructure Layer        │  基础设施层 - 具体实现
├─────────────────────────────────────┤
│            Domain Layer             │  领域层 - 核心业务
└─────────────────────────────────────┘
```

### 🎯 领域层 (Domain Layer)
**核心业务逻辑和领域模型**
- **缓存领域**: 缓存服务、淘汰策略、布隆过滤器
- **锁领域**: 分布式锁模型、重试策略、生命周期管理
- **哈希领域**: 一致性哈希算法、节点管理、负载均衡
- **工具领域**: ID生成器（雪花算法、UUID）
- **错误领域**: 统一的错误类型定义和处理

### 🔧 基础设施层 (Infrastructure Layer)
**技术实现和外部依赖**
- **缓存实现**: 11种缓存实现（MaxMemory、BuildInMap、ReadThrough等）
- **淘汰策略**: LRU、FIFO、Random策略的具体实现
- **锁实现**: 内存分布式锁、重试机制、自动续约
- **哈希实现**: 一致性哈希映射、SingleFlight节点选择器

### 🚀 应用层 (Application Layer)
**业务用例协调和数据转换**
- **缓存应用服务**: 协调缓存相关的业务用例，支持多种缓存模式
- **锁应用服务**: 协调分布式锁的业务用例，支持多种重试策略
- **哈希应用服务**: 协调一致性哈希的业务用例，支持动态扩缩容

### 📡 接口层 (Interfaces Layer)
**统一的抽象定义**
- **Cache接口**: 统一的缓存操作抽象
- **类型定义**: 通用的数据类型和契约
- **API规范**: 对外暴露的接口标准

## 📚 文档体系

Hamster 项目提供了完整的三层文档体系，共35个详细技术文档：

### 📖 文档结构
```
文档体系
├── 📄 项目级文档 (1个)
│   └── README.md - 项目总览和快速开始
├── 📁 层级文档 (4个)
│   ├── domain/README.md - 领域层概览
│   ├── infrastructure/README.md - 基础设施层概览
│   ├── application/README.md - 应用层概览
│   └── docs/ - 项目文档集合
├── 📁 包级文档 (6个)
│   ├── tools/README.md - ID生成工具包
│   ├── errs/README.md - 错误定义包
│   ├── cache/README.md - 缓存包概览
│   └── consistent_hash/README.md - 一致性哈希包
└── 📄 组件级文档 (24个)
    ├── 缓存组件 (11个) - 各种缓存实现的详细文档
    ├── 锁组件 (3个) - 分布式锁相关文档
    ├── 哈希组件 (3个) - 一致性哈希相关文档
    ├── 工具组件 (2个) - ID生成器文档
    ├── 错误组件 (2个) - 错误处理文档
    ├── 应用服务 (3个) - 应用层服务文档
    └── 接口定义 (1个) - 接口类型文档
```

### 📋 文档特点
- **中文文档**: 便于中文开发者理解
- **完整覆盖**: 从理论到实践的全方位指导
- **实用示例**: 大量可运行的代码示例
- **最佳实践**: 生产环境的使用建议
- **性能指导**: 详细的性能优化和监控策略

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
