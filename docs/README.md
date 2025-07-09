# Hamster 文档中心

欢迎使用 Hamster 企业级分布式系统工具库！本文档中心提供了完整的使用指南和参考资料。

## 📚 文档结构

### 🚀 [用户使用文档](./USER_GUIDE.md)
**完整的企业级用户使用文档**

包含内容：
- 项目概述和架构设计
- 详细的安装和配置指南
- 三大核心模块完整使用说明
- API参考文档
- 最佳实践和性能调优
- 故障排除和监控指南

**适用人群**：开发工程师、架构师、运维工程师

### ⚡ [快速参考手册](./QUICK_REFERENCE.md)
**精简的快速查阅文档**

包含内容：
- 快速安装和基本使用
- 常用配置参数速查表
- 常见错误处理方法
- 最佳实践要点
- 故障排除速查表

**适用人群**：有经验的开发者、快速查阅需求

## 🎯 核心功能模块

### 1. 一致性哈希（Consistent Hash）
- **功能**：分布式系统节点选择和负载均衡
- **特性**：虚拟节点、动态扩缩容、负载均衡
- **应用场景**：分布式缓存、数据分片、微服务路由

### 2. 分布式锁（Distributed Lock）
- **功能**：分布式环境下的锁机制和并发控制
- **特性**：多种重试策略、自动续约、Singleflight优化
- **应用场景**：资源互斥、任务调度、数据一致性

### 3. 缓存系统（Cache）
- **功能**：高性能缓存解决方案
- **特性**：多种缓存模式、淘汰策略、布隆过滤器
- **应用场景**：数据缓存、性能优化、减少DB压力

## 🏗️ 架构特点

- **DDD架构**：领域驱动设计，代码结构清晰
- **高性能**：经过优化的算法和数据结构
- **线程安全**：所有组件都支持并发访问
- **易扩展**：模块化设计，支持自定义扩展
- **企业级**：完整的监控、日志、错误处理

## 📖 如何使用本文档

### 新手用户
1. 先阅读 [用户使用文档](./USER_GUIDE.md) 的"项目概述"和"快速开始"章节
2. 根据需要选择相应的核心模块进行学习
3. 参考"最佳实践"章节进行实际开发

### 有经验用户
1. 直接查阅 [快速参考手册](./QUICK_REFERENCE.md) 获取关键信息
2. 需要详细信息时参考 [用户使用文档](./USER_GUIDE.md) 的相应章节
3. 遇到问题时查看"故障排除"章节

### 运维人员
1. 重点关注"性能调优"和"监控指标"章节
2. 熟悉"故障排除"和"日志监控"内容
3. 了解各模块的配置参数和性能特征

## 🔧 快速开始

### 安装
```bash
go get github.com/justinwongcn/hamster
```

### 基本使用
```go
import (
    "github.com/justinwongcn/hamster/internal/application/consistent_hash"
    "github.com/justinwongcn/hamster/internal/application/lock"
    "github.com/justinwongcn/hamster/internal/application/cache"
)

// 一致性哈希
peerPicker := consistent_hash.NewConsistentHashPeerPicker(150)
hashService := consistent_hash.NewConsistentHashApplicationService(peerPicker)

// 分布式锁
distributedLock := lock.NewMemoryDistributedLock()
lockService := lock.NewDistributedLockApplicationService(distributedLock)

// 缓存系统
policy := cache.NewCachePolicy()
repository := cache.NewInMemoryCacheRepository(policy)
cacheService := cache.NewApplicationService(repository, nil, nil)
```

## 📊 性能指标

| 模块 | QPS | 延迟(P99) | 内存使用 |
|------|-----|-----------|----------|
| 一致性哈希 | 1,000,000 | 0.1ms | 10MB/1000节点 |
| 分布式锁 | 100,000 | 0.5ms | 1KB/锁 |
| 缓存系统 | 2,000,000 | 0.1ms | 可配置 |

## 🆘 获取帮助

### 文档问题
- 如果文档内容不清楚或有错误，请提交 Issue
- 建议改进文档结构或内容，欢迎提交 PR

### 技术问题
- 查看 [故障排除](./USER_GUIDE.md#故障排除) 章节
- 搜索已有的 GitHub Issues
- 提交新的 Issue，请包含：
  - 详细的问题描述
  - 完整的错误日志
  - 相关的配置信息
  - 复现步骤

### 功能建议
- 提交 Feature Request Issue
- 描述具体的使用场景和需求
- 说明期望的功能行为

## 📝 文档更新

本文档会随着项目版本更新而持续维护：

- **主要版本更新**：会更新架构设计和API接口文档
- **次要版本更新**：会更新功能说明和使用示例
- **补丁版本更新**：会更新故障排除和最佳实践

## 🤝 贡献指南

欢迎为文档贡献内容：

1. **改进现有文档**：修正错误、补充说明、优化结构
2. **添加使用示例**：提供更多实际场景的使用案例
3. **翻译文档**：支持多语言版本
4. **完善故障排除**：添加更多常见问题和解决方案

### 贡献流程
1. Fork 项目仓库
2. 创建功能分支
3. 修改文档内容
4. 提交 Pull Request
5. 等待代码审查

---

**开始探索 Hamster 的强大功能吧！** 🎉

如有任何问题，请随时通过 GitHub Issues 联系我们。
