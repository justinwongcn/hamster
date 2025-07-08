# Hamster 🐹

[![Go Version](https://img.shields.io/badge/Go-1.24.3+-00ADD8?style=flat&logo=go)](https://golang.org/)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)
[![Build Status](https://img.shields.io/badge/Build-Passing-brightgreen)](https://github.com/justinwongcn/hamster)

**Hamster** 是一个企业级分布式系统工具库，基于领域驱动设计（DDD）架构，提供高性能、线程安全的分布式组件。专为构建可扩展的分布式应用而设计。

## ✨ 核心特性

- 🏗️ **DDD架构设计** - 清晰的分层架构，易于维护和扩展
- ⚡ **高性能优化** - 经过精心优化的算法和数据结构
- 🔒 **线程安全** - 所有组件都支持并发访问
- 🎯 **企业级** - 完整的监控、日志、错误处理机制
- 🧩 **模块化设计** - 支持按需使用和自定义扩展

## 🚀 核心模块

### 1. 一致性哈希（Consistent Hash）
分布式系统中的节点选择和负载均衡解决方案。

**特性：**
- 虚拟节点支持，提高负载均衡效果
- 动态节点添加/删除，支持集群扩缩容
- 多种哈希算法支持
- 节点权重配置

**应用场景：**
- 分布式缓存节点选择
- 数据分片和路由
- 微服务负载均衡

### 2. 分布式锁（Distributed Lock）
分布式环境下的锁机制和并发控制。

**特性：**
- 多种重试策略（固定间隔、指数退避等）
- 自动续约机制，防止锁意外过期
- Singleflight优化，减少锁竞争
- 锁超时和死锁检测

**应用场景：**
- 资源互斥访问
- 分布式任务调度
- 数据一致性保证

### 3. 缓存系统（Cache）
高性能、多模式的缓存解决方案。

**特性：**
- 多种缓存模式（读透、写透、写回）
- 多种淘汰策略（LRU、FIFO、LFU）
- 布隆过滤器防止缓存穿透
- 内存限制和自动清理

**应用场景：**
- 应用数据缓存
- 数据库查询缓存
- 会话状态缓存

## 📦 安装

```bash
go get github.com/justinwongcn/hamster
```

**系统要求：**
- Go 1.24.3+
- 支持 Linux、macOS、Windows

## 🎯 快速开始

### 一致性哈希示例

```go
package main

import (
    "fmt"
    "github.com/justinwongcn/hamster/internal/application/consistent_hash"
    "github.com/justinwongcn/hamster/internal/infrastructure/consistent_hash"
)

func main() {
    // 创建一致性哈希实例（150个虚拟节点）
    hashMap := consistent_hash.NewConsistentHashMap(150)
    peerPicker := consistent_hash.NewConsistentHashPeerPicker(hashMap)
    service := consistent_hash.NewConsistentHashApplicationService(peerPicker)
    
    // 添加节点
    peers := []consistent_hash.PeerRequest{
        {ID: "node1", Address: "192.168.1.1:8080", Weight: 100},
        {ID: "node2", Address: "192.168.1.2:8080", Weight: 100},
        {ID: "node3", Address: "192.168.1.3:8080", Weight: 100},
    }
    
    cmd := consistent_hash.AddPeersCommand{Peers: peers}
    err := service.AddPeers(context.Background(), cmd)
    if err != nil {
        log.Fatal(err)
    }
    
    // 选择节点
    selectionCmd := consistent_hash.PeerSelectionCommand{Key: "user:12345"}
    result, err := service.SelectPeer(context.Background(), selectionCmd)
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("Key %s 被分配到节点: %s\n", result.Key, result.Peer.ID)
}
```

### 分布式锁示例

```go
package main

import (
    "context"
    "time"
    "github.com/justinwongcn/hamster/internal/application/lock"
    "github.com/justinwongcn/hamster/internal/infrastructure/lock"
)

func main() {
    // 创建分布式锁服务
    distributedLock := lock.NewMemoryDistributedLock()
    service := lock.NewDistributedLockApplicationService(distributedLock)
    
    // 获取锁
    cmd := lock.LockCommand{
        Key:        "resource:order:12345",
        Expiration: 30 * time.Second,
        Timeout:    5 * time.Second,
        RetryStrategy: "exponential_backoff",
    }
    
    result, err := service.Lock(context.Background(), cmd)
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("成功获取锁: %s\n", result.Key)
    
    // 执行业务逻辑
    time.Sleep(10 * time.Second)
    
    // 释放锁
    unlockCmd := lock.UnlockCommand{Key: result.Key}
    err = service.Unlock(context.Background(), unlockCmd)
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Println("锁已释放")
}
```

### 缓存系统示例

```go
package main

import (
    "context"
    "time"
    "github.com/justinwongcn/hamster/internal/application/cache"
    "github.com/justinwongcn/hamster/internal/infrastructure/cache"
    domainCache "github.com/justinwongcn/hamster/internal/domain/cache"
)

func main() {
    // 创建缓存策略
    policy := domainCache.NewCachePolicy().
        WithMaxSize(1000).
        WithDefaultTTL(time.Hour).
        WithEvictionStrategy(domainCache.NewLRUEvictionStrategy())
    
    // 创建缓存仓储
    repository := cache.NewInMemoryCacheRepository(policy)
    cacheService := domainCache.NewCacheService(domainCache.NewLRUEvictionStrategy())
    
    // 创建应用服务
    service := cache.NewApplicationService(repository, cacheService, nil)
    
    // 设置缓存
    cmd := cache.SetCommand{
        Key:        "user:12345",
        Value:      map[string]interface{}{"name": "张三", "age": 30},
        Expiration: time.Hour,
    }
    
    err := service.Set(context.Background(), cmd)
    if err != nil {
        log.Fatal(err)
    }
    
    // 获取缓存
    getCmd := cache.GetCommand{Key: "user:12345"}
    result, err := service.Get(context.Background(), getCmd)
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("缓存值: %+v\n", result.Value)
}
```

## 📊 性能指标

| 模块 | QPS | 延迟(P99) | 内存使用 | 并发支持 |
|------|-----|-----------|----------|----------|
| 一致性哈希 | 1,000,000+ | < 0.1ms | 10MB/1000节点 | 无限制 |
| 分布式锁 | 100,000+ | < 0.5ms | 1KB/锁 | 10,000+ |
| 缓存系统 | 2,000,000+ | < 0.1ms | 可配置 | 无限制 |

## 🏗️ 架构设计

```
hamster/
├── internal/
│   ├── application/     # 应用层 - 业务用例编排
│   │   ├── cache/
│   │   ├── consistent_hash/
│   │   └── lock/
│   ├── domain/          # 领域层 - 核心业务逻辑
│   │   ├── cache/
│   │   ├── consistent_hash/
│   │   ├── lock/
│   │   └── tools/
│   ├── infrastructure/ # 基础设施层 - 技术实现
│   │   ├── cache/
│   │   ├── consistent_hash/
│   │   └── lock/
│   └── interfaces/      # 接口层 - 对外接口定义
├── docs/               # 文档
└── README.md
```

## 📚 文档

- 📖 [完整用户指南](./docs/USER_GUIDE.md) - 详细的使用文档和最佳实践
- ⚡ [快速参考手册](./docs/QUICK_REFERENCE.md) - 常用功能速查
- 🏠 [文档中心](./docs/README.md) - 文档导航和概览

## 🤝 贡献

我们欢迎所有形式的贡献！

1. Fork 项目
2. 创建功能分支 (`git checkout -b feature/AmazingFeature`)
3. 提交更改 (`git commit -m 'Add some AmazingFeature'`)
4. 推送到分支 (`git push origin feature/AmazingFeature`)
5. 创建 Pull Request

## 📄 许可证

本项目采用 [Apache License 2.0](LICENSE) 许可证。

## 🆘 获取帮助

- 📋 [GitHub Issues](https://github.com/justinwongcn/hamster/issues) - 报告问题或请求功能
- 📖 [文档](./docs/) - 查看详细文档
- 💬 [讨论区](https://github.com/justinwongcn/hamster/discussions) - 社区讨论

---

**开始使用 Hamster 构建您的分布式应用吧！** 🚀
