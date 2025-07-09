# fifo_policy.go - FIFO淘汰策略实现

## 文件概述

`fifo_policy.go` 实现了FIFO（First In First Out）缓存淘汰策略。采用单向链表+哈希表的数据结构，提供O(1)
时间复杂度的插入、删除和查找操作。该实现线程安全，支持并发访问，适用于顺序访问模式的缓存场景。

## 核心功能

### 1. 数据结构设计

#### fifoNode 队列节点

```go
type fifoNode struct {
    key  string    // 节点存储的键
    next *fifoNode // 指向下一个节点的指针
}
```

**设计特点：**

- 单向链表节点，节省内存空间
- 只存储键，值存储在外部缓存中
- 简洁的结构设计，减少内存占用

#### FIFOPolicy 策略结构

```go
type FIFOPolicy struct {
    capacity int                  // 容量限制，0表示无限制
    size     int                  // 当前大小
    cache    map[string]*fifoNode // 哈希表，快速定位节点
    head     *fifoNode            // 队列头（最早添加）
    tail     *fifoNode            // 队列尾（最新添加）
    mutex    sync.RWMutex         // 读写锁，保证并发安全
}
```

**设计特点：**

- 哈希表提供O(1)查找性能
- 单向链表维护插入顺序
- 头尾指针简化队列操作
- 读写锁保证线程安全

### 2. 构造函数

#### NewFIFOPolicy

```go
func NewFIFOPolicy(capacity ...int) *FIFOPolicy
```

创建新的FIFO策略实例，支持可选的容量限制。

**参数：**

- `capacity`: 可选参数，容量限制，0或不传表示无限制

**示例：**

```go
// 无限制容量
policy := NewFIFOPolicy()

// 限制容量为100
policy := NewFIFOPolicy(100)
```

### 3. 核心操作

#### KeyAccessed - 记录访问

```go
func (f *FIFOPolicy) KeyAccessed(ctx context.Context, key string) error
```

记录键被访问，FIFO策略的特点是已存在的键不会改变位置。

**实现逻辑：**

1. 检查键是否已存在
2. 如果已存在，不更新位置（FIFO特性）
3. 如果不存在，添加到队列尾部
4. 检查容量限制，必要时自动淘汰头部节点

**示例：**

```go
err := policy.KeyAccessed(ctx, "user:123")
if err != nil {
    log.Printf("记录访问失败: %v", err)
}
```

#### Evict - 执行淘汰

```go
func (f *FIFOPolicy) Evict(ctx context.Context) (string, error)
```

执行淘汰操作，移除最早添加的键（队列头部）。

**实现逻辑：**

1. 检查队列是否为空
2. 移除队列头部节点
3. 更新头指针
4. 从哈希表中删除对应条目
5. 更新大小计数

**示例：**

```go
evictedKey, err := policy.Evict(ctx)
if err != nil {
    log.Printf("淘汰失败: %v", err)
} else if evictedKey != "" {
    log.Printf("淘汰了键: %s", evictedKey)
}
```

#### Remove - 移除指定键

```go
func (f *FIFOPolicy) Remove(ctx context.Context, key string) error
```

从策略中移除指定的键。

**实现逻辑：**

1. 在哈希表中查找节点
2. 如果是头节点，直接更新头指针
3. 如果不是头节点，遍历找到前驱节点
4. 更新链表连接
5. 从哈希表中删除

### 4. 查询操作

#### Has - 检查键是否存在

```go
func (f *FIFOPolicy) Has(ctx context.Context, key string) (bool, error)
```

#### Size - 获取当前大小

```go
func (f *FIFOPolicy) Size(ctx context.Context) (int, error)
```

#### Clear - 清空策略

```go
func (f *FIFOPolicy) Clear(ctx context.Context) error
```

## 算法原理

### FIFO算法核心思想

1. **先进先出原则**: 最先加入的数据最先被淘汰
2. **插入顺序**: 严格按照数据插入的时间顺序进行淘汰
3. **访问无关**: 数据的访问频率不影响淘汰顺序

### 数据结构选择

1. **单向链表**: 维护插入顺序，支持O(1)头部删除
2. **哈希表**: 提供O(1)查找性能
3. **头尾指针**: 简化队列操作，支持O(1)尾部插入

### 操作复杂度

- **查找**: O(1) - 哈希表查找
- **插入**: O(1) - 尾部插入
- **删除**: O(1) - 头部删除，O(n) - 中间删除
- **淘汰**: O(1) - 头部删除

## 使用示例

### 1. 基本使用

```go
package main

import (
    "context"
    "fmt"
    "log"
    
    "github.com/justinwongcn/hamster/internal/infrastructure/cache"
)

func main() {
    // 创建FIFO策略
    policy := cache.NewFIFOPolicy(3) // 容量限制为3
    
    ctx := context.Background()
    
    // 记录访问
    keys := []string{"key1", "key2", "key3", "key4"}
    for _, key := range keys {
        err := policy.KeyAccessed(ctx, key)
        if err != nil {
            log.Printf("记录访问失败: %v", err)
            continue
        }
        
        size, _ := policy.Size(ctx)
        fmt.Printf("添加 %s，当前大小: %d\n", key, size)
    }
    
    // key1应该被自动淘汰了
    has, _ := policy.Has(ctx, "key1")
    fmt.Printf("key1是否存在: %v\n", has) // 输出: false
    
    has, _ = policy.Has(ctx, "key4")
    fmt.Printf("key4是否存在: %v\n", has) // 输出: true
}
```

### 2. 手动淘汰

```go
func demonstrateManualEviction() {
    policy := cache.NewFIFOPolicy()
    ctx := context.Background()
    
    // 添加一些键
    keys := []string{"user:1", "user:2", "user:3"}
    for _, key := range keys {
        policy.KeyAccessed(ctx, key)
        fmt.Printf("添加: %s\n", key)
    }
    
    // 手动淘汰
    for i := 0; i < 2; i++ {
        evictedKey, err := policy.Evict(ctx)
        if err == nil && evictedKey != "" {
            fmt.Printf("淘汰了: %s\n", evictedKey)
        }
    }
    
    // 检查剩余
    size, _ := policy.Size(ctx)
    fmt.Printf("剩余大小: %d\n", size) // 输出: 1
}
```

### 3. FIFO特性演示

```go
func demonstrateFIFOBehavior() {
    policy := cache.NewFIFOPolicy(3)
    ctx := context.Background()
    
    // 添加初始键
    initialKeys := []string{"a", "b", "c"}
    for _, key := range initialKeys {
        policy.KeyAccessed(ctx, key)
        fmt.Printf("添加: %s\n", key)
    }
    
    // 重复访问已存在的键（不会改变顺序）
    fmt.Println("重复访问 'a'...")
    policy.KeyAccessed(ctx, "a") // 不会改变a的位置
    
    // 添加新键，应该淘汰最早的键
    fmt.Println("添加新键 'd'...")
    policy.KeyAccessed(ctx, "d")
    
    // 检查哪个键被淘汰了
    for _, key := range []string{"a", "b", "c", "d"} {
        has, _ := policy.Has(ctx, key)
        fmt.Printf("%s 是否存在: %v\n", key, has)
    }
    // 输出应该显示 'a' 被淘汰了，即使它刚被访问过
}
```

### 4. 容量管理

```go
func demonstrateCapacityManagement() {
    // 测试不同容量设置
    capacities := []int{0, 2, 5} // 0表示无限制
    
    for _, capacity := range capacities {
        fmt.Printf("\n测试容量: %d\n", capacity)
        
        var policy *cache.FIFOPolicy
        if capacity == 0 {
            policy = cache.NewFIFOPolicy() // 无限制
        } else {
            policy = cache.NewFIFOPolicy(capacity)
        }
        
        ctx := context.Background()
        
        // 添加10个键
        for i := 1; i <= 10; i++ {
            key := fmt.Sprintf("item:%d", i)
            policy.KeyAccessed(ctx, key)
        }
        
        size, _ := policy.Size(ctx)
        fmt.Printf("添加10个键后，实际大小: %d\n", size)
        
        // 检查前几个键是否还存在
        for i := 1; i <= 3; i++ {
            key := fmt.Sprintf("item:%d", i)
            has, _ := policy.Has(ctx, key)
            fmt.Printf("  %s 存在: %v\n", key, has)
        }
    }
}
```

### 5. 并发访问测试

```go
func demonstrateConcurrentAccess() {
    policy := cache.NewFIFOPolicy(100)
    ctx := context.Background()
    
    var wg sync.WaitGroup
    
    // 启动多个goroutine并发访问
    for i := 0; i < 10; i++ {
        wg.Add(1)
        go func(id int) {
            defer wg.Done()
            
            for j := 0; j < 100; j++ {
                key := fmt.Sprintf("goroutine_%d_key_%d", id, j)
                err := policy.KeyAccessed(ctx, key)
                if err != nil {
                    log.Printf("Goroutine %d 访问失败: %v", id, err)
                }
            }
        }(i)
    }
    
    wg.Wait()
    
    size, _ := policy.Size(ctx)
    fmt.Printf("并发访问后大小: %d\n", size)
}
```

### 6. 性能基准测试

```go
func demonstratePerformanceBenchmark() {
    policy := cache.NewFIFOPolicy(1000)
    ctx := context.Background()
    
    // 测试插入性能
    fmt.Println("测试插入性能...")
    start := time.Now()
    
    for i := 0; i < 10000; i++ {
        key := fmt.Sprintf("perf_key_%d", i)
        policy.KeyAccessed(ctx, key)
    }
    
    insertDuration := time.Since(start)
    fmt.Printf("插入10000个键耗时: %v\n", insertDuration)
    fmt.Printf("平均插入时间: %v\n", insertDuration/10000)
    
    // 测试查找性能
    fmt.Println("测试查找性能...")
    start = time.Now()
    
    for i := 0; i < 1000; i++ {
        key := fmt.Sprintf("perf_key_%d", i)
        policy.Has(ctx, key)
    }
    
    lookupDuration := time.Since(start)
    fmt.Printf("查找1000个键耗时: %v\n", lookupDuration)
    fmt.Printf("平均查找时间: %v\n", lookupDuration/1000)
    
    // 测试淘汰性能
    fmt.Println("测试淘汰性能...")
    start = time.Now()
    
    for i := 0; i < 100; i++ {
        policy.Evict(ctx)
    }
    
    evictDuration := time.Since(start)
    fmt.Printf("淘汰100个键耗时: %v\n", evictDuration)
    fmt.Printf("平均淘汰时间: %v\n", evictDuration/100)
}
```

### 7. 与LRU策略对比

```go
func compareFIFOWithLRU() {
    fifoPolicy := cache.NewFIFOPolicy(3)
    lruPolicy := cache.NewLRUPolicy(3)
    ctx := context.Background()
    
    // 相同的访问序列
    accessSequence := []string{"a", "b", "c", "a", "d"}
    
    fmt.Println("FIFO策略:")
    for _, key := range accessSequence {
        fifoPolicy.KeyAccessed(ctx, key)
        fmt.Printf("访问 %s\n", key)
    }
    
    fmt.Println("FIFO策略中剩余的键:")
    for _, key := range []string{"a", "b", "c", "d"} {
        has, _ := fifoPolicy.Has(ctx, key)
        if has {
            fmt.Printf("  %s\n", key)
        }
    }
    
    fmt.Println("\nLRU策略:")
    for _, key := range accessSequence {
        lruPolicy.KeyAccessed(ctx, key)
        fmt.Printf("访问 %s\n", key)
    }
    
    fmt.Println("LRU策略中剩余的键:")
    for _, key := range []string{"a", "b", "c", "d"} {
        has, _ := lruPolicy.Has(ctx, key)
        if has {
            fmt.Printf("  %s\n", key)
        }
    }
    
    // FIFO: 会淘汰最早的'a'，剩余 b, c, d
    // LRU:  会淘汰最少使用的'b'，剩余 a, c, d
}
```

## 性能特性

### 时间复杂度

- **KeyAccessed**: O(1) - 哈希查找 + 链表尾部插入
- **Evict**: O(1) - 链表头部删除
- **Remove**: O(1) - 头部删除，O(n) - 中间删除
- **Has**: O(1) - 哈希查找
- **Size**: O(1) - 直接返回计数

### 空间复杂度

- **存储空间**: O(n) - n为键的数量
- **额外空间**: O(1) - 固定的头尾指针和元数据

### 并发性能

- **读操作**: 使用读锁，支持并发读取
- **写操作**: 使用写锁，保证数据一致性
- **锁粒度**: 整个策略级别的锁

## 适用场景

### 1. 顺序访问模式

```go
// ✅ 适合：日志处理、流式数据
// 数据按时间顺序产生，旧数据不再需要
logPolicy := cache.NewFIFOPolicy(1000)
```

### 2. 缓存预热

```go
// ✅ 适合：批量数据预加载
// 按顺序加载数据，优先淘汰最早加载的
preloadPolicy := cache.NewFIFOPolicy(500)
```

### 3. 简单缓存需求

```go
// ✅ 适合：不需要复杂淘汰逻辑的场景
// 实现简单，性能稳定
simplePolicy := cache.NewFIFOPolicy(100)
```

## 注意事项

### 1. 访问模式考虑

```go
// ✅ 适合：顺序访问模式
// 数据访问具有时间局部性，旧数据不再访问

// ❌ 不适合：随机访问模式
// 频繁访问的数据可能被过早淘汰
```

### 2. 容量设置

```go
// ✅ 推荐：根据内存和访问模式设置合理容量
policy := cache.NewFIFOPolicy(1000) // 根据实际需求设置

// ❌ 避免：容量过小导致频繁淘汰
policy := cache.NewFIFOPolicy(1) // 容量太小

// ❌ 避免：无限制容量可能导致内存泄漏
policy := cache.NewFIFOPolicy() // 在高负载场景下要小心
```

### 3. 并发安全

```go
// ✅ 正确：FIFO策略本身是线程安全的
go func() {
    policy.KeyAccessed(ctx, "key1")
}()

go func() {
    policy.Evict(ctx)
}()

// ❌ 避免：不要在外部加锁
var mu sync.Mutex
mu.Lock()
policy.KeyAccessed(ctx, "key") // 不必要的锁
mu.Unlock()
```

### 4. 性能考虑

```go
// ✅ 推荐：批量操作时考虑性能
// FIFO策略的Remove操作在中间删除时是O(n)的
for _, key := range keysToRemove {
    policy.Remove(ctx, key) // 可能较慢
}

// 考虑使用Clear然后重新添加需要保留的键
if len(keysToRemove) > policy.Size()/2 {
    // 如果要删除的键超过一半，考虑重建
    policy.Clear(ctx)
    for _, key := range keysToKeep {
        policy.KeyAccessed(ctx, key)
    }
}
```

### 5. 监控和调优

```go
// ✅ 推荐：监控FIFO策略的效果
func monitorFIFOPolicy(policy *cache.FIFOPolicy) {
    go func() {
        ticker := time.NewTicker(time.Minute)
        defer ticker.Stop()
        
        for {
            select {
            case <-ticker.C:
                size, _ := policy.Size(context.Background())
                log.Printf("FIFO策略当前大小: %d", size)
                
                // 可以添加更多监控逻辑
                // 比如淘汰频率、命中率等
            }
        }
    }()
}
```
