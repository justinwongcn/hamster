# Infrastructure 基础设施层

基础设施层提供了领域层接口的具体实现，包括缓存存储、分布式锁、一致性哈希等技术实现。该层负责与外部系统交互，如内存管理、网络通信、持久化存储等。

## 📁 包结构

```
infrastructure/
├── cache/                          # 缓存基础设施实现
│   ├── bloom_filter.go            # 布隆过滤器实现
│   ├── bloom_filter_cache.go      # 布隆过滤器缓存
│   ├── bloom_filter_test.go       # 布隆过滤器测试
│   ├── in_memory_bloom_filter.go  # 内存布隆过滤器
│   ├── max_memory_cache.go        # 最大内存缓存
│   ├── max_memory_cache_test.go   # 最大内存缓存测试
│   ├── mock_cache.go              # 模拟缓存（测试用）
│   ├── read_through_cache.go      # 读透缓存
│   ├── read_through_cache_test.go # 读透缓存测试
│   ├── write_back_cache.go        # 写回缓存
│   ├── write_back_cache_test.go   # 写回缓存测试
│   ├── write_through_cache.go     # 写透缓存
│   └── write_through_cache_test.go# 写透缓存测试
├── consistent_hash/                # 一致性哈希基础设施实现
│   ├── consistent_hash_map.go     # 一致性哈希映射
│   ├── consistent_hash_test.go    # 一致性哈希测试
│   └── singleflight_peer_picker.go# SingleFlight节点选择器
└── lock/                           # 分布式锁基础设施实现
    ├── memory_distributed_lock.go # 内存分布式锁
    └── memory_distributed_lock_test.go # 内存分布式锁测试
```

## 🎯 设计原则

### 1. 依赖倒置

- 实现领域层定义的接口
- 不向上依赖应用层
- 通过依赖注入获取外部依赖

### 2. 单一职责

- 每个实现类专注于一种技术方案
- 分离关注点，便于测试和维护
- 避免技术细节泄漏到领域层

### 3. 开闭原则

- 通过实现接口扩展功能
- 支持多种实现策略的切换
- 便于添加新的技术实现

## 🏗️ 核心实现

### 缓存实现

- **MaxMemoryCache**: 支持内存限制和LRU淘汰的基础缓存
- **ReadThroughCache**: 读透模式缓存，自动加载缺失数据
- **WriteThroughCache**: 写透模式缓存，同步写入持久化存储
- **WriteBackCache**: 写回模式缓存，异步批量写入
- **BloomFilterCache**: 带布隆过滤器的缓存，防止缓存穿透

### 分布式锁实现

- **MemoryDistributedLock**: 基于内存的分布式锁
- 支持UUID标识、过期时间、自动续约
- 集成SingleFlight优化，减少竞争

### 一致性哈希实现

- **ConsistentHashMap**: 一致性哈希映射实现
- **SingleflightPeerPicker**: 带SingleFlight优化的节点选择器
- 支持虚拟节点、自定义哈希函数

## 🔧 使用指南

### 缓存使用示例

```go
// 创建最大内存缓存
cache := NewMaxMemoryCache(1024 * 1024) // 1MB

// 创建读透缓存
readThrough := NewReadThroughCache(cache)

// 创建写透缓存
writeThrough := NewWriteThroughCache(cache)

// 创建写回缓存
writeBack := NewWriteBackCache(cache, time.Minute, 100)

// 创建布隆过滤器缓存
bloomConfig, _ := domain.NewBloomFilterConfig(1000, 0.01)
bloomFilter := NewInMemoryBloomFilter(bloomConfig)
bloomCache := NewBloomFilterCacheSimple(cache, bloomFilter, loadFunc)
```

### 分布式锁使用示例

```go
// 创建分布式锁管理器
lockManager := NewMemoryDistributedLock()

// 创建重试策略
retryStrategy := NewFixedIntervalRetryStrategy(100*time.Millisecond, 3)

// 获取锁
lock, err := lockManager.Lock(ctx, "resource_key", time.Minute, 5*time.Second, retryStrategy)
if err != nil {
    return err
}

// 自动续约
go func() {
    _ = lock.AutoRefresh(30*time.Second, 5*time.Second)
}()

// 释放锁
defer lock.Unlock(ctx)
```

### 一致性哈希使用示例

```go
// 创建一致性哈希映射
hashMap := NewConsistentHashMap(150, nil) // 150个虚拟节点

// 创建节点选择器
picker := NewSingleflightPeerPicker(hashMap)

// 添加节点
peer1, _ := domain.NewPeerInfo("server1", "192.168.1.1:8080", 100)
peer2, _ := domain.NewPeerInfo("server2", "192.168.1.2:8080", 100)
picker.AddPeers(peer1, peer2)

// 选择节点
selectedPeer, err := picker.PickPeer("user_123")
if err != nil {
    return err
}

fmt.Printf("用户分配到服务器: %s\n", selectedPeer.ID())
```

## ⚠️ 注意事项

### 1. 内存管理

- 注意内存泄漏，及时清理不用的资源
- 合理设置缓存大小限制
- 监控内存使用情况

### 2. 并发安全

- 所有实现都必须是线程安全的
- 使用适当的同步原语（mutex、channel等）
- 避免死锁和竞态条件

### 3. 错误处理

- 区分可恢复和不可恢复的错误
- 提供有意义的错误信息
- 实现适当的重试机制

### 4. 性能优化

- 避免不必要的内存分配
- 使用对象池减少GC压力
- 合理使用缓存和预计算

### 5. 配置管理

- 提供合理的默认配置
- 支持运行时配置调整
- 验证配置参数的有效性

## 🧪 测试策略

### 单元测试

- 每个实现类都有对应的测试文件
- 使用表格驱动测试提高覆盖率
- 模拟外部依赖进行隔离测试

### 集成测试

- 测试多个组件的协作
- 验证接口契约的正确实现
- 测试异常情况的处理

### 性能测试

- 基准测试关键路径
- 内存使用情况分析
- 并发性能测试

### 示例测试代码

```go
func TestMaxMemoryCache(t *testing.T) {
    tests := []struct {
        name     string
        maxSize  int64
        ops      []operation
        wantSize int64
    }{
        {
            name:    "基本操作",
            maxSize: 1024,
            ops: []operation{
                {op: "set", key: "key1", value: "value1"},
                {op: "get", key: "key1", want: "value1"},
                {op: "delete", key: "key1"},
                {op: "get", key: "key1", wantErr: true},
            },
            wantSize: 0,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            cache := NewMaxMemoryCache(tt.maxSize)
            executeOperations(t, cache, tt.ops)
            assert.Equal(t, tt.wantSize, cache.Size())
        })
    }
}
```

## 🔄 扩展指南

### 添加新的缓存实现

1. **实现领域接口**:

```go
type RedisCacheRepository struct {
    client redis.Client
}

func (r *RedisCacheRepository) Get(ctx context.Context, key string) (any, error) {
    return r.client.Get(ctx, key).Result()
}

func (r *RedisCacheRepository) Set(ctx context.Context, key string, val any, expiration time.Duration) error {
    return r.client.Set(ctx, key, val, expiration).Err()
}
```

2. **添加配置选项**:

```go
type RedisCacheConfig struct {
    Address  string
    Password string
    DB       int
    PoolSize int
}

func NewRedisCacheRepository(config RedisCacheConfig) *RedisCacheRepository {
    client := redis.NewClient(&redis.Options{
        Addr:     config.Address,
        Password: config.Password,
        DB:       config.DB,
        PoolSize: config.PoolSize,
    })
    
    return &RedisCacheRepository{client: client}
}
```

3. **编写测试**:

```go
func TestRedisCacheRepository(t *testing.T) {
    // 使用testcontainers启动Redis实例
    // 或使用mock客户端进行测试
}
```

### 性能优化建议

1. **对象池使用**:

```go
var entryPool = sync.Pool{
    New: func() interface{} {
        return &Entry{}
    },
}

func getEntry() *Entry {
    return entryPool.Get().(*Entry)
}

func putEntry(e *Entry) {
    e.Reset()
    entryPool.Put(e)
}
```

2. **批量操作**:

```go
func (c *Cache) SetBatch(ctx context.Context, items map[string]any, expiration time.Duration) error {
    // 批量设置，减少锁竞争
    c.mu.Lock()
    defer c.mu.Unlock()
    
    for key, value := range items {
        c.data[key] = value
    }
    
    return nil
}
```

3. **异步处理**:

```go
func (c *WriteBackCache) asyncFlush() {
    ticker := time.NewTicker(c.flushInterval)
    defer ticker.Stop()
    
    for {
        select {
        case <-ticker.C:
            c.flushDirtyData()
        case <-c.stopCh:
            return
        }
    }
}
```
