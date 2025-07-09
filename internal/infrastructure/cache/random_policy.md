# random_policy.go - 随机淘汰策略实现

## 文件概述

`random_policy.go` 实现了随机淘汰策略，当需要淘汰缓存项时随机选择一个键进行淘汰。该实现展示了如何实现EvictionPolicy接口，提供了简单而有效的淘汰算法，适用于对淘汰顺序没有特殊要求的场景。

## 核心功能

### 1. RandomPolicy 结构体

```go
type RandomPolicy struct {
    capacity int            // 容量限制，0表示无限制
    keys     []string       // 存储所有key的切片
    keySet   map[string]int // key到索引的映射，用于快速查找
    mutex    sync.RWMutex   // 读写锁，保证并发安全
    rand     *rand.Rand     // 随机数生成器
}
```

**设计特点：**

- 使用切片存储键的顺序
- 使用map提供O(1)查找性能
- 独立的随机数生成器确保线程安全
- 支持容量限制和无限制模式
- 高效的删除操作（交换到末尾）

### 2. 构造函数

#### NewRandomPolicy - 创建随机策略实例

```go
func NewRandomPolicy(capacity ...int) *RandomPolicy
```

**参数：**

- `capacity`: 可选参数，容量限制，0或不传表示无限制

**特性：**

- 使用当前时间作为随机种子
- 初始化空的键集合
- 支持可选的容量限制

**示例：**

```go
// 无限制容量
policy := NewRandomPolicy()

// 限制容量为100
policy := NewRandomPolicy(100)
```

## 主要方法

### 1. 核心操作

#### KeyAccessed - 记录键访问

```go
func (r *RandomPolicy) KeyAccessed(ctx context.Context, key string) error
```

**实现逻辑：**

1. 检查键是否已存在
2. 如果不存在，添加到键集合
3. 检查容量限制，必要时自动淘汰

**特点：**

- 随机策略不关心访问顺序
- 已存在的键不需要更新位置
- 自动容量管理

**示例：**

```go
err := policy.KeyAccessed(ctx, "user:123")
if err != nil {
    log.Printf("记录访问失败: %v", err)
}
```

#### Evict - 执行淘汰

```go
func (r *RandomPolicy) Evict(ctx context.Context) (string, error)
```

**实现逻辑：**

1. 检查是否有键可淘汰
2. 随机选择一个索引
3. 移除对应的键
4. 返回被淘汰的键

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
func (r *RandomPolicy) Remove(ctx context.Context, key string) error
```

**高效删除算法：**

1. 找到键的索引位置
2. 将最后一个元素移动到该位置
3. 删除最后一个元素
4. 更新索引映射

### 2. 查询操作

#### Has - 检查键是否存在

```go
func (r *RandomPolicy) Has(ctx context.Context, key string) (bool, error)
```

#### Size - 获取当前大小

```go
func (r *RandomPolicy) Size(ctx context.Context) (int, error)
```

#### Clear - 清空策略

```go
func (r *RandomPolicy) Clear(ctx context.Context) error
```

## 算法原理

### 随机淘汰算法特点

1. **公平性**: 每个键被淘汰的概率相等
2. **简单性**: 实现简单，逻辑清晰
3. **性能**: O(1)的淘汰操作
4. **无偏性**: 不受访问模式影响

### 数据结构选择

1. **切片**: 存储键的集合，支持随机访问
2. **映射**: 提供O(1)的查找性能
3. **交换删除**: 避免数组移动，保持O(1)删除

### 操作复杂度

- **添加**: O(1) - 直接追加到切片末尾
- **查找**: O(1) - 哈希表查找
- **删除**: O(1) - 交换到末尾删除
- **淘汰**: O(1) - 随机选择删除

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
    // 创建随机策略
    policy := cache.NewRandomPolicy(3) // 容量限制为3
    
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
    
    // key4的添加会触发自动淘汰
    // 检查哪些键还存在
    for _, key := range keys {
        has, _ := policy.Has(ctx, key)
        fmt.Printf("%s 是否存在: %v\n", key, has)
    }
}
```

### 2. 手动淘汰演示

```go
func demonstrateManualEviction() {
    policy := cache.NewRandomPolicy()
    ctx := context.Background()
    
    // 添加一些键
    keys := []string{"user:1", "user:2", "user:3", "user:4", "user:5"}
    for _, key := range keys {
        policy.KeyAccessed(ctx, key)
        fmt.Printf("添加: %s\n", key)
    }
    
    // 手动淘汰几个键
    fmt.Println("开始随机淘汰:")
    for i := 0; i < 3; i++ {
        evictedKey, err := policy.Evict(ctx)
        if err == nil && evictedKey != "" {
            fmt.Printf("第%d次淘汰: %s\n", i+1, evictedKey)
        }
    }
    
    // 检查剩余键
    size, _ := policy.Size(ctx)
    fmt.Printf("剩余键数量: %d\n", size)
    
    for _, key := range keys {
        has, _ := policy.Has(ctx, key)
        if has {
            fmt.Printf("剩余键: %s\n", key)
        }
    }
}
```

### 3. 随机性验证

```go
func demonstrateRandomness() {
    policy := cache.NewRandomPolicy()
    ctx := context.Background()
    
    // 添加键
    keys := []string{"A", "B", "C", "D", "E"}
    for _, key := range keys {
        policy.KeyAccessed(ctx, key)
    }
    
    // 多次淘汰测试随机性
    evictionCount := make(map[string]int)
    testRounds := 1000
    
    for round := 0; round < testRounds; round++ {
        // 重置策略
        policy.Clear(ctx)
        for _, key := range keys {
            policy.KeyAccessed(ctx, key)
        }
        
        // 淘汰一个键
        evictedKey, err := policy.Evict(ctx)
        if err == nil && evictedKey != "" {
            evictionCount[evictedKey]++
        }
    }
    
    // 分析随机性
    fmt.Printf("随机性测试结果 (%d 轮):\n", testRounds)
    expectedCount := float64(testRounds) / float64(len(keys))
    
    for _, key := range keys {
        count := evictionCount[key]
        percentage := float64(count) / float64(testRounds) * 100
        deviation := float64(count) - expectedCount
        
        fmt.Printf("键 %s: %d 次 (%.1f%%), 偏差: %.1f\n", 
            key, count, percentage, deviation)
    }
}
```

### 4. 性能基准测试

```go
func demonstratePerformanceBenchmark() {
    policy := cache.NewRandomPolicy(1000)
    ctx := context.Background()
    
    // 测试添加性能
    fmt.Println("测试添加性能...")
    start := time.Now()
    
    for i := 0; i < 10000; i++ {
        key := fmt.Sprintf("key_%d", i)
        policy.KeyAccessed(ctx, key)
    }
    
    addDuration := time.Since(start)
    fmt.Printf("添加10000个键耗时: %v\n", addDuration)
    fmt.Printf("平均添加时间: %v\n", addDuration/10000)
    
    // 测试查找性能
    fmt.Println("测试查找性能...")
    start = time.Now()
    
    for i := 0; i < 1000; i++ {
        key := fmt.Sprintf("key_%d", rand.Intn(1000))
        policy.Has(ctx, key)
    }
    
    lookupDuration := time.Since(start)
    fmt.Printf("查找1000次耗时: %v\n", lookupDuration)
    fmt.Printf("平均查找时间: %v\n", lookupDuration/1000)
    
    // 测试淘汰性能
    fmt.Println("测试淘汰性能...")
    start = time.Now()
    
    for i := 0; i < 100; i++ {
        policy.Evict(ctx)
    }
    
    evictDuration := time.Since(start)
    fmt.Printf("淘汰100次耗时: %v\n", evictDuration)
    fmt.Printf("平均淘汰时间: %v\n", evictDuration/100)
}
```

### 5. 并发访问测试

```go
func demonstrateConcurrentAccess() {
    policy := cache.NewRandomPolicy(100)
    ctx := context.Background()
    
    var wg sync.WaitGroup
    
    // 启动多个写入goroutine
    for i := 0; i < 5; i++ {
        wg.Add(1)
        go func(id int) {
            defer wg.Done()
            
            for j := 0; j < 100; j++ {
                key := fmt.Sprintf("writer_%d_key_%d", id, j)
                err := policy.KeyAccessed(ctx, key)
                if err != nil {
                    log.Printf("写入失败: %v", err)
                }
            }
        }(i)
    }
    
    // 启动多个读取goroutine
    for i := 0; i < 3; i++ {
        wg.Add(1)
        go func(id int) {
            defer wg.Done()
            
            for j := 0; j < 200; j++ {
                key := fmt.Sprintf("writer_%d_key_%d", rand.Intn(5), rand.Intn(100))
                
                has, err := policy.Has(ctx, key)
                if err == nil && has {
                    fmt.Printf("读取器%d找到键: %s\n", id, key)
                }
                
                time.Sleep(time.Millisecond)
            }
        }(i)
    }
    
    // 启动淘汰goroutine
    wg.Add(1)
    go func() {
        defer wg.Done()
        
        for i := 0; i < 50; i++ {
            evictedKey, err := policy.Evict(ctx)
            if err == nil && evictedKey != "" {
                fmt.Printf("淘汰了键: %s\n", evictedKey)
            }
            
            time.Sleep(10 * time.Millisecond)
        }
    }()
    
    wg.Wait()
    
    size, _ := policy.Size(ctx)
    fmt.Printf("并发测试完成，最终大小: %d\n", size)
}
```

### 6. 与其他策略对比

```go
func compareWithOtherPolicies() {
    ctx := context.Background()
    capacity := 5
    
    // 创建不同策略
    randomPolicy := cache.NewRandomPolicy(capacity)
    lruPolicy := cache.NewLRUPolicy(capacity)
    fifoPolicy := cache.NewFIFOPolicy(capacity)
    
    policies := []cache.EvictionPolicy{randomPolicy, lruPolicy, fifoPolicy}
    policyNames := []string{"Random", "LRU", "FIFO"}
    
    // 相同的访问序列
    accessSequence := []string{"A", "B", "C", "D", "E", "A", "F", "G"}
    
    for i, policy := range policies {
        fmt.Printf("%s 策略测试:\n", policyNames[i])
        
        for _, key := range accessSequence {
            fmt.Printf("  访问: %s\n", key)
            
            err := policy.KeyAccessed(ctx, key)
            if err != nil {
                log.Printf("访问失败: %v", err)
                continue
            }
            
            size, _ := policy.Size(ctx)
            fmt.Printf("    当前大小: %d\n", size)
        }
        
        // 显示最终状态
        fmt.Printf("  最终状态:\n")
        for _, key := range []string{"A", "B", "C", "D", "E", "F", "G"} {
            has, _ := policy.Has(ctx, key)
            if has {
                fmt.Printf("    保留: %s\n", key)
            }
        }
        
        fmt.Println()
    }
}
```

### 7. 容量管理演示

```go
func demonstrateCapacityManagement() {
    // 测试不同容量设置
    capacities := []int{0, 3, 10} // 0表示无限制
    
    for _, capacity := range capacities {
        fmt.Printf("测试容量: %d\n", capacity)
        
        var policy *cache.RandomPolicy
        if capacity == 0 {
            policy = cache.NewRandomPolicy() // 无限制
        } else {
            policy = cache.NewRandomPolicy(capacity)
        }
        
        ctx := context.Background()
        
        // 添加超过容量的键
        for i := 1; i <= 15; i++ {
            key := fmt.Sprintf("item:%d", i)
            err := policy.KeyAccessed(ctx, key)
            if err != nil {
                log.Printf("添加键失败: %v", err)
            }
            
            size, _ := policy.Size(ctx)
            fmt.Printf("  添加 %s，当前大小: %d\n", key, size)
            
            if capacity > 0 && size > capacity {
                fmt.Printf("  警告: 大小超过容量限制\n")
            }
        }
        
        finalSize, _ := policy.Size(ctx)
        fmt.Printf("最终大小: %d\n", finalSize)
        fmt.Println()
    }
}
```

## 性能特性

### 时间复杂度

- **KeyAccessed**: O(1) - 哈希查找 + 切片追加
- **Evict**: O(1) - 随机选择 + 交换删除
- **Remove**: O(1) - 哈希查找 + 交换删除
- **Has**: O(1) - 哈希查找
- **Size**: O(1) - 直接返回长度

### 空间复杂度

- **存储空间**: O(n) - n为键的数量
- **额外空间**: O(n) - 索引映射表

### 并发性能

- **读操作**: 使用读锁，支持并发读取
- **写操作**: 使用写锁，保证数据一致性
- **随机数生成**: 独立实例，避免全局锁竞争

## 适用场景

### 1. 无特殊要求的缓存

```go
// ✅ 适合：对淘汰顺序没有特殊要求
// 简单、公平的淘汰策略
policy := NewRandomPolicy(1000)
```

### 2. 避免最坏情况

```go
// ✅ 适合：避免LRU/FIFO的最坏情况
// 随机策略不会被特定访问模式影响
policy := NewRandomPolicy(500)
```

### 3. 性能敏感场景

```go
// ✅ 适合：需要稳定O(1)性能
// 所有操作都是O(1)时间复杂度
policy := NewRandomPolicy(100)
```

## 注意事项

### 1. 随机种子

```go
// ✅ 推荐：使用时间作为种子（已在构造函数中实现）
rand := rand.New(rand.NewSource(time.Now().UnixNano()))

// ❌ 避免：使用固定种子（测试除外）
rand := rand.New(rand.NewSource(1)) // 会产生相同的序列
```

### 2. 线程安全

```go
// ✅ 推荐：RandomPolicy本身是线程安全的
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

### 3. 容量设置

```go
// ✅ 推荐：根据内存和性能要求设置容量
policy := NewRandomPolicy(1000) // 合理的容量

// ❌ 避免：容量过小导致频繁淘汰
policy := NewRandomPolicy(1) // 容量太小

// ✅ 推荐：无限制容量需要外部控制
policy := NewRandomPolicy() // 确保有其他机制控制大小
```

### 4. 性能监控

```go
// ✅ 推荐：监控策略性能
func monitorRandomPolicy(policy *cache.RandomPolicy) {
    go func() {
        ticker := time.NewTicker(time.Minute)
        defer ticker.Stop()
        
        for {
            select {
            case <-ticker.C:
                size, _ := policy.Size(context.Background())
                log.Printf("随机策略当前大小: %d", size)
            }
        }
    }()
}
```

### 5. 测试验证

```go
// ✅ 推荐：验证随机性
func testRandomness(policy *cache.RandomPolicy) {
    // 多次测试验证淘汰的随机性
    // 确保没有明显的偏向性
}

// ✅ 推荐：性能基准测试
func benchmarkRandomPolicy(b *testing.B) {
    policy := cache.NewRandomPolicy(1000)
    ctx := context.Background()
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        key := fmt.Sprintf("key_%d", i)
        policy.KeyAccessed(ctx, key)
    }
}
```

RandomPolicy提供了简单而有效的随机淘汰策略，适用于对淘汰顺序没有特殊要求且需要稳定性能的场景。
