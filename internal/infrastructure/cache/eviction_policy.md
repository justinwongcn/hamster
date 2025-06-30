# eviction_policy.go - 淘汰策略接口定义

## 文件概述

`eviction_policy.go` 定义了缓存淘汰策略的核心接口，遵循DDD设计原则，为不同的淘汰算法提供了统一的抽象。该接口支持用户自定义淘汰算法的实现，是缓存系统中策略模式的核心组件。

## 核心接口

### EvictionPolicy 淘汰策略接口

```go
type EvictionPolicy interface {
    KeyAccessed(ctx context.Context, key string) error
    Evict(ctx context.Context) (string, error)
    Remove(ctx context.Context, key string) error
    Has(ctx context.Context, key string) (bool, error)
    Size(ctx context.Context) (int, error)
    Clear(ctx context.Context) error
}
```

**设计原则：**

- 遵循DDD设计原则
- 支持上下文传递
- 统一的错误处理
- 可扩展的策略实现

## 方法详解

### 1. KeyAccessed - 记录键访问

```go
KeyAccessed(ctx context.Context, key string) error
```

**用途：**

- 记录键被访问的事件
- 更新键的访问状态
- 不同策略有不同的处理方式

**参数：**

- `ctx`: 上下文，传递请求级别信息
- `key`: 被访问的缓存键

**返回值：**

- `error`: 操作错误，nil表示成功

**实现要点：**

- LRU策略：将键移动到最近使用位置
- FIFO策略：如果键不存在则添加到队列尾部
- Random策略：可能不需要特殊处理

### 2. Evict - 执行淘汰

```go
Evict(ctx context.Context) (string, error)
```

**用途：**

- 根据策略选择要淘汰的键
- 执行淘汰操作
- 返回被淘汰的键

**参数：**

- `ctx`: 上下文

**返回值：**

- `string`: 被淘汰的键，空字符串表示没有可淘汰的键
- `error`: 操作错误

**实现要点：**

- LRU策略：淘汰最久未使用的键
- FIFO策略：淘汰最早添加的键
- Random策略：随机选择一个键淘汰

### 3. Remove - 移除指定键

```go
Remove(ctx context.Context, key string) error
```

**用途：**

- 从策略中移除指定的键
- 通常在键被删除时调用
- 清理策略状态

**参数：**

- `ctx`: 上下文
- `key`: 要移除的缓存键

**返回值：**

- `error`: 操作错误

### 4. Has - 检查键是否存在

```go
Has(ctx context.Context, key string) (bool, error)
```

**用途：**

- 检查策略是否正在跟踪指定的键
- 用于调试和监控
- 验证策略状态

### 5. Size - 获取跟踪的键数量

```go
Size(ctx context.Context) (int, error)
```

**用途：**

- 返回策略中跟踪的键数量
- 用于监控和调试
- 检查策略容量

### 6. Clear - 清空所有键

```go
Clear(ctx context.Context) error
```

**用途：**

- 清空策略中的所有键
- 重置策略状态
- 用于缓存清理

## 实现示例

### 1. LRU策略实现示例

```go
type LRUPolicy struct {
    capacity int
    cache    map[string]*Node
    head     *Node
    tail     *Node
    mutex    sync.RWMutex
}

func (l *LRUPolicy) KeyAccessed(ctx context.Context, key string) error {
    l.mutex.Lock()
    defer l.mutex.Unlock()
    
    if node, exists := l.cache[key]; exists {
        // 移动到头部
        l.moveToHead(node)
    } else {
        // 添加新节点
        newNode := &Node{Key: key}
        l.cache[key] = newNode
        l.addToHead(newNode)
        
        // 检查容量
        if len(l.cache) > l.capacity {
            tail := l.removeTail()
            delete(l.cache, tail.Key)
        }
    }
    
    return nil
}

func (l *LRUPolicy) Evict(ctx context.Context) (string, error) {
    l.mutex.Lock()
    defer l.mutex.Unlock()
    
    if len(l.cache) == 0 {
        return "", nil
    }
    
    tail := l.removeTail()
    delete(l.cache, tail.Key)
    
    return tail.Key, nil
}
```

### 2. FIFO策略实现示例

```go
type FIFOPolicy struct {
    capacity int
    queue    []string
    cache    map[string]bool
    mutex    sync.RWMutex
}

func (f *FIFOPolicy) KeyAccessed(ctx context.Context, key string) error {
    f.mutex.Lock()
    defer f.mutex.Unlock()
    
    if !f.cache[key] {
        f.queue = append(f.queue, key)
        f.cache[key] = true
        
        // 检查容量
        if len(f.queue) > f.capacity {
            oldKey := f.queue[0]
            f.queue = f.queue[1:]
            delete(f.cache, oldKey)
        }
    }
    
    return nil
}

func (f *FIFOPolicy) Evict(ctx context.Context) (string, error) {
    f.mutex.Lock()
    defer f.mutex.Unlock()
    
    if len(f.queue) == 0 {
        return "", nil
    }
    
    key := f.queue[0]
    f.queue = f.queue[1:]
    delete(f.cache, key)
    
    return key, nil
}
```

### 3. Random策略实现示例

```go
type RandomPolicy struct {
    keys  []string
    cache map[string]int // key -> index
    mutex sync.RWMutex
}

func (r *RandomPolicy) KeyAccessed(ctx context.Context, key string) error {
    r.mutex.Lock()
    defer r.mutex.Unlock()
    
    if _, exists := r.cache[key]; !exists {
        r.keys = append(r.keys, key)
        r.cache[key] = len(r.keys) - 1
    }
    
    return nil
}

func (r *RandomPolicy) Evict(ctx context.Context) (string, error) {
    r.mutex.Lock()
    defer r.mutex.Unlock()
    
    if len(r.keys) == 0 {
        return "", nil
    }
    
    // 随机选择一个键
    index := rand.Intn(len(r.keys))
    key := r.keys[index]
    
    // 移除键
    r.keys[index] = r.keys[len(r.keys)-1]
    r.keys = r.keys[:len(r.keys)-1]
    
    // 更新索引
    if index < len(r.keys) {
        r.cache[r.keys[index]] = index
    }
    delete(r.cache, key)
    
    return key, nil
}
```

## 使用示例

### 1. 策略工厂模式

```go
type PolicyType int

const (
    LRU PolicyType = iota
    FIFO
    Random
)

func CreateEvictionPolicy(policyType PolicyType, capacity int) EvictionPolicy {
    switch policyType {
    case LRU:
        return NewLRUPolicy(capacity)
    case FIFO:
        return NewFIFOPolicy(capacity)
    case Random:
        return NewRandomPolicy(capacity)
    default:
        return NewLRUPolicy(capacity) // 默认使用LRU
    }
}

// 使用示例
func demonstratePolicyFactory() {
    ctx := context.Background()
    
    // 创建不同类型的策略
    lruPolicy := CreateEvictionPolicy(LRU, 100)
    fifoPolicy := CreateEvictionPolicy(FIFO, 100)
    randomPolicy := CreateEvictionPolicy(Random, 100)
    
    policies := []EvictionPolicy{lruPolicy, fifoPolicy, randomPolicy}
    policyNames := []string{"LRU", "FIFO", "Random"}
    
    // 测试所有策略
    for i, policy := range policies {
        fmt.Printf("测试 %s 策略:\n", policyNames[i])
        
        // 添加一些键
        for j := 0; j < 5; j++ {
            key := fmt.Sprintf("key_%d", j)
            err := policy.KeyAccessed(ctx, key)
            if err != nil {
                log.Printf("添加键失败: %v", err)
            }
        }
        
        // 检查大小
        size, err := policy.Size(ctx)
        if err == nil {
            fmt.Printf("  策略大小: %d\n", size)
        }
        
        // 执行淘汰
        evictedKey, err := policy.Evict(ctx)
        if err == nil && evictedKey != "" {
            fmt.Printf("  淘汰的键: %s\n", evictedKey)
        }
        
        fmt.Println()
    }
}
```

### 2. 策略性能比较

```go
func comparePolicyPerformance() {
    ctx := context.Background()
    capacity := 1000
    testKeys := 10000
    
    policies := map[string]EvictionPolicy{
        "LRU":    CreateEvictionPolicy(LRU, capacity),
        "FIFO":   CreateEvictionPolicy(FIFO, capacity),
        "Random": CreateEvictionPolicy(Random, capacity),
    }
    
    for name, policy := range policies {
        fmt.Printf("测试 %s 策略性能:\n", name)
        
        // 测试添加性能
        start := time.Now()
        for i := 0; i < testKeys; i++ {
            key := fmt.Sprintf("key_%d", i)
            policy.KeyAccessed(ctx, key)
        }
        addDuration := time.Since(start)
        
        // 测试查询性能
        start = time.Now()
        for i := 0; i < 1000; i++ {
            key := fmt.Sprintf("key_%d", rand.Intn(testKeys))
            policy.Has(ctx, key)
        }
        queryDuration := time.Since(start)
        
        // 测试淘汰性能
        start = time.Now()
        for i := 0; i < 100; i++ {
            policy.Evict(ctx)
        }
        evictDuration := time.Since(start)
        
        fmt.Printf("  添加 %d 个键: %v\n", testKeys, addDuration)
        fmt.Printf("  查询 1000 次: %v\n", queryDuration)
        fmt.Printf("  淘汰 100 次: %v\n", evictDuration)
        fmt.Printf("  平均添加时间: %v\n", addDuration/time.Duration(testKeys))
        fmt.Printf("  平均查询时间: %v\n", queryDuration/1000)
        fmt.Printf("  平均淘汰时间: %v\n", evictDuration/100)
        fmt.Println()
    }
}
```

### 3. 策略行为测试

```go
func testPolicyBehavior() {
    ctx := context.Background()
    capacity := 3
    
    policies := map[string]EvictionPolicy{
        "LRU":  CreateEvictionPolicy(LRU, capacity),
        "FIFO": CreateEvictionPolicy(FIFO, capacity),
    }
    
    // 测试访问序列
    accessSequence := []string{"A", "B", "C", "A", "D"}
    
    for name, policy := range policies {
        fmt.Printf("%s 策略行为测试:\n", name)
        
        for _, key := range accessSequence {
            fmt.Printf("  访问键: %s\n", key)
            
            // 记录访问
            err := policy.KeyAccessed(ctx, key)
            if err != nil {
                log.Printf("访问失败: %v", err)
                continue
            }
            
            // 检查当前大小
            size, err := policy.Size(ctx)
            if err == nil {
                fmt.Printf("    当前大小: %d\n", size)
            }
            
            // 如果超过容量，执行淘汰
            if size > capacity {
                evictedKey, err := policy.Evict(ctx)
                if err == nil && evictedKey != "" {
                    fmt.Printf("    淘汰键: %s\n", evictedKey)
                }
            }
        }
        
        fmt.Println()
    }
}
```

### 4. 策略监控

```go
type PolicyMonitor struct {
    policy    EvictionPolicy
    hits      int64
    misses    int64
    evictions int64
    mutex     sync.RWMutex
}

func NewPolicyMonitor(policy EvictionPolicy) *PolicyMonitor {
    return &PolicyMonitor{
        policy: policy,
    }
}

func (m *PolicyMonitor) KeyAccessed(ctx context.Context, key string) error {
    // 检查键是否存在
    exists, err := m.policy.Has(ctx, key)
    if err != nil {
        return err
    }
    
    m.mutex.Lock()
    if exists {
        m.hits++
    } else {
        m.misses++
    }
    m.mutex.Unlock()
    
    return m.policy.KeyAccessed(ctx, key)
}

func (m *PolicyMonitor) Evict(ctx context.Context) (string, error) {
    key, err := m.policy.Evict(ctx)
    if err == nil && key != "" {
        m.mutex.Lock()
        m.evictions++
        m.mutex.Unlock()
    }
    return key, err
}

func (m *PolicyMonitor) GetStats() (hits, misses, evictions int64, hitRate float64) {
    m.mutex.RLock()
    defer m.mutex.RUnlock()
    
    hits = m.hits
    misses = m.misses
    evictions = m.evictions
    
    total := hits + misses
    if total > 0 {
        hitRate = float64(hits) / float64(total)
    }
    
    return
}

// 使用监控器
func demonstrateMonitoring() {
    ctx := context.Background()
    policy := CreateEvictionPolicy(LRU, 100)
    monitor := NewPolicyMonitor(policy)
    
    // 模拟访问
    for i := 0; i < 1000; i++ {
        key := fmt.Sprintf("key_%d", rand.Intn(150)) // 150个不同的键，容量只有100
        monitor.KeyAccessed(ctx, key)
    }
    
    // 获取统计信息
    hits, misses, evictions, hitRate := monitor.GetStats()
    
    fmt.Printf("策略监控统计:\n")
    fmt.Printf("  命中: %d\n", hits)
    fmt.Printf("  未命中: %d\n", misses)
    fmt.Printf("  淘汰: %d\n", evictions)
    fmt.Printf("  命中率: %.2f%%\n", hitRate*100)
}
```

## 设计原则

### 1. 接口隔离原则

- 接口方法职责单一
- 每个方法都有明确的用途
- 避免接口过于庞大

### 2. 开闭原则

- 对扩展开放：可以实现新的淘汰策略
- 对修改关闭：不需要修改现有代码

### 3. 依赖倒置原则

- 高层模块依赖抽象接口
- 具体实现依赖抽象接口
- 便于测试和扩展

### 4. 策略模式

- 封装不同的淘汰算法
- 运行时可以切换策略
- 算法独立于使用它的客户端

## 实现建议

### 1. 线程安全

```go
// ✅ 推荐：使用读写锁保证并发安全
type MyPolicy struct {
    mutex sync.RWMutex
    data  map[string]*Node
}

func (p *MyPolicy) KeyAccessed(ctx context.Context, key string) error {
    p.mutex.Lock()
    defer p.mutex.Unlock()
    // 实现逻辑
    return nil
}

func (p *MyPolicy) Has(ctx context.Context, key string) (bool, error) {
    p.mutex.RLock()
    defer p.mutex.RUnlock()
    // 实现逻辑
    return false, nil
}
```

### 2. 错误处理

```go
// ✅ 推荐：定义明确的错误类型
var (
    ErrPolicyFull    = errors.New("策略已满")
    ErrKeyNotFound   = errors.New("键不存在")
    ErrInvalidKey    = errors.New("无效的键")
)

func (p *MyPolicy) KeyAccessed(ctx context.Context, key string) error {
    if key == "" {
        return ErrInvalidKey
    }
    // 实现逻辑
    return nil
}
```

### 3. 上下文处理

```go
// ✅ 推荐：正确处理上下文
func (p *MyPolicy) KeyAccessed(ctx context.Context, key string) error {
    select {
    case <-ctx.Done():
        return ctx.Err()
    default:
    }
    
    // 实现逻辑
    return nil
}
```

### 4. 性能优化

```go
// ✅ 推荐：使用高效的数据结构
type LRUPolicy struct {
    capacity int
    cache    map[string]*Node  // O(1) 查找
    head     *Node             // 双向链表头
    tail     *Node             // 双向链表尾
}

// ✅ 推荐：避免不必要的内存分配
func (p *LRUPolicy) moveToHead(node *Node) {
    // 直接操作指针，避免创建新节点
    p.removeNode(node)
    p.addToHead(node)
}
```

## 注意事项

### 1. 容量管理

- 合理设置策略容量
- 避免无限增长
- 考虑内存使用

### 2. 性能考虑

- 选择合适的数据结构
- 优化热点路径
- 减少锁竞争

### 3. 一致性保证

- 确保策略状态一致
- 处理并发访问
- 避免竞态条件

### 4. 可测试性

- 提供状态查询方法
- 支持策略重置
- 便于单元测试

EvictionPolicy接口为缓存系统提供了灵活的淘汰策略抽象，支持多种算法实现，是构建高效缓存系统的重要组件。
