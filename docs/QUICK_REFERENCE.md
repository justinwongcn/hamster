# Hamster 快速参考手册

## 快速安装

```bash
go get github.com/justinwongcn/hamster
```

## 一致性哈希 - 快速开始

```go
// 创建服务
peerPicker := consistent_hash.NewConsistentHashPeerPicker(150)
service := consistent_hash.NewConsistentHashApplicationService(peerPicker)

// 添加节点
cmd := consistent_hash.AddPeersCommand{
    Peers: []consistent_hash.PeerRequest{
        {ID: "node1", Address: "192.168.1.1:8080", Weight: 100},
        {ID: "node2", Address: "192.168.1.2:8080", Weight: 100},
    },
}
service.AddPeers(ctx, cmd)

// 选择节点
result, err := service.SelectPeer(ctx, consistent_hash.PeerSelectionCommand{
    Key: "user:12345",
})
```

## 分布式锁 - 快速开始

```go
// 创建服务
distributedLock := lock.NewMemoryDistributedLock()
service := lock.NewDistributedLockApplicationService(distributedLock)

// 获取锁
lockCmd := lock.LockCommand{
    Key:        "resource:123",
    Expiration: 30 * time.Second,
    Timeout:    5 * time.Second,
    RetryType:  "exponential",
    RetryCount: 3,
    RetryBase:  100 * time.Millisecond,
}

result, err := service.Lock(ctx, lockCmd)
if err != nil {
    log.Fatal(err)
}

// 释放锁
defer service.Unlock(ctx, lock.UnlockCommand{Key: "resource:123"})
```

## 缓存系统 - 快速开始

```go
// 创建缓存策略
policy := cache.NewCachePolicy().
    WithMaxSize(1000).
    WithMaxMemory(100 * 1024 * 1024). // 100MB
    WithDefaultTTL(time.Hour).
    WithEvictionStrategy(cache.NewLRUEvictionStrategy())

// 创建服务
repository := cache.NewInMemoryCacheRepository(policy)
cacheService := cache.NewCacheService(cache.NewLRUEvictionStrategy())
appService := cache.NewApplicationService(repository, cacheService, nil)

// 设置缓存
setCmd := cache.CacheItemCommand{
    Key:        "user:12345",
    Value:      map[string]interface{}{"name": "张三", "age": 30},
    Expiration: time.Hour,
}
appService.Set(ctx, setCmd)

// 获取缓存
getQuery := cache.CacheItemQuery{Key: "user:12345"}
result, err := appService.Get(ctx, getQuery)
```

## 常用配置参数

### 一致性哈希

| 参数 | 推荐值 | 说明 |
|------|--------|------|
| replicas | 150 | 虚拟节点数，节点少时可增加到300 |
| hashFunc | crc32.ChecksumIEEE | 默认哈希函数 |

### 分布式锁

| 参数 | 推荐值 | 说明 |
|------|--------|------|
| expiration | 30s | 锁过期时间 |
| timeout | 5s | 获取锁超时 |
| retryType | "exponential" | 重试策略 |
| retryCount | 3-5 | 重试次数 |

### 缓存系统

| 参数 | 推荐值 | 说明 |
|------|--------|------|
| maxSize | 1000-10000 | 最大条目数 |
| maxMemory | 100MB-1GB | 最大内存 |
| defaultTTL | 1h | 默认过期时间 |
| evictionStrategy | LRU | 淘汰策略 |

## 常见错误处理

```go
// 一致性哈希错误
if err == domain.ErrNoPeersAvailable {
    // 没有可用节点，需要添加节点
}

// 分布式锁错误
if err == domain.ErrFailedToPreemptLock {
    // 抢锁失败，可以重试或等待
}
if err == domain.ErrLockExpired {
    // 锁已过期，需要重新获取
}

// 缓存错误
if err == domain.ErrKeyNotFound {
    // 键未找到，正常情况
}
```

## 最佳实践要点

### 一致性哈希
- 根据节点数量调整虚拟节点数
- 设置合理的节点权重
- 定期检查健康状态

### 分布式锁
- 使用细粒度锁，避免粗粒度锁
- 设置合理的过期时间
- 使用defer确保锁释放
- 长任务使用自动续约

### 缓存系统
- 选择合适的缓存模式（读透/写回）
- 设计有意义的缓存键
- 使用布隆过滤器优化
- 监控内存使用率

## 性能调优要点

1. **虚拟节点数量**：节点少时增加，节点多时减少
2. **重试策略**：高并发用指数退避，低并发用固定间隔
3. **缓存大小**：根据系统内存合理设置
4. **批量操作**：减少锁竞争，提高性能

## 监控指标

### 关键指标
- 一致性哈希：负载均衡度、选择延迟
- 分布式锁：获取延迟、竞争率
- 缓存系统：命中率、内存使用率

### 告警阈值
- 负载均衡度 < 80%
- 锁获取延迟 > 100ms
- 缓存命中率 < 90%
- 内存使用率 > 90%

## 故障排除速查

| 问题 | 症状 | 解决方案 |
|------|------|----------|
| 负载不均衡 | 某些节点过载 | 增加虚拟节点数 |
| 死锁 | 程序hang住 | 检查锁释放逻辑 |
| 缓存击穿 | 大量DB请求 | 使用singleflight |
| 内存泄漏 | 内存持续增长 | 检查淘汰策略 |

## 联系支持

- 文档：[完整用户指南](./USER_GUIDE.md)
- 问题反馈：GitHub Issues
- 技术支持：请提供详细的错误日志和配置信息
