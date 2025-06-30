# singleflight_peer_picker.go - SingleFlight节点选择器实现

## 文件概述

`singleflight_peer_picker.go`
实现了带SingleFlight优化的节点选择器，结合一致性哈希算法和SingleFlight机制，确保相同键的并发请求只执行一次节点选择操作。该实现提供了高效的节点选择、故障转移和状态管理功能，适用于分布式系统的负载均衡场景。

## 核心功能

### 1. SingleflightPeerPicker 结构体

```go
type SingleflightPeerPicker struct {
    consistentHash domainHash.ConsistentHash // 一致性哈希实现
    peers          map[string]domainHash.Peer // 节点ID到节点实例的映射
    mu             sync.RWMutex               // 保护peers映射
    g              singleflight.Group         // singleflight组
}
```

**设计特点：**

- 结合一致性哈希和SingleFlight优化
- 维护节点ID到节点实例的映射
- 支持节点状态管理和故障转移
- 线程安全的并发访问控制
- 提供丰富的统计和管理接口

### 2. 构造函数

```go
func NewSingleflightPeerPicker(consistentHash domainHash.ConsistentHash) *SingleflightPeerPicker
```

**参数：**

- `consistentHash`: 一致性哈希实现

**示例：**

```go
// 创建一致性哈希
config, _ := domain.NewVirtualNodeConfig(150, nil)
hashRing := infrastructure.NewConsistentHashMap(150, nil)

// 创建SingleFlight节点选择器
picker := NewSingleflightPeerPicker(hashRing)
```

## 主要方法

### 1. 节点选择

#### PickPeer - 选择单个节点

```go
func (p *SingleflightPeerPicker) PickPeer(key string) (domainHash.Peer, error)
```

**执行流程：**

1. 使用SingleFlight确保相同键只执行一次选择
2. 调用一致性哈希获取节点ID
3. 检查节点是否存在和存活
4. 如果节点不可用，选择替代节点

**示例：**

```go
peer, err := picker.PickPeer("user:12345")
if err != nil {
    log.Printf("选择节点失败: %v", err)
    return
}

fmt.Printf("选择的节点: %s (%s)\n", peer.ID(), peer.Address())
```

#### PickPeers - 选择多个节点

```go
func (p *SingleflightPeerPicker) PickPeers(key string, count int) ([]domainHash.Peer, error)
```

**多节点选择逻辑：**

1. 生成唯一的SingleFlight键（包含count信息）
2. 从一致性哈希获取多个节点ID
3. 过滤出存活的节点
4. 返回可用节点列表

**示例：**

```go
// 为数据副本选择3个节点
peers, err := picker.PickPeers("data:67890", 3)
if err != nil {
    log.Printf("选择多个节点失败: %v", err)
    return
}

fmt.Printf("选择的副本节点: ")
for _, peer := range peers {
    fmt.Printf("%s ", peer.ID())
}
fmt.Println()
```

### 2. 节点管理

#### AddPeers - 添加节点

```go
func (p *SingleflightPeerPicker) AddPeers(peers ...domainHash.Peer)
```

**添加逻辑：**

1. 更新节点映射表
2. 将节点ID添加到一致性哈希
3. 线程安全的操作

#### RemovePeers - 移除节点

```go
func (p *SingleflightPeerPicker) RemovePeers(peers ...domainHash.Peer)
```

#### GetAllPeers - 获取所有节点

```go
func (p *SingleflightPeerPicker) GetAllPeers() []domainHash.Peer
```

**示例：**

```go
// 创建节点
peer1, _ := domain.NewPeerInfo("server1", "192.168.1.1:8080", 100)
peer2, _ := domain.NewPeerInfo("server2", "192.168.1.2:8080", 100)
peer3, _ := domain.NewPeerInfo("server3", "192.168.1.3:8080", 100)

// 添加节点
picker.AddPeers(peer1, peer2, peer3)

// 获取所有节点
allPeers := picker.GetAllPeers()
fmt.Printf("总节点数: %d\n", len(allPeers))

// 移除节点
picker.RemovePeers(peer2)
fmt.Printf("移除后节点数: %d\n", picker.GetPeerCount())
```

### 3. 健康检查和状态管理

#### IsHealthy - 检查健康状态

```go
func (p *SingleflightPeerPicker) IsHealthy() (bool, error)
```

**健康检查逻辑：**

1. 检查是否有节点
2. 检查是否有存活的节点
3. 返回健康状态

#### UpdatePeerStatus - 更新节点状态

```go
func (p *SingleflightPeerPicker) UpdatePeerStatus(peerID string, alive bool) error
```

#### GetAlivePeerCount - 获取存活节点数量

```go
func (p *SingleflightPeerPicker) GetAlivePeerCount() int
```

**示例：**

```go
// 检查健康状态
healthy, err := picker.IsHealthy()
if !healthy {
    log.Printf("节点选择器不健康: %v", err)
}

// 模拟节点故障
err = picker.UpdatePeerStatus("server2", false)
if err != nil {
    log.Printf("更新节点状态失败: %v", err)
}

// 检查存活节点
aliveCount := picker.GetAlivePeerCount()
totalCount := picker.GetPeerCount()
fmt.Printf("存活节点: %d/%d\n", aliveCount, totalCount)
```

### 4. SingleFlight管理

#### ForgetKey - 清除SingleFlight缓存

```go
func (p *SingleflightPeerPicker) ForgetKey(key string)
func (p *SingleflightPeerPicker) ForgetMultipleKey(key string, count int)
```

**缓存管理：**

- 清除特定键的SingleFlight缓存
- 支持单节点和多节点选择的缓存清理

**示例：**

```go
// 清除单节点选择缓存
picker.ForgetKey("user:12345")

// 清除多节点选择缓存
picker.ForgetMultipleKey("data:67890", 3)
```

### 5. 内部实现

#### pickPeerInternal - 内部节点选择

```go
func (p *SingleflightPeerPicker) pickPeerInternal(key string) (domainHash.Peer, error)
```

#### pickAlternativePeer - 选择替代节点

```go
func (p *SingleflightPeerPicker) pickAlternativePeer(key, excludePeerID string) (domainHash.Peer, error)
```

**故障转移逻辑：**

1. 获取多个候选节点
2. 排除故障节点
3. 选择第一个可用节点

## 使用示例

### 1. 基本节点选择

```go
package main

import (
    "fmt"
    "log"
    
    "github.com/justinwongcn/hamster/internal/infrastructure/consistent_hash"
    "github.com/justinwongcn/hamster/internal/domain/consistent_hash"
)

func main() {
    // 创建一致性哈希
    hashMap := consistent_hash.NewConsistentHashMap(150, nil)
    
    // 创建SingleFlight节点选择器
    picker := consistent_hash.NewSingleflightPeerPicker(hashMap)
    
    // 创建节点
    peer1, _ := domain.NewPeerInfo("server1", "192.168.1.1:8080", 100)
    peer2, _ := domain.NewPeerInfo("server2", "192.168.1.2:8080", 100)
    peer3, _ := domain.NewPeerInfo("server3", "192.168.1.3:8080", 100)
    
    // 添加节点
    picker.AddPeers(peer1, peer2, peer3)
    
    // 选择节点
    testKeys := []string{"user:1", "user:2", "user:3", "data:a", "data:b"}
    
    for _, key := range testKeys {
        peer, err := picker.PickPeer(key)
        if err != nil {
            log.Printf("选择节点失败: %v", err)
            continue
        }
        
        fmt.Printf("键 %s -> 节点 %s (%s)\n", key, peer.ID(), peer.Address())
    }
}
```

### 2. 故障转移演示

```go
func demonstrateFailover() {
    hashMap := consistent_hash.NewConsistentHashMap(150, nil)
    picker := consistent_hash.NewSingleflightPeerPicker(hashMap)
    
    // 添加节点
    peer1, _ := domain.NewPeerInfo("server1", "192.168.1.1:8080", 100)
    peer2, _ := domain.NewPeerInfo("server2", "192.168.1.2:8080", 100)
    peer3, _ := domain.NewPeerInfo("server3", "192.168.1.3:8080", 100)
    
    picker.AddPeers(peer1, peer2, peer3)
    
    key := "important:data"
    
    // 正常选择
    fmt.Println("正常情况下的节点选择:")
    peer, err := picker.PickPeer(key)
    if err == nil {
        fmt.Printf("选择的节点: %s\n", peer.ID())
    }
    
    // 模拟节点故障
    fmt.Println("模拟节点故障...")
    if peer != nil {
        picker.UpdatePeerStatus(peer.ID(), false)
        fmt.Printf("节点 %s 标记为故障\n", peer.ID())
    }
    
    // 故障转移
    fmt.Println("故障转移后的节点选择:")
    newPeer, err := picker.PickPeer(key)
    if err == nil {
        fmt.Printf("故障转移到节点: %s\n", newPeer.ID())
    } else {
        log.Printf("故障转移失败: %v", err)
    }
    
    // 检查健康状态
    healthy, err := picker.IsHealthy()
    fmt.Printf("系统健康状态: %v\n", healthy)
    if err != nil {
        fmt.Printf("健康检查错误: %v\n", err)
    }
}
```

### 3. 并发访问测试

```go
func demonstrateConcurrentAccess() {
    hashMap := consistent_hash.NewConsistentHashMap(150, nil)
    picker := consistent_hash.NewSingleflightPeerPicker(hashMap)
    
    // 添加节点
    for i := 1; i <= 5; i++ {
        peer, _ := domain.NewPeerInfo(
            fmt.Sprintf("server%d", i),
            fmt.Sprintf("192.168.1.%d:8080", i),
            100,
        )
        picker.AddPeers(peer)
    }
    
    var wg sync.WaitGroup
    results := make(chan string, 100)
    
    // 启动多个goroutine并发选择同一个键
    key := "concurrent:test"
    
    fmt.Printf("启动10个goroutine并发选择键: %s\n", key)
    
    for i := 0; i < 10; i++ {
        wg.Add(1)
        go func(id int) {
            defer wg.Done()
            
            start := time.Now()
            peer, err := picker.PickPeer(key)
            duration := time.Since(start)
            
            if err != nil {
                results <- fmt.Sprintf("Goroutine %d: 错误 - %v (耗时: %v)", id, err, duration)
            } else {
                results <- fmt.Sprintf("Goroutine %d: 选择节点 %s (耗时: %v)", id, peer.ID(), duration)
            }
        }(i)
    }
    
    wg.Wait()
    close(results)
    
    // 输出结果
    for result := range results {
        fmt.Println(result)
    }
    
    // SingleFlight应该确保所有goroutine选择相同的节点
}
```

### 4. 多节点副本选择

```go
func demonstrateReplicaSelection() {
    hashMap := consistent_hash.NewConsistentHashMap(150, nil)
    picker := consistent_hash.NewSingleflightPeerPicker(hashMap)
    
    // 添加足够的节点
    for i := 1; i <= 8; i++ {
        peer, _ := domain.NewPeerInfo(
            fmt.Sprintf("server%d", i),
            fmt.Sprintf("192.168.1.%d:8080", i),
            100,
        )
        picker.AddPeers(peer)
    }
    
    // 为不同的数据选择副本节点
    dataItems := []string{"data:user_profiles", "data:order_history", "data:product_catalog"}
    
    for _, dataKey := range dataItems {
        fmt.Printf("\n为数据 %s 选择副本节点:\n", dataKey)
        
        // 选择3个副本节点
        replicas, err := picker.PickPeers(dataKey, 3)
        if err != nil {
            log.Printf("选择副本失败: %v", err)
            continue
        }
        
        for i, replica := range replicas {
            fmt.Printf("  副本%d: %s (%s)\n", i+1, replica.ID(), replica.Address())
        }
        
        // 模拟一个副本节点故障
        if len(replicas) > 0 {
            failedNode := replicas[0]
            picker.UpdatePeerStatus(failedNode.ID(), false)
            fmt.Printf("  模拟节点 %s 故障\n", failedNode.ID())
            
            // 重新选择副本
            newReplicas, err := picker.PickPeers(dataKey, 3)
            if err == nil {
                fmt.Printf("  故障转移后的副本:\n")
                for i, replica := range newReplicas {
                    fmt.Printf("    副本%d: %s (%s)\n", i+1, replica.ID(), replica.Address())
                }
            }
            
            // 恢复节点状态
            picker.UpdatePeerStatus(failedNode.ID(), true)
        }
    }
}
```

### 5. 性能监控和统计

```go
func demonstrateMonitoring() {
    hashMap := consistent_hash.NewConsistentHashMap(150, nil)
    picker := consistent_hash.NewSingleflightPeerPicker(hashMap)
    
    // 添加节点
    for i := 1; i <= 6; i++ {
        peer, _ := domain.NewPeerInfo(
            fmt.Sprintf("server%d", i),
            fmt.Sprintf("192.168.1.%d:8080", i),
            100,
        )
        picker.AddPeers(peer)
    }
    
    // 启动监控
    go func() {
        ticker := time.NewTicker(5 * time.Second)
        defer ticker.Stop()
        
        for {
            select {
            case <-ticker.C:
                // 基本统计
                totalPeers := picker.GetPeerCount()
                alivePeers := picker.GetAlivePeerCount()
                
                fmt.Printf("节点统计: 总数=%d, 存活=%d\n", totalPeers, alivePeers)
                
                // 健康检查
                healthy, err := picker.IsHealthy()
                if !healthy {
                    log.Printf("系统不健康: %v", err)
                }
                
                // 一致性哈希统计
                stats := picker.GetStats()
                fmt.Printf("哈希环统计: 节点=%d, 虚拟节点=%d, 负载均衡度=%.4f\n",
                    stats.TotalPeers(), stats.VirtualNodes(), stats.LoadBalance())
            }
        }
    }()
    
    // 模拟负载测试
    fmt.Println("开始负载测试...")
    
    start := time.Now()
    successCount := 0
    errorCount := 0
    
    for i := 0; i < 1000; i++ {
        key := fmt.Sprintf("load_test_key_%d", i)
        
        _, err := picker.PickPeer(key)
        if err != nil {
            errorCount++
        } else {
            successCount++
        }
        
        // 随机模拟节点故障和恢复
        if i%100 == 0 && i > 0 {
            peerID := fmt.Sprintf("server%d", (i/100)%6+1)
            alive := i%200 != 0 // 每200次有一次故障
            picker.UpdatePeerStatus(peerID, alive)
        }
    }
    
    duration := time.Since(start)
    
    fmt.Printf("负载测试结果:\n")
    fmt.Printf("  总请求: 1000\n")
    fmt.Printf("  成功: %d\n", successCount)
    fmt.Printf("  失败: %d\n", errorCount)
    fmt.Printf("  成功率: %.2f%%\n", float64(successCount)/1000*100)
    fmt.Printf("  总耗时: %v\n", duration)
    fmt.Printf("  平均延迟: %v\n", duration/1000)
    fmt.Printf("  吞吐量: %.2f ops/sec\n", 1000.0/duration.Seconds())
}
```

### 6. SingleFlight缓存管理

```go
func demonstrateSingleflightManagement() {
    hashMap := consistent_hash.NewConsistentHashMap(150, nil)
    picker := consistent_hash.NewSingleflightPeerPicker(hashMap)
    
    // 添加节点
    peer1, _ := domain.NewPeerInfo("server1", "192.168.1.1:8080", 100)
    picker.AddPeers(peer1)
    
    key := "cache_test_key"
    
    // 第一次选择（会缓存结果）
    fmt.Println("第一次选择节点...")
    peer, err := picker.PickPeer(key)
    if err == nil {
        fmt.Printf("选择的节点: %s\n", peer.ID())
    }
    
    // 添加新节点
    peer2, _ := domain.NewPeerInfo("server2", "192.168.1.2:8080", 100)
    picker.AddPeers(peer2)
    
    // 第二次选择（可能使用缓存的结果）
    fmt.Println("添加新节点后再次选择...")
    peer, err = picker.PickPeer(key)
    if err == nil {
        fmt.Printf("选择的节点: %s\n", peer.ID())
    }
    
    // 清除缓存
    fmt.Println("清除SingleFlight缓存...")
    picker.ForgetKey(key)
    
    // 第三次选择（重新计算）
    fmt.Println("清除缓存后再次选择...")
    peer, err = picker.PickPeer(key)
    if err == nil {
        fmt.Printf("选择的节点: %s\n", peer.ID())
    }
    
    // 多节点选择缓存管理
    fmt.Println("测试多节点选择缓存...")
    peers, err := picker.PickPeers(key, 2)
    if err == nil {
        fmt.Printf("选择的多个节点: ")
        for _, p := range peers {
            fmt.Printf("%s ", p.ID())
        }
        fmt.Println()
    }
    
    // 清除多节点选择缓存
    picker.ForgetMultipleKey(key, 2)
    fmt.Println("多节点选择缓存已清除")
}
```

## 性能特性

### 时间复杂度

- **PickPeer**: O(1) - SingleFlight + 哈希查找
- **PickPeers**: O(k) - k为请求的节点数量
- **AddPeers**: O(n*log(m)) - n为新增节点数，m为虚拟节点数
- **RemovePeers**: O(n*log(m)) - n为移除节点数，m为虚拟节点数

### 空间复杂度

- **节点存储**: O(n) - n为节点数量
- **一致性哈希**: O(n*r) - r为虚拟节点倍数
- **SingleFlight缓存**: O(k) - k为正在处理的键数量

### 性能优势

1. **SingleFlight优化**: 相同键的并发请求只执行一次
2. **故障转移**: 自动选择替代节点
3. **负载均衡**: 基于一致性哈希的均匀分布
4. **状态管理**: 实时的节点健康状态跟踪

## 注意事项

### 1. SingleFlight缓存管理

```go
// ✅ 推荐：在节点变化时清除相关缓存
func (p *SingleflightPeerPicker) AddPeers(peers ...domainHash.Peer) {
    // ... 添加节点逻辑 ...
    
    // 清除可能受影响的缓存
    // 注意：这里需要根据实际情况决定是否清除
}

// ✅ 推荐：定期清理长时间未使用的缓存
go func() {
    ticker := time.NewTicker(time.Hour)
    defer ticker.Stop()
    
    for {
        select {
        case <-ticker.C:
            // 可以实现基于时间的缓存清理逻辑
        }
    }
}()
```

### 2. 节点状态管理

```go
// ✅ 推荐：定期检查节点健康状态
func healthCheck(picker *SingleflightPeerPicker) {
    peers := picker.GetAllPeers()
    for _, peer := range peers {
        alive := checkPeerHealth(peer.Address())
        picker.UpdatePeerStatus(peer.ID(), alive)
    }
}

// ✅ 推荐：监控存活节点数量
aliveCount := picker.GetAlivePeerCount()
if aliveCount < minRequiredNodes {
    log.Printf("警告: 存活节点数量过少 (%d)", aliveCount)
}
```

### 3. 故障转移策略

```go
// ✅ 推荐：设置合理的故障转移逻辑
// 在pickAlternativePeer中最多尝试5个节点
// 可以根据实际需求调整这个数量

// ✅ 推荐：记录故障转移事件
func (p *SingleflightPeerPicker) pickAlternativePeer(key, excludePeerID string) (domainHash.Peer, error) {
    log.Printf("节点 %s 不可用，为键 %s 选择替代节点", excludePeerID, key)
    // ... 故障转移逻辑 ...
}
```

### 4. 性能监控

```go
// ✅ 推荐：监控关键指标
// - 节点选择延迟
// - 故障转移频率
// - SingleFlight命中率
// - 负载均衡度

func monitorPicker(picker *SingleflightPeerPicker) {
    go func() {
        ticker := time.NewTicker(time.Minute)
        defer ticker.Stop()
        
        for {
            select {
            case <-ticker.C:
                stats := picker.GetStats()
                if stats.LoadBalance() > threshold {
                    log.Printf("警告: 负载不均衡度过高: %.4f", stats.LoadBalance())
                }
            }
        }
    }()
}
```

### 5. 内存管理

```go
// ✅ 推荐：避免内存泄漏
// SingleFlight会缓存正在进行的请求
// 长时间运行的选择操作可能导致内存积累

// 设置合理的超时
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

peer, err := picker.PickPeer("some_key")
```
