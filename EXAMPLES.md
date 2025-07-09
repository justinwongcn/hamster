# Hamster 库使用示例

本文档提供了 Hamster 库的详细使用示例，展示如何在实际项目中使用缓存、一致性哈希和分布式锁功能。

## 安装

### 使用 go get 安装

```bash
go get github.com/justinwongcn/hamster
```

### 在项目中导入

```go
import (
    "github.com/justinwongcn/hamster"
    "github.com/justinwongcn/hamster/cache"
    "github.com/justinwongcn/hamster/hash"
    "github.com/justinwongcn/hamster/lock"
)
```

## 快速开始

### 基本示例

```go
package main

import (
    "context"
    "fmt"
    "time"
    
    "github.com/justinwongcn/hamster"
)

func main() {
    // 获取版本信息
    fmt.Printf("Hamster 版本: %s\n", hamster.GetVersion())
    
    // 创建缓存服务
    cacheService, err := hamster.NewCache()
    if err != nil {
        panic(err)
    }
    
    ctx := context.Background()
    
    // 使用缓存
    err = cacheService.Set(ctx, "key", "value", time.Hour)
    if err != nil {
        panic(err)
    }
    
    value, err := cacheService.Get(ctx, "key")
    if err != nil {
        panic(err)
    }
    
    fmt.Printf("缓存值: %v\n", value)
}
```

## 缓存功能示例

### 1. 基本缓存操作

```go
package main

import (
    "context"
    "fmt"
    "log"
    "time"
    
    "github.com/justinwongcn/hamster"
    "github.com/justinwongcn/hamster/cache"
)

func basicCacheExample() {
    // 创建缓存服务，使用自定义配置
    cacheService, err := hamster.NewCache(
        cache.WithMaxMemory(1024*1024), // 1MB
        cache.WithEvictionPolicy("lru"), // LRU 淘汰策略
        cache.WithDefaultExpiration(time.Hour),
        cache.WithCleanupInterval(10*time.Minute),
    )
    if err != nil {
        log.Fatalf("创建缓存服务失败: %v", err)
    }

    ctx := context.Background()

    // 设置不同类型的缓存值
    examples := map[string]interface{}{
        "string_key": "Hello, World!",
        "int_key":    42,
        "struct_key": struct {
            Name string
            Age  int
        }{"John", 30},
        "slice_key": []string{"a", "b", "c"},
        "map_key":   map[string]int{"one": 1, "two": 2},
    }

    // 批量设置缓存
    for key, value := range examples {
        err := cacheService.Set(ctx, key, value, time.Hour)
        if err != nil {
            log.Printf("设置缓存 %s 失败: %v", key, err)
            continue
        }
        fmt.Printf("✓ 设置缓存: %s\n", key)
    }

    // 批量获取缓存
    for key := range examples {
        value, err := cacheService.Get(ctx, key)
        if err != nil {
            log.Printf("获取缓存 %s 失败: %v", key, err)
            continue
        }
        fmt.Printf("✓ 获取缓存 %s: %v\n", key, value)
    }

    // 获取缓存统计信息
    stats, err := cacheService.Stats(ctx)
    if err != nil {
        log.Printf("获取统计信息失败: %v", err)
    } else {
        fmt.Printf("缓存统计: 命中率=%.2f%%, 项目数=%d\n", 
            stats.HitRate*100, stats.ItemCount)
    }

    // 删除缓存
    err = cacheService.Delete(ctx, "string_key")
    if err != nil {
        log.Printf("删除缓存失败: %v", err)
    } else {
        fmt.Println("✓ 缓存删除成功")
    }

    // 获取并删除
    value, err := cacheService.LoadAndDelete(ctx, "int_key")
    if err != nil {
        log.Printf("获取并删除失败: %v", err)
    } else {
        fmt.Printf("✓ 获取并删除成功: %v\n", value)
    }
}
```

### 2. 读透缓存示例

```go
func readThroughCacheExample() {
    // 创建读透缓存服务
    readThroughCache, err := hamster.NewReadThroughCache(
        cache.WithMaxMemory(512*1024),
        cache.WithEvictionPolicy("lru"),
    )
    if err != nil {
        log.Fatalf("创建读透缓存失败: %v", err)
    }

    ctx := context.Background()

    // 模拟数据库查询函数
    userLoader := func(ctx context.Context, userID string) (any, error) {
        fmt.Printf("从数据库加载用户: %s\n", userID)
        // 模拟数据库查询延迟
        time.Sleep(100 * time.Millisecond)
        
        return map[string]interface{}{
            "id":    userID,
            "name":  fmt.Sprintf("User_%s", userID),
            "email": fmt.Sprintf("user_%s@example.com", userID),
        }, nil
    }

    // 第一次获取（会触发加载器）
    user1, err := readThroughCache.GetWithLoader(
        ctx, "user:123", userLoader, time.Hour)
    if err != nil {
        log.Printf("获取用户失败: %v", err)
    } else {
        fmt.Printf("✓ 第一次获取用户: %v\n", user1)
    }

    // 第二次获取（从缓存获取，不会触发加载器）
    user2, err := readThroughCache.GetWithLoader(
        ctx, "user:123", userLoader, time.Hour)
    if err != nil {
        log.Printf("获取用户失败: %v", err)
    } else {
        fmt.Printf("✓ 第二次获取用户: %v\n", user2)
    }
}
```

### 3. 高级缓存配置

```go
func advancedCacheExample() {
    // 使用配置结构体创建缓存
    config := &cache.Config{
        MaxMemory:         2 * 1024 * 1024, // 2MB
        DefaultExpiration: 30 * time.Minute,
        CleanupInterval:   5 * time.Minute,
        EvictionPolicy:    "lru",
        EnableBloomFilter: true,
        BloomFilterFalsePositiveRate: 0.01,
    }

    cacheService, err := cache.NewServiceWithConfig(config)
    if err != nil {
        log.Fatalf("创建缓存服务失败: %v", err)
    }

    ctx := context.Background()

    // 设置淘汰回调函数
    cacheService.OnEvicted(func(key string, val any) {
        fmt.Printf("缓存项被淘汰: key=%s, value=%v\n", key, val)
    })

    // 测试大量数据，触发淘汰
    for i := 0; i < 1000; i++ {
        key := fmt.Sprintf("key_%d", i)
        value := fmt.Sprintf("large_value_%d_%s", i, 
            string(make([]byte, 1024))) // 1KB 数据
        
        err := cacheService.Set(ctx, key, value, time.Hour)
        if err != nil {
            log.Printf("设置缓存失败: %v", err)
        }
    }

    fmt.Println("✓ 大量数据写入完成")
}
```

## 一致性哈希示例

### 1. 基本节点管理

```go
func consistentHashExample() {
    // 创建一致性哈希服务
    hashService, err := hamster.NewConsistentHash(
        hash.WithReplicas(150), // 每个节点150个虚拟节点
        hash.WithSingleflight(true), // 启用单飞模式
    )
    if err != nil {
        log.Fatalf("创建一致性哈希服务失败: %v", err)
    }

    ctx := context.Background()

    // 添加服务器节点
    servers := []hash.Peer{
        {ID: "server1", Address: "192.168.1.1:8080", Weight: 100},
        {ID: "server2", Address: "192.168.1.2:8080", Weight: 100},
        {ID: "server3", Address: "192.168.1.3:8080", Weight: 150}, // 更高权重
        {ID: "server4", Address: "192.168.1.4:8080", Weight: 100},
    }

    err = hashService.AddPeers(ctx, servers)
    if err != nil {
        log.Printf("添加节点失败: %v", err)
        return
    }
    fmt.Printf("✓ 成功添加 %d 个服务器节点\n", len(servers))

    // 测试键分布
    keys := []string{
        "user:123", "user:456", "user:789",
        "session:abc", "session:def", "session:ghi",
        "cache:data1", "cache:data2", "cache:data3",
    }

    distribution := make(map[string]int)
    
    for _, key := range keys {
        peer, err := hashService.SelectPeer(ctx, key)
        if err != nil {
            log.Printf("选择节点失败: %v", err)
            continue
        }
        
        distribution[peer.ID]++
        fmt.Printf("键 %-12s -> 节点 %s (%s)\n", 
            key, peer.ID, peer.Address)
    }

    // 显示分布统计
    fmt.Println("\n节点分布统计:")
    for serverID, count := range distribution {
        fmt.Printf("节点 %s: %d 个键\n", serverID, count)
    }
}
```

### 2. 节点故障处理

```go
func nodeFailureExample() {
    hashService, err := hamster.NewConsistentHash()
    if err != nil {
        log.Fatalf("创建服务失败: %v", err)
    }

    ctx := context.Background()

    // 初始节点
    initialPeers := []hash.Peer{
        {ID: "server1", Address: "192.168.1.1:8080", Weight: 100},
        {ID: "server2", Address: "192.168.1.2:8080", Weight: 100},
        {ID: "server3", Address: "192.168.1.3:8080", Weight: 100},
    }

    err = hashService.AddPeers(ctx, initialPeers)
    if err != nil {
        log.Printf("添加初始节点失败: %v", err)
        return
    }

    testKey := "important_data"
    
    // 故障前的节点选择
    peer1, err := hashService.SelectPeer(ctx, testKey)
    if err != nil {
        log.Printf("选择节点失败: %v", err)
        return
    }
    fmt.Printf("故障前: 键 %s -> 节点 %s\n", testKey, peer1.ID)

    // 模拟节点故障，移除 server2
    err = hashService.RemovePeer(ctx, "server2")
    if err != nil {
        log.Printf("移除节点失败: %v", err)
        return
    }
    fmt.Println("✓ 模拟节点 server2 故障，已移除")

    // 故障后的节点选择
    peer2, err := hashService.SelectPeer(ctx, testKey)
    if err != nil {
        log.Printf("选择节点失败: %v", err)
        return
    }
    fmt.Printf("故障后: 键 %s -> 节点 %s\n", testKey, peer2.ID)

    // 添加新节点
    newPeer := hash.Peer{
        ID: "server4", Address: "192.168.1.4:8080", Weight: 100}
    err = hashService.AddPeer(ctx, newPeer)
    if err != nil {
        log.Printf("添加新节点失败: %v", err)
        return
    }
    fmt.Println("✓ 添加新节点 server4")

    // 新节点加入后的选择
    peer3, err := hashService.SelectPeer(ctx, testKey)
    if err != nil {
        log.Printf("选择节点失败: %v", err)
        return
    }
    fmt.Printf("新节点后: 键 %s -> 节点 %s\n", testKey, peer3.ID)
}
```

### 3. 多副本选择

```go
func multiReplicaExample() {
    hashService, err := hamster.NewConsistentHash(
        hash.WithReplicas(100),
    )
    if err != nil {
        log.Fatalf("创建服务失败: %v", err)
    }

    ctx := context.Background()

    // 添加多个节点
    peers := []hash.Peer{
        {ID: "node1", Address: "192.168.1.1:8080", Weight: 100},
        {ID: "node2", Address: "192.168.1.2:8080", Weight: 100},
        {ID: "node3", Address: "192.168.1.3:8080", Weight: 100},
        {ID: "node4", Address: "192.168.1.4:8080", Weight: 100},
        {ID: "node5", Address: "192.168.1.5:8080", Weight: 100},
    }

    err = hashService.AddPeers(ctx, peers)
    if err != nil {
        log.Printf("添加节点失败: %v", err)
        return
    }

    // 为数据选择多个副本节点
    dataKey := "critical_data"
    replicaCount := 3

    selectedPeers, err := hashService.SelectPeers(ctx, dataKey, replicaCount)
    if err != nil {
        log.Printf("选择副本节点失败: %v", err)
        return
    }

    fmt.Printf("为键 %s 选择 %d 个副本节点:\n", dataKey, replicaCount)
    for i, peer := range selectedPeers {
        fmt.Printf("  副本 %d: 节点 %s (%s)\n", 
            i+1, peer.ID, peer.Address)
    }

    // 获取统计信息
    stats, err := hashService.GetStats(ctx)
    if err != nil {
        log.Printf("获取统计失败: %v", err)
        return
    }

    fmt.Printf("\n哈希环统计:\n")
    fmt.Printf("  总节点数: %d\n", stats.TotalPeers)
    fmt.Printf("  虚拟节点数: %d\n", stats.VirtualNodes)
    fmt.Printf("  副本倍数: %d\n", stats.Replicas)
    fmt.Printf("  负载均衡度: %.2f\n", stats.LoadBalance)
}
```

## 分布式锁示例

### 1. 基本锁操作

```go
func basicLockExample() {
    // 创建分布式锁服务
    lockService, err := hamster.NewDistributedLock(
        lock.WithDefaultExpiration(30*time.Second),
        lock.WithDefaultTimeout(5*time.Second),
        lock.WithDefaultRetry(lock.RetryTypeExponential, 3, 100*time.Millisecond),
    )
    if err != nil {
        log.Fatalf("创建锁服务失败: %v", err)
    }

    ctx := context.Background()

    // 尝试获取锁（不重试）
    lockInfo, err := lockService.TryLock(ctx, "resource:database")
    if err != nil {
        log.Printf("获取锁失败: %v", err)
        return
    }

    fmt.Printf("✓ 成功获取锁:\n")
    fmt.Printf("  键: %s\n", lockInfo.Key)
    fmt.Printf("  值: %s\n", lockInfo.Value)
    fmt.Printf("  创建时间: %s\n", lockInfo.CreatedAt.Format("15:04:05"))
    fmt.Printf("  过期时间: %s\n", lockInfo.ExpiresAt.Format("15:04:05"))
    fmt.Printf("  是否有效: %t\n", lockInfo.IsValid)

    // 模拟业务处理
    fmt.Println("正在处理关键业务...")
    time.Sleep(2 * time.Second)

    fmt.Println("✓ 业务处理完成")
    
    // 注意：当前实现中 Unlock 等方法暂未完全实现
    // 在实际使用中，锁会在过期时间后自动释放
}
```

### 2. 带重试的锁获取

```go
func lockWithRetryExample() {
    lockService, err := hamster.NewDistributedLock()
    if err != nil {
        log.Fatalf("创建锁服务失败: %v", err)
    }

    ctx := context.Background()

    // 使用自定义选项获取锁
    options := lock.LockOptions{
        Expiration: 60 * time.Second,
        Timeout:    10 * time.Second,
        RetryType:  lock.RetryTypeExponential,
        RetryCount: 5,
        RetryBase:  200 * time.Millisecond,
    }

    fmt.Println("尝试获取锁（支持重试）...")
    start := time.Now()
    
    lockInfo, err := lockService.Lock(ctx, "resource:critical_section", options)
    if err != nil {
        log.Printf("获取锁失败: %v", err)
        return
    }

    elapsed := time.Since(start)
    fmt.Printf("✓ 成功获取锁，耗时: %v\n", elapsed)
    fmt.Printf("  锁键: %s\n", lockInfo.Key)
    fmt.Printf("  锁值: %s\n", lockInfo.Value)
}
```

### 3. 并发锁测试

```go
func concurrentLockExample() {
    lockService, err := hamster.NewDistributedLock(
        lock.WithDefaultExpiration(10*time.Second),
    )
    if err != nil {
        log.Fatalf("创建锁服务失败: %v", err)
    }

    const goroutineCount = 5
    const resourceKey = "shared_resource"

    var wg sync.WaitGroup
    results := make(chan string, goroutineCount)

    // 启动多个 goroutine 竞争同一个锁
    for i := 0; i < goroutineCount; i++ {
        wg.Add(1)
        go func(id int) {
            defer wg.Done()
            
            ctx := context.Background()
            workerName := fmt.Sprintf("Worker-%d", id)
            
            fmt.Printf("%s: 尝试获取锁...\n", workerName)
            
            lockInfo, err := lockService.TryLock(ctx, resourceKey)
            if err != nil {
                results <- fmt.Sprintf("%s: 获取锁失败 - %v", workerName, err)
                return
            }
            
            results <- fmt.Sprintf("%s: 成功获取锁 - %s", 
                workerName, lockInfo.Value)
            
            // 模拟工作
            time.Sleep(2 * time.Second)
            
            results <- fmt.Sprintf("%s: 工作完成", workerName)
        }(i)
    }

    // 等待所有 goroutine 完成
    go func() {
        wg.Wait()
        close(results)
    }()

    // 收集结果
    fmt.Println("\n并发锁测试结果:")
    for result := range results {
        fmt.Println(result)
    }
}
```

## 完整应用示例

### Web 缓存服务

```go
package main

import (
    "context"
    "encoding/json"
    "fmt"
    "log"
    "net/http"
    "time"
    
    "github.com/justinwongcn/hamster"
    "github.com/justinwongcn/hamster/cache"
)

type User struct {
    ID    string `json:"id"`
    Name  string `json:"name"`
    Email string `json:"email"`
}

type UserService struct {
    cache *cache.Service
}

func NewUserService() *UserService {
    cacheService, err := hamster.NewCache(
        cache.WithMaxMemory(10*1024*1024), // 10MB
        cache.WithEvictionPolicy("lru"),
        cache.WithDefaultExpiration(time.Hour),
    )
    if err != nil {
        log.Fatalf("创建缓存失败: %v", err)
    }
    
    return &UserService{cache: cacheService}
}

func (s *UserService) GetUser(ctx context.Context, userID string) (*User, error) {
    cacheKey := fmt.Sprintf("user:%s", userID)
    
    // 尝试从缓存获取
    if cached, err := s.cache.Get(ctx, cacheKey); err == nil {
        if user, ok := cached.(*User); ok {
            fmt.Printf("缓存命中: %s\n", userID)
            return user, nil
        }
    }
    
    // 缓存未命中，从数据库加载
    fmt.Printf("缓存未命中，从数据库加载: %s\n", userID)
    user := &User{
        ID:    userID,
        Name:  fmt.Sprintf("User_%s", userID),
        Email: fmt.Sprintf("user_%s@example.com", userID),
    }
    
    // 存入缓存
    err := s.cache.Set(ctx, cacheKey, user, time.Hour)
    if err != nil {
        log.Printf("缓存设置失败: %v", err)
    }
    
    return user, nil
}

func (s *UserService) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    userID := r.URL.Query().Get("id")
    if userID == "" {
        http.Error(w, "缺少用户ID", http.StatusBadRequest)
        return
    }
    
    user, err := s.GetUser(r.Context(), userID)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(user)
}

func webCacheExample() {
    userService := NewUserService()
    
    http.Handle("/user", userService)
    
    fmt.Println("Web 缓存服务启动在 :8080")
    fmt.Println("访问: http://localhost:8080/user?id=123")
    
    log.Fatal(http.ListenAndServe(":8080", nil))
}
```

## 最佳实践

### 1. 错误处理

```go
func errorHandlingExample() {
    cacheService, err := hamster.NewCache()
    if err != nil {
        log.Fatalf("创建缓存失败: %v", err)
    }
    
    ctx := context.Background()
    
    // 设置缓存时的错误处理
    if err := cacheService.Set(ctx, "key", "value", time.Hour); err != nil {
        log.Printf("设置缓存失败: %v", err)
        // 根据业务需求决定是否继续执行
    }
    
    // 获取缓存时的错误处理
    value, err := cacheService.Get(ctx, "key")
    if err != nil {
        log.Printf("获取缓存失败: %v", err)
        // 可以从其他数据源获取数据
        value = "default_value"
    }
    
    fmt.Printf("最终值: %v\n", value)
}
```

### 2. 上下文使用

```go
func contextExample() {
    cacheService, _ := hamster.NewCache()
    
    // 使用带超时的上下文
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    
    // 所有操作都使用带超时的上下文
    err := cacheService.Set(ctx, "key", "value", time.Hour)
    if err != nil {
        if ctx.Err() == context.DeadlineExceeded {
            log.Println("操作超时")
        } else {
            log.Printf("操作失败: %v", err)
        }
    }
}
```

### 3. 资源清理

```go
func resourceCleanupExample() {
    // 虽然当前版本没有显式的 Close 方法，
    // 但建议在应用关闭时进行适当的清理
    
    cacheService, _ := hamster.NewCache()
    
    // 在应用关闭时清理缓存
    defer func() {
        ctx := context.Background()
        if err := cacheService.Clear(ctx); err != nil {
            log.Printf("清理缓存失败: %v", err)
        }
    }()
    
    // 应用逻辑...
}
```

## 总结

Hamster 库提供了简洁而强大的缓存、一致性哈希和分布式锁功能。通过这些示例，您可以：

1. **快速上手**: 使用默认配置快速创建服务
2. **灵活配置**: 根据需求自定义各种参数
3. **错误处理**: 正确处理各种异常情况
4. **最佳实践**: 遵循推荐的使用模式

更多详细信息请参考：
- [公共 API 文档](PUBLIC_API.md)
- [重构总结](REFACTOR_SUMMARY.md)
- [测试覆盖率报告](TEST_COVERAGE_REPORT.md)
