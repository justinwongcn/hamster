# Hamster API 文档

本文档描述了 Hamster 分布式缓存系统的主要 API 接口和使用方法。

## 📋 目录

- [缓存 API](#缓存-api)
- [分布式锁 API](#分布式锁-api)
- [一致性哈希 API](#一致性哈希-api)
- [布隆过滤器 API](#布隆过滤器-api)
- [错误处理](#错误处理)
- [最佳实践](#最佳实践)

## 🗄️ 缓存 API

### 基础缓存操作

#### 设置缓存

```go
func (repo Repository) Set(ctx context.Context, key string, val any, expiration time.Duration) error
```

**参数：**

- `ctx`: 上下文，用于超时控制
- `key`: 缓存键，不能为空，长度不超过250字符
- `val`: 缓存值，支持任意类型
- `expiration`: 过期时间，0表示永不过期

**示例：**

```go
err := cache.Set(ctx, "user:123", userData, time.Hour)
if err != nil {
log.Printf("设置缓存失败: %v", err)
}
```

#### 获取缓存

```go
func (repo Repository) Get(ctx context.Context, key string) (any, error)
```

**返回：**

- 缓存值或 `ErrKeyNotFound` 错误

**示例：**

```go
value, err := cache.Get(ctx, "user:123")
if err != nil {
if errors.Is(err, ErrKeyNotFound) {
// 处理键不存在的情况
return handleMissingKey()
}
return err
}
```

#### 删除缓存

```go
func (repo Repository) Delete(ctx context.Context, key string) error
```

#### 检查键是否存在

```go
func (repo Repository) Exists(ctx context.Context, key string) (bool, error)
```

#### 获取统计信息

```go
func (repo Repository) Stats(ctx context.Context) (map[string]any, error)
```

### 高级缓存模式

#### 读透缓存

```go
func (repo ReadThroughRepository) GetWithLoader(
ctx context.Context,
key string,
loader func (ctx context.Context, key string) (any, error)
) (any, error)
```

**示例：**

```go
user, err := readThroughCache.GetWithLoader(ctx, "user:123", func (ctx context.Context, key string) (any, error) {
return userService.LoadFromDB(ctx, key)
})
```

#### 写透缓存

```go
func (repo WriteThroughRepository) SetWithWriter(
ctx context.Context,
key string,
val any,
expiration time.Duration,
writer func (ctx context.Context, key string, val any) error
) error
```

#### 写回缓存

```go
func (repo WriteBackRepository) MarkDirty(ctx context.Context, key string) error
func (repo WriteBackRepository) FlushDirty(ctx context.Context) error
func (repo WriteBackRepository) FlushKey(ctx context.Context, key string) error
```

## 🔒 分布式锁 API

### 基础锁操作

#### 尝试获取锁

```go
func (lock DistributedLock) TryLock(ctx context.Context, key string, expiration time.Duration) (Lock, error)
```

**示例：**

```go
lock, err := lockManager.TryLock(ctx, "resource:123", time.Minute)
if err != nil {
if errors.Is(err, ErrFailedToPreemptLock) {
// 锁被其他进程持有
return handleLockBusy()
}
return err
}
defer lock.Unlock(ctx)
```

#### 带重试的获取锁

```go
func (lock DistributedLock) Lock(
ctx context.Context,
key string,
expiration time.Duration,
timeout time.Duration,
retryStrategy RetryStrategy
) (Lock, error)
```

**示例：**

```go
retryStrategy := NewExponentialBackoffRetryStrategy(100*time.Millisecond, 2.0, 5)
lock, err := lockManager.Lock(ctx, "resource:123", time.Minute, 10*time.Second, retryStrategy)
```

#### SingleFlight 优化锁

```go
func (lock DistributedLock) SingleflightLock(
ctx context.Context,
key string,
expiration time.Duration,
timeout time.Duration,
retryStrategy RetryStrategy
) (Lock, error)
```

### 锁管理操作

#### 手动续约

```go
func (lock Lock) Refresh(ctx context.Context) error
```

#### 自动续约

```go
func (lock Lock) AutoRefresh(interval time.Duration, timeout time.Duration) error
```

**示例：**

```go
// 启动自动续约（异步）
go func () {
err := lock.AutoRefresh(30*time.Second, 5*time.Second)
if err != nil {
log.Printf("自动续约失败: %v", err)
}
}()
```

#### 释放锁

```go
func (lock Lock) Unlock(ctx context.Context) error
```

### 重试策略

#### 固定间隔重试

```go
strategy := NewFixedIntervalRetryStrategy(100*time.Millisecond, 5)
```

#### 指数退避重试

```go
strategy := NewExponentialBackoffRetryStrategy(100*time.Millisecond, 2.0, 5)
```

#### 线性退避重试

```go
strategy := NewLinearBackoffRetryStrategy(100*time.Millisecond, 50*time.Millisecond, 5)
```

## ⚖️ 一致性哈希 API

### 哈希映射操作

#### 创建一致性哈希映射

```go
hashMap := NewConsistentHashMap(replicas int, hashFunc Hash) *ConsistentHashMap
```

**示例：**

```go
// 使用默认哈希函数和150个虚拟节点
hashMap := NewConsistentHashMap(150, nil)

// 使用自定义哈希函数
customHash := func (data []byte) uint32 {
return crc32.ChecksumIEEE(data)
}
hashMap := NewConsistentHashMap(100, customHash)
```

#### 添加节点

```go
func (m *ConsistentHashMap) Add(peers ...string)
```

**示例：**

```go
hashMap.Add("server1", "server2", "server3")
```

#### 移除节点

```go
func (m *ConsistentHashMap) Remove(peers ...string)
```

#### 获取节点

```go
func (m *ConsistentHashMap) Get(key string) (string, error)
```

**示例：**

```go
server, err := hashMap.Get("user:123")
if err != nil {
return err
}
fmt.Printf("用户分配到服务器: %s\n", server)
```

#### 获取多个节点

```go
func (m *ConsistentHashMap) GetMultiple(key string, count int) ([]string, error)
```

### 节点选择器

#### 创建节点选择器

```go
picker := NewSingleflightPeerPicker(consistentHash ConsistentHash) *SingleflightPeerPicker
```

#### 选择节点

```go
func (p *SingleflightPeerPicker) PickPeer(key string) (Peer, error)
func (p *SingleflightPeerPicker) PickPeers(key string, count int) ([]Peer, error)
```

**示例：**

```go
peer, err := picker.PickPeer("user:123")
if err != nil {
return err
}

fmt.Printf("选中节点: %s (%s)\n", peer.ID(), peer.Address())
```

#### 节点管理

```go
func (p *SingleflightPeerPicker) AddPeers(peers ...Peer)
func (p *SingleflightPeerPicker) RemovePeers(peers ...Peer)
func (p *SingleflightPeerPicker) UpdatePeerStatus(peerID string, alive bool) error
```

## 🌸 布隆过滤器 API

### 布隆过滤器操作

#### 创建布隆过滤器

```go
config, err := NewBloomFilterConfig(expectedElements uint64, falsePositiveRate float64)
bloomFilter := NewInMemoryBloomFilter(config)
```

**示例：**

```go
// 预期1000个元素，1%假阳性率
config, err := NewBloomFilterConfig(1000, 0.01)
if err != nil {
return err
}

bloomFilter := NewInMemoryBloomFilter(config)
```

#### 添加元素

```go
func (bf BloomFilter) Add(ctx context.Context, key string) error
```

#### 检查元素

```go
func (bf BloomFilter) HasKey(ctx context.Context, key string) bool
```

**示例：**

```go
// 添加元素
err := bloomFilter.Add(ctx, "user:123")
if err != nil {
return err
}

// 检查元素
exists := bloomFilter.HasKey(ctx, "user:123")
if exists {
fmt.Println("元素可能存在")
} else {
fmt.Println("元素一定不存在")
}
```

#### 获取统计信息

```go
func (bf BloomFilter) Stats(ctx context.Context) (BloomFilterStats, error)
```

### 布隆过滤器缓存

#### 创建布隆过滤器缓存

```go
bloomCache := NewBloomFilterCacheSimple(
repository Repository,
bloomFilter BloomFilter,
loadFunc func (ctx context.Context, key string) (any, error)
)
```

**示例：**

```go
loadFunc := func (ctx context.Context, key string) (any, error) {
return userService.LoadFromDB(ctx, key)
}

bloomCache := NewBloomFilterCacheSimple(memoryCache, bloomFilter, loadFunc)

// 使用时会自动利用布隆过滤器优化
user, err := bloomCache.Get(ctx, "user:123")
```

## ❌ 错误处理

### 常见错误类型

```go
// 缓存错误
var (
ErrKeyNotFound = errors.New("键不存在")
ErrInvalidCacheKey = errors.New("无效的缓存键")
ErrCacheFull            = errors.New("缓存已满")
ErrFailedToRefreshCache = errors.New("刷新缓存失败")
)

// 锁错误
var (
ErrFailedToPreemptLock = errors.New("抢锁失败")
ErrLockNotHold = errors.New("你没有持有锁")
ErrLockExpired         = errors.New("锁已过期")
)

// 哈希错误
var (
ErrNoPeers = errors.New("没有可用的节点")
ErrInvalidPeer = errors.New("无效的节点")
)
```

### 错误处理示例

```go
value, err := cache.Get(ctx, "key")
if err != nil {
switch {
case errors.Is(err, ErrKeyNotFound):
// 处理键不存在
return handleMissingKey()
case errors.Is(err, context.DeadlineExceeded):
// 处理超时
return handleTimeout()
default:
// 处理其他错误
return fmt.Errorf("获取缓存失败: %w", err)
}
}
```

## 💡 最佳实践

### 1. 上下文使用

```go
// ✅ 正确：设置合理的超时时间
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

value, err := cache.Get(ctx, "key")
```

### 2. 错误处理

```go
// ✅ 正确：区分不同类型的错误
value, err := cache.Get(ctx, "key")
if err != nil {
if errors.Is(err, ErrKeyNotFound) {
// 键不存在是正常情况，不需要记录错误日志
return nil, nil
}
// 其他错误需要记录日志
log.Printf("获取缓存失败: %v", err)
return nil, err
}
```

### 3. 资源清理

```go
// ✅ 正确：确保锁被释放
lock, err := lockManager.TryLock(ctx, "resource", time.Minute)
if err != nil {
return err
}
defer func () {
if unlockErr := lock.Unlock(ctx); unlockErr != nil {
log.Printf("释放锁失败: %v", unlockErr)
}
}()
```

### 4. 配置优化

```go
// ✅ 正确：根据业务需求配置参数
config, err := NewBloomFilterConfig(
expectedElements, // 根据实际数据量设置
0.01,             // 1%假阳性率，平衡内存和准确性
)

hashMap := NewConsistentHashMap(
150, // 虚拟节点数，提升负载均衡
nil, // 使用默认哈希函数
)
```

### 5. 监控和观测

```go
// 定期获取统计信息
go func () {
ticker := time.NewTicker(time.Minute)
defer ticker.Stop()

for {
select {
case <-ticker.C:
stats, err := cache.Stats(ctx)
if err != nil {
log.Printf("获取统计信息失败: %v", err)
continue
}

// 记录关键指标
log.Printf("缓存统计: %+v", stats)
}
}
}()
```
