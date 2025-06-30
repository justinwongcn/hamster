# Hamster 性能优化指南

本文档提供了 Hamster 分布式缓存系统的性能优化建议和最佳实践。

## 📊 性能指标

### 关键性能指标 (KPIs)

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

## 🚀 缓存性能优化

### 1. 内存管理优化

#### 设置合理的内存限制

```go
// ✅ 推荐：根据系统内存设置合理的缓存大小
totalMemory := 8 * 1024 * 1024 * 1024 // 8GB
cacheMemory := totalMemory * 60 / 100  // 使用60%内存作为缓存
cache := NewMaxMemoryCache(cacheMemory)

// ❌ 避免：设置过大的内存限制导致OOM
cache := NewMaxMemoryCache(math.MaxInt64)
```

#### 优化对象大小

```go
// ✅ 推荐：存储轻量级对象
type UserCache struct {
    ID   int64  `json:"id"`
    Name string `json:"name"`
}

// ❌ 避免：存储包含大量数据的对象
type HeavyUserCache struct {
    ID       int64    `json:"id"`
    Name     string   `json:"name"`
    Avatar   []byte   `json:"avatar"`     // 大文件
    History  []string `json:"history"`   // 大数组
    Metadata map[string]interface{} `json:"metadata"` // 复杂对象
}
```

#### 使用对象池减少GC压力

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
    e.Reset() // 重置对象状态
    entryPool.Put(e)
}
```

### 2. 缓存策略优化

#### 选择合适的缓存模式

```go
// 读多写少的场景：使用读透缓存
readThroughCache := NewReadThroughCache(baseCache)

// 写多读少的场景：使用写回缓存
writeBackCache := NewWriteBackCache(baseCache, 
    time.Minute,  // 刷新间隔
    1000,         // 批量大小
)

// 强一致性要求：使用写透缓存
writeThroughCache := NewWriteThroughCache(baseCache)
```

#### 优化过期时间设置

```go
// ✅ 推荐：根据数据特性设置不同的过期时间
func getCacheExpiration(dataType string) time.Duration {
    switch dataType {
    case "user_profile":
        return 24 * time.Hour    // 用户资料变化较少
    case "user_session":
        return 30 * time.Minute  // 会话数据中等时效
    case "real_time_data":
        return 5 * time.Minute   // 实时数据短时效
    default:
        return time.Hour
    }
}
```

#### 使用布隆过滤器防止缓存穿透

```go
// 配置布隆过滤器
config, _ := NewBloomFilterConfig(
    100000, // 预期元素数量
    0.01,   // 1%假阳性率
)
bloomFilter := NewInMemoryBloomFilter(config)

// 创建带布隆过滤器的缓存
bloomCache := NewBloomFilterCacheSimple(baseCache, bloomFilter, loadFunc)
```

### 3. 并发优化

#### 减少锁竞争

```go
// ✅ 推荐：使用分段锁减少竞争
type ShardedCache struct {
    shards []*CacheShard
    mask   uint32
}

func (c *ShardedCache) getShard(key string) *CacheShard {
    hash := fnv.New32a()
    hash.Write([]byte(key))
    return c.shards[hash.Sum32()&c.mask]
}

// ❌ 避免：全局锁导致性能瓶颈
type GlobalLockCache struct {
    mu   sync.RWMutex
    data map[string]interface{}
}
```

#### 使用读写锁优化读操作

```go
type OptimizedCache struct {
    mu   sync.RWMutex
    data map[string]*Entry
}

func (c *OptimizedCache) Get(key string) (interface{}, error) {
    c.mu.RLock()         // 读锁
    defer c.mu.RUnlock()
    
    entry, exists := c.data[key]
    if !exists {
        return nil, ErrKeyNotFound
    }
    
    return entry.Value, nil
}
```

## 🔒 分布式锁性能优化

### 1. 锁粒度优化

```go
// ✅ 推荐：细粒度锁
func processUser(userID string) error {
    lockKey := fmt.Sprintf("user:%s", userID)
    lock, err := lockManager.TryLock(ctx, lockKey, time.Minute)
    // 只锁定特定用户
}

// ❌ 避免：粗粒度锁
func processUser(userID string) error {
    lock, err := lockManager.TryLock(ctx, "global_user_lock", time.Minute)
    // 锁定所有用户操作
}
```

### 2. 重试策略优化

```go
// ✅ 推荐：指数退避重试，避免惊群效应
retryStrategy := NewExponentialBackoffRetryStrategy(
    10*time.Millisecond,  // 初始间隔
    2.0,                  // 倍数因子
    5,                    // 最大重试次数
)

// 添加随机抖动
type JitteredRetryStrategy struct {
    base RetryStrategy
}

func (j *JitteredRetryStrategy) Iterator() iter.Seq[time.Duration] {
    return func(yield func(time.Duration) bool) {
        for interval := range j.base.Iterator() {
            // 添加±25%的随机抖动
            jitter := time.Duration(rand.Float64() * 0.5 * float64(interval))
            actualInterval := interval + jitter - time.Duration(0.25*float64(interval))
            if !yield(actualInterval) {
                return
            }
        }
    }
}
```

### 3. 锁超时优化

```go
// ✅ 推荐：根据业务逻辑设置合理的锁超时时间
func processOrder(orderID string) error {
    // 订单处理通常需要较长时间
    lockTimeout := 5 * time.Minute
    lockExpiration := 10 * time.Minute
    
    lock, err := lockManager.Lock(ctx, 
        fmt.Sprintf("order:%s", orderID),
        lockExpiration,
        lockTimeout,
        retryStrategy,
    )
    
    // 启动自动续约
    go func() {
        _ = lock.AutoRefresh(2*time.Minute, 30*time.Second)
    }()
    
    defer lock.Unlock(ctx)
    
    return processOrderLogic(orderID)
}
```

## ⚖️ 一致性哈希性能优化

### 1. 虚拟节点数量优化

```go
// 根据节点数量和负载均衡要求调整虚拟节点数量
func calculateOptimalReplicas(nodeCount int, targetBalance float64) int {
    // 经验公式：虚拟节点数 = 150 * log(节点数)
    replicas := int(150 * math.Log(float64(nodeCount)))
    
    // 最小值保证
    if replicas < 50 {
        replicas = 50
    }
    
    // 最大值限制
    if replicas > 500 {
        replicas = 500
    }
    
    return replicas
}

// 使用优化的虚拟节点数量
nodeCount := 10
replicas := calculateOptimalReplicas(nodeCount, 0.1) // 10%的负载不均衡容忍度
hashMap := NewConsistentHashMap(replicas, nil)
```

### 2. 哈希函数优化

```go
// ✅ 推荐：使用高性能哈希函数
import "github.com/cespare/xxhash/v2"

func xxHash(data []byte) uint32 {
    return uint32(xxhash.Sum64(data))
}

hashMap := NewConsistentHashMap(150, xxHash)

// 性能对比测试
func BenchmarkHashFunctions(b *testing.B) {
    data := []byte("test_key_for_hashing")
    
    b.Run("CRC32", func(b *testing.B) {
        for i := 0; i < b.N; i++ {
            _ = crc32.ChecksumIEEE(data)
        }
    })
    
    b.Run("XXHash", func(b *testing.B) {
        for i := 0; i < b.N; i++ {
            _ = xxhash.Sum64(data)
        }
    })
}
```

### 3. 节点选择优化

```go
// ✅ 推荐：使用SingleFlight减少重复计算
picker := NewSingleflightPeerPicker(hashMap)

// 批量节点选择优化
func (p *SingleflightPeerPicker) PickPeersBatch(keys []string) (map[string]Peer, error) {
    results := make(map[string]Peer)
    var mu sync.Mutex
    var wg sync.WaitGroup
    
    // 并发处理多个键
    for _, key := range keys {
        wg.Add(1)
        go func(k string) {
            defer wg.Done()
            
            peer, err := p.PickPeer(k)
            if err == nil {
                mu.Lock()
                results[k] = peer
                mu.Unlock()
            }
        }(key)
    }
    
    wg.Wait()
    return results, nil
}
```

## 📈 监控和调优

### 1. 性能监控

```go
// 性能指标收集
type PerformanceMetrics struct {
    CacheHits        int64
    CacheMisses      int64
    LockAcquisitions int64
    LockFailures     int64
    HashOperations   int64
    ResponseTimes    []time.Duration
}

func (m *PerformanceMetrics) RecordCacheHit() {
    atomic.AddInt64(&m.CacheHits, 1)
}

func (m *PerformanceMetrics) RecordResponseTime(duration time.Duration) {
    // 使用环形缓冲区记录最近的响应时间
    m.mu.Lock()
    defer m.mu.Unlock()
    
    if len(m.ResponseTimes) >= 1000 {
        m.ResponseTimes = m.ResponseTimes[1:]
    }
    m.ResponseTimes = append(m.ResponseTimes, duration)
}

func (m *PerformanceMetrics) GetP95ResponseTime() time.Duration {
    m.mu.RLock()
    defer m.mu.RUnlock()
    
    if len(m.ResponseTimes) == 0 {
        return 0
    }
    
    sorted := make([]time.Duration, len(m.ResponseTimes))
    copy(sorted, m.ResponseTimes)
    sort.Slice(sorted, func(i, j int) bool {
        return sorted[i] < sorted[j]
    })
    
    index := int(float64(len(sorted)) * 0.95)
    return sorted[index]
}
```

### 2. 自动调优

```go
// 自适应缓存大小调整
type AdaptiveCache struct {
    *MaxMemoryCache
    metrics     *PerformanceMetrics
    adjustTimer *time.Timer
}

func (c *AdaptiveCache) autoTune() {
    hitRate := float64(c.metrics.CacheHits) / float64(c.metrics.CacheHits + c.metrics.CacheMisses)
    
    if hitRate < 0.8 && c.GetMemoryUsage() < c.GetMaxMemory()*0.8 {
        // 命中率低且内存充足，增加缓存大小
        newSize := c.GetMaxMemory() * 110 / 100 // 增加10%
        c.SetMaxMemory(newSize)
    } else if hitRate > 0.95 && c.GetMemoryUsage() > c.GetMaxMemory()*0.9 {
        // 命中率高但内存紧张，可以适当减少缓存大小
        newSize := c.GetMaxMemory() * 95 / 100 // 减少5%
        c.SetMaxMemory(newSize)
    }
}
```

### 3. 性能基准测试

```go
func BenchmarkCacheOperations(b *testing.B) {
    cache := NewMaxMemoryCache(1024 * 1024) // 1MB
    
    b.Run("Set", func(b *testing.B) {
        b.ResetTimer()
        for i := 0; i < b.N; i++ {
            key := fmt.Sprintf("key_%d", i)
            _ = cache.Set(context.Background(), key, "value", time.Hour)
        }
    })
    
    b.Run("Get", func(b *testing.B) {
        // 预填充数据
        for i := 0; i < 1000; i++ {
            key := fmt.Sprintf("key_%d", i)
            _ = cache.Set(context.Background(), key, "value", time.Hour)
        }
        
        b.ResetTimer()
        for i := 0; i < b.N; i++ {
            key := fmt.Sprintf("key_%d", i%1000)
            _, _ = cache.Get(context.Background(), key)
        }
    })
}

func BenchmarkConcurrentAccess(b *testing.B) {
    cache := NewMaxMemoryCache(1024 * 1024)
    
    b.RunParallel(func(pb *testing.PB) {
        i := 0
        for pb.Next() {
            key := fmt.Sprintf("key_%d", i%1000)
            if i%2 == 0 {
                _ = cache.Set(context.Background(), key, "value", time.Hour)
            } else {
                _, _ = cache.Get(context.Background(), key)
            }
            i++
        }
    })
}
```

## 🎯 性能调优检查清单

### 缓存层面

- [ ] 设置合理的内存限制
- [ ] 选择适当的缓存策略
- [ ] 配置合理的过期时间
- [ ] 使用布隆过滤器防止穿透
- [ ] 实现缓存预热机制

### 并发层面

- [ ] 使用分段锁减少竞争
- [ ] 优化读写锁使用
- [ ] 避免长时间持有锁
- [ ] 实现无锁数据结构

### 分布式锁层面

- [ ] 使用细粒度锁
- [ ] 配置合理的重试策略
- [ ] 实现自动续约机制
- [ ] 添加锁超时保护

### 一致性哈希层面

- [ ] 优化虚拟节点数量
- [ ] 选择高性能哈希函数
- [ ] 使用SingleFlight优化
- [ ] 实现节点健康检查

### 监控层面

- [ ] 收集关键性能指标
- [ ] 设置性能告警阈值
- [ ] 定期进行性能测试
- [ ] 实现自动调优机制
