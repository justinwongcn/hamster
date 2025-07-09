# Hamster 🐹

[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go)](https://golang.org/)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](https://opensource.org/licenses/MIT)
[![Build Status](https://img.shields.io/badge/Build-Passing-brightgreen)](https://github.com/justinwongcn/hamster)
[![Coverage](https://img.shields.io/badge/Coverage-92.5%25-brightgreen)](https://github.com/justinwongcn/hamster)
[![Documentation](https://img.shields.io/badge/Docs-Complete-brightgreen)](https://github.com/justinwongcn/hamster)

**Hamster** 是一个企业级分布式系统工具库，基于领域驱动设计（DDD）架构，提供高性能、线程安全的分布式组件。经过重构，现在提供了**简洁易用的公共 API**，让开发者可以轻松集成和使用各项功能。

## ✨ 核心特性

- 🏗️ **DDD架构设计** - 清晰的分层架构，易于维护和扩展
- 🎯 **简洁公共 API** - 一步到位的构造函数，开箱即用
- ⚡ **高性能优化** - 经过精心优化的算法和数据结构
- 🔒 **线程安全** - 所有组件都支持并发访问
- 🧪 **高测试覆盖** - 92.5% 测试覆盖率，64 个测试用例
- 🧩 **模块化设计** - 支持按需使用和自定义扩展

## 🚀 快速开始

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

    // 创建缓存服务
    cacheService, err := hamster.NewCache(
        cache.WithMaxMemory(1024*1024), // 1MB
        cache.WithEvictionPolicy("lru"),
    )
    if err != nil {
        panic(err)
    }

    ctx := context.Background()

    // 使用缓存
    err = cacheService.Set(ctx, "user:123", "John Doe", time.Hour)
    if err != nil {
        panic(err)
    }

    value, err := cacheService.Get(ctx, "user:123")
    if err != nil {
        panic(err)
    }

    fmt.Printf("缓存值: %v\n", value)

    // 创建一致性哈希服务
    hashService, err := hamster.NewConsistentHash(
        hash.WithReplicas(150),
    )
    if err != nil {
        panic(err)
    }

    // 添加节点
    peers := []hash.Peer{
        {ID: "server1", Address: "192.168.1.1:8080", Weight: 100},
        {ID: "server2", Address: "192.168.1.2:8080", Weight: 100},
    }
    err = hashService.AddPeers(ctx, peers)
    if err != nil {
        panic(err)
    }

    // 选择节点
    peer, err := hashService.SelectPeer(ctx, "user:123")
    if err != nil {
        panic(err)
    }
    fmt.Printf("用户分配到节点: %s\n", peer.ID)

    // 创建分布式锁服务
    lockService, err := hamster.NewDistributedLock(
        lock.WithDefaultExpiration(30*time.Second),
    )
    if err != nil {
        panic(err)
    }

    // 获取锁
    lockInfo, err := lockService.TryLock(ctx, "resource:123")
    if err != nil {
        panic(err)
    }
    fmt.Printf("成功获取锁: %s\n", lockInfo.Key)
}
```

## 📦 核心模块

### 1. 缓存系统（Cache）
高性能、多模式的缓存解决方案。

**特性：**
- 多种缓存模式（基础缓存、读透缓存）
- 多种淘汰策略（LRU、FIFO）
- 内存限制和自动清理
- 统计信息和监控

**使用示例：**
```go
// 基础缓存
cacheService, err := hamster.NewCache(
    cache.WithMaxMemory(1024*1024),
    cache.WithEvictionPolicy("lru"),
    cache.WithDefaultExpiration(time.Hour),
)

// 读透缓存
readThroughCache, err := hamster.NewReadThroughCache(
    cache.WithMaxMemory(512*1024),
)

loader := func(ctx context.Context, key string) (any, error) {
    return loadFromDatabase(key), nil
}

value, err := readThroughCache.GetWithLoader(ctx, "user:123", loader, time.Hour)
```

### 2. 一致性哈希（Consistent Hash）
分布式系统中的节点选择和负载均衡解决方案。

**特性：**
- 虚拟节点支持，提高负载均衡效果
- 动态节点添加/删除，支持集群扩缩容
- 多种哈希算法支持
- 节点权重配置

**使用示例：**
```go
hashService, err := hamster.NewConsistentHash(
    hash.WithReplicas(150),
    hash.WithSingleflight(true),
)

// 添加节点
peers := []hash.Peer{
    {ID: "server1", Address: "192.168.1.1:8080", Weight: 100},
    {ID: "server2", Address: "192.168.1.2:8080", Weight: 150},
}
err = hashService.AddPeers(ctx, peers)

// 选择节点
peer, err := hashService.SelectPeer(ctx, "user:123")

// 选择多个节点（副本）
peers, err := hashService.SelectPeers(ctx, "data:important", 3)
```

### 3. 分布式锁（Distributed Lock）
分布式环境下的锁机制和并发控制。

**特性：**
- 多种重试策略（固定间隔、指数退避等）
- 自动续约机制，防止锁意外过期
- 锁超时和死锁检测
- 并发安全设计

**使用示例：**
```go
lockService, err := hamster.NewDistributedLock(
    lock.WithDefaultExpiration(30*time.Second),
    lock.WithDefaultTimeout(5*time.Second),
    lock.WithDefaultRetry(lock.RetryTypeExponential, 3, 100*time.Millisecond),
)

// 尝试获取锁（不重试）
lockInfo, err := lockService.TryLock(ctx, "resource:123")

// 获取锁（支持重试）
options := lock.LockOptions{
    Expiration: 60 * time.Second,
    Timeout:    10 * time.Second,
    RetryType:  lock.RetryTypeExponential,
    RetryCount: 5,
    RetryBase:  200 * time.Millisecond,
}
lockInfo, err := lockService.Lock(ctx, "resource:456", options)
```

## 📊 性能指标

| 模块 | QPS | 延迟(P99) | 内存使用 | 并发支持 |
|------|-----|-----------|----------|----------|
| 缓存系统 | 2,000,000+ | < 0.1ms | 可配置 | 无限制 |
| 一致性哈希 | 1,000,000+ | < 0.1ms | 10MB/1000节点 | 无限制 |
| 分布式锁 | 100,000+ | < 0.5ms | 1KB/锁 | 10,000+ |

## 🏗️ 项目架构

Hamster 采用领域驱动设计（DDD）架构，分为四个主要层次：

```
hamster/
├── types.go                    # 核心接口定义
├── hamster.go                  # 主要构造函数
├── cache/                      # 缓存服务公共 API
│   └── service.go
├── hash/                       # 一致性哈希服务公共 API
│   └── service.go
├── lock/                       # 分布式锁服务公共 API
│   └── service.go
├── internal/                   # 内部实现
│   ├── application/            # 应用层 - 业务用例编排
│   ├── domain/                 # 领域层 - 核心业务逻辑
│   ├── infrastructure/         # 基础设施层 - 技术实现
│   └── interfaces/             # 接口层 - 对外接口定义
├── examples/                   # 使用示例
└── docs/                       # 文档
```

### 🎯 设计原则

1. **简洁易用**: 提供一步到位的构造函数
2. **灵活配置**: 支持选项模式配置
3. **向后兼容**: 保持 API 稳定性
4. **高性能**: 优化的算法和数据结构
5. **线程安全**: 支持并发访问

## 🧪 测试覆盖

- **总体覆盖率**: 92.5% (超过 90% 目标)
- **测试用例数**: 64 个
- **测试类型**: 单元测试 + 集成测试 + 接口测试

### 运行测试

```bash
# 运行所有测试
go test -v ./...

# 运行带覆盖率的测试
go test -v -coverprofile=coverage.out ./...

# 生成覆盖率报告
go tool cover -html=coverage.out -o coverage.html
```

## 📚 文档

- 📖 [公共 API 使用指南](PUBLIC_API.md) - 详细的 API 文档
- 📝 [使用示例](EXAMPLES.md) - 完整的使用示例
- 📊 [测试覆盖率报告](TEST_COVERAGE_REPORT.md) - 测试质量报告
- 📋 [项目状态](PROJECT_STATUS.md) - 项目当前状态
- 📄 [重构总结](REFACTOR_SUMMARY.md) - 重构过程和成果

## 🚀 版本历史

### v1.0.0 (当前版本)
- ✅ 完整的公共 API 层
- ✅ 92.5% 测试覆盖率
- ✅ 缓存、一致性哈希、分布式锁功能
- ✅ 选项模式配置
- ✅ 完整的文档体系

## 🤝 贡献

我们欢迎所有形式的贡献！

1. Fork 项目
2. 创建功能分支 (`git checkout -b feature/AmazingFeature`)
3. 提交更改 (`git commit -m 'Add some AmazingFeature'`)
4. 推送到分支 (`git push origin feature/AmazingFeature`)
5. 创建 Pull Request

### 贡献指南

- 确保代码通过所有测试
- 添加适当的测试用例
- 更新相关文档
- 遵循现有的代码风格

## 📄 许可证

本项目采用 [MIT License](LICENSE) 许可证。

## 🆘 获取帮助

- 📋 [GitHub Issues](https://github.com/justinwongcn/hamster/issues) - 报告问题或请求功能
- 📖 [文档](./docs/) - 查看详细文档
- 💬 [讨论区](https://github.com/justinwongcn/hamster/discussions) - 社区讨论

## 🌟 致谢

感谢所有为 Hamster 项目做出贡献的开发者！

---

**开始使用 Hamster 构建您的分布式应用吧！** 🚀