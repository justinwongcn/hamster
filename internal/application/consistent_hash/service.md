# service.go - 一致性哈希应用服务

## 文件概述

`service.go` 实现了一致性哈希应用服务层，遵循DDD架构模式，协调领域服务和基础设施层，实现具体的一致性哈希业务用例。该文件提供了完整的节点选择、集群管理、健康检查等功能，为分布式系统的负载均衡和数据分片提供了高级抽象。

## 核心功能

### 1. ConsistentHashApplicationService 应用服务

```go
type ConsistentHashApplicationService struct {
    peerPicker domainHash.PeerPicker
}
```

**设计特点：**

- 协调领域服务和基础设施层
- 实现一致性哈希的业务用例
- 提供输入验证和错误处理
- 转换数据传输对象

### 2. 数据传输对象 (DTOs)

#### PeerSelectionCommand - 节点选择命令

```go
type PeerSelectionCommand struct {
    Key string `json:"key"`
}
```

#### MultiplePeerSelectionCommand - 多节点选择命令

```go
type MultiplePeerSelectionCommand struct {
    Key   string `json:"key"`
    Count int    `json:"count"`
}
```

#### PeerRequest - 节点请求

```go
type PeerRequest struct {
    ID      string `json:"id"`
    Address string `json:"address"`
    Weight  int    `json:"weight"`
}
```

#### PeerResult - 节点结果

```go
type PeerResult struct {
    ID      string `json:"id"`
    Address string `json:"address"`
    Weight  int    `json:"weight"`
    IsAlive bool   `json:"is_alive"`
}
```

#### PeerSelectionResult - 节点选择结果

```go
type PeerSelectionResult struct {
    Key  string     `json:"key"`
    Peer PeerResult `json:"peer"`
}
```

#### HashStatsResult - 哈希统计结果

```go
type HashStatsResult struct {
    TotalPeers      int                `json:"total_peers"`
    VirtualNodes    int                `json:"virtual_nodes"`
    Replicas        int                `json:"replicas"`
    KeyDistribution map[string]int     `json:"key_distribution"`
    LoadBalance     float64            `json:"load_balance"`
}
```

## 主要方法

### 1. 节点选择操作

#### SelectPeer - 选择节点

```go
func (s *ConsistentHashApplicationService) SelectPeer(ctx context.Context, cmd PeerSelectionCommand) (*PeerSelectionResult, error)
```

**用例**: 用户想要根据键选择一个节点来处理请求

**示例：**

```go
service := NewConsistentHashApplicationService(peerPicker)

cmd := PeerSelectionCommand{Key: "user:12345"}
result, err := service.SelectPeer(ctx, cmd)
if err != nil {
    log.Printf("选择节点失败: %v", err)
    return
}

fmt.Printf("为键 %s 选择的节点: %s (%s)\n", 
    result.Key, result.Peer.ID, result.Peer.Address)
```

#### SelectMultiplePeers - 选择多个节点

```go
func (s *ConsistentHashApplicationService) SelectMultiplePeers(ctx context.Context, cmd MultiplePeerSelectionCommand) (*MultiplePeerSelectionResult, error)
```

**用例**: 用户想要选择多个节点来实现数据副本或负载分担

**示例：**

```go
cmd := MultiplePeerSelectionCommand{
    Key:   "data:important",
    Count: 3,
}

result, err := service.SelectMultiplePeers(ctx, cmd)
if err != nil {
    log.Printf("选择多个节点失败: %v", err)
    return
}

fmt.Printf("为键 %s 选择了 %d 个节点:\n", result.Key, result.Count)
for i, peer := range result.Peers {
    fmt.Printf("  副本%d: %s (%s)\n", i+1, peer.ID, peer.Address)
}
```

### 2. 集群管理操作

#### AddPeers - 添加节点

```go
func (s *ConsistentHashApplicationService) AddPeers(ctx context.Context, cmd AddPeersCommand) error
```

**用例**: 用户想要向集群中添加新的节点

**示例：**

```go
cmd := AddPeersCommand{
    Peers: []PeerRequest{
        {ID: "server1", Address: "192.168.1.1:8080", Weight: 100},
        {ID: "server2", Address: "192.168.1.2:8080", Weight: 150},
        {ID: "server3", Address: "192.168.1.3:8080", Weight: 120},
    },
}

err := service.AddPeers(ctx, cmd)
if err != nil {
    log.Printf("添加节点失败: %v", err)
} else {
    fmt.Println("节点添加成功")
}
```

#### RemovePeers - 移除节点

```go
func (s *ConsistentHashApplicationService) RemovePeers(ctx context.Context, cmd RemovePeersCommand) error
```

#### GetAllPeers - 获取所有节点

```go
func (s *ConsistentHashApplicationService) GetAllPeers(ctx context.Context) ([]PeerResult, error)
```

### 3. 监控和健康检查

#### GetHashStats - 获取哈希统计信息

```go
func (s *ConsistentHashApplicationService) GetHashStats(ctx context.Context) (*HashStatsResult, error)
```

**用例**: 用户想要查看一致性哈希的统计信息和负载分布

#### CheckHealth - 检查健康状态

```go
func (s *ConsistentHashApplicationService) CheckHealth(ctx context.Context) (*HealthCheckResult, error)
```

**用例**: 用户想要检查一致性哈希系统是否健康

## 使用示例

### 1. 基本节点选择

```go
package main

import (
    "context"
    "fmt"
    "log"
    
    "github.com/justinwongcn/hamster/internal/application/consistent_hash"
    domainHash "github.com/justinwongcn/hamster/internal/domain/consistent_hash"
    infraHash "github.com/justinwongcn/hamster/internal/infrastructure/consistent_hash"
)

func main() {
    // 创建一致性哈希实现
    hashMap := infraHash.NewConsistentHashMap(150, nil)
    peerPicker := infraHash.NewSingleflightPeerPicker(hashMap)
    
    // 创建应用服务
    service := consistent_hash.NewConsistentHashApplicationService(peerPicker)
    
    ctx := context.Background()
    
    // 添加节点
    addCmd := consistent_hash.AddPeersCommand{
        Peers: []consistent_hash.PeerRequest{
            {ID: "server1", Address: "192.168.1.1:8080", Weight: 100},
            {ID: "server2", Address: "192.168.1.2:8080", Weight: 100},
            {ID: "server3", Address: "192.168.1.3:8080", Weight: 100},
        },
    }
    
    err := service.AddPeers(ctx, addCmd)
    if err != nil {
        log.Printf("添加节点失败: %v", err)
        return
    }
    
    fmt.Println("节点添加成功")
    
    // 选择节点
    testKeys := []string{"user:1", "user:2", "data:a", "cache:x"}
    
    for _, key := range testKeys {
        cmd := consistent_hash.PeerSelectionCommand{Key: key}
        result, err := service.SelectPeer(ctx, cmd)
        if err != nil {
            log.Printf("选择节点失败: %v", err)
            continue
        }
        
        fmt.Printf("键 %s -> 节点 %s (%s)\n", 
            key, result.Peer.ID, result.Peer.Address)
    }
}
```

### 2. 多节点副本选择

```go
func demonstrateReplicaSelection() {
    service := consistent_hash.NewConsistentHashApplicationService(peerPicker)
    ctx := context.Background()
    
    // 添加足够的节点
    addCmd := consistent_hash.AddPeersCommand{
        Peers: []consistent_hash.PeerRequest{
            {ID: "server1", Address: "192.168.1.1:8080", Weight: 100},
            {ID: "server2", Address: "192.168.1.2:8080", Weight: 100},
            {ID: "server3", Address: "192.168.1.3:8080", Weight: 100},
            {ID: "server4", Address: "192.168.1.4:8080", Weight: 100},
            {ID: "server5", Address: "192.168.1.5:8080", Weight: 100},
        },
    }
    
    service.AddPeers(ctx, addCmd)
    
    // 为重要数据选择多个副本节点
    dataItems := []string{"user_profiles", "order_history", "product_catalog"}
    
    for _, dataKey := range dataItems {
        cmd := consistent_hash.MultiplePeerSelectionCommand{
            Key:   dataKey,
            Count: 3, // 3个副本
        }
        
        result, err := service.SelectMultiplePeers(ctx, cmd)
        if err != nil {
            log.Printf("选择副本节点失败: %v", err)
            continue
        }
        
        fmt.Printf("\n数据 %s 的副本节点:\n", dataKey)
        for i, peer := range result.Peers {
            fmt.Printf("  副本%d: %s (%s) - 存活: %v\n", 
                i+1, peer.ID, peer.Address, peer.IsAlive)
        }
    }
}
```

### 3. 动态集群管理

```go
func demonstrateDynamicClusterManagement() {
    service := consistent_hash.NewConsistentHashApplicationService(peerPicker)
    ctx := context.Background()
    
    // 初始集群
    initialPeers := consistent_hash.AddPeersCommand{
        Peers: []consistent_hash.PeerRequest{
            {ID: "server1", Address: "192.168.1.1:8080", Weight: 100},
            {ID: "server2", Address: "192.168.1.2:8080", Weight: 100},
        },
    }
    
    service.AddPeers(ctx, initialPeers)
    
    // 查看初始状态
    peers, err := service.GetAllPeers(ctx)
    if err == nil {
        fmt.Printf("初始集群节点数: %d\n", len(peers))
        for _, peer := range peers {
            fmt.Printf("  节点: %s (%s) - 权重: %d\n", 
                peer.ID, peer.Address, peer.Weight)
        }
    }
    
    // 测试负载分布
    fmt.Println("\n测试负载分布:")
    distribution := make(map[string]int)
    
    for i := 0; i < 1000; i++ {
        key := fmt.Sprintf("key_%d", i)
        cmd := consistent_hash.PeerSelectionCommand{Key: key}
        result, err := service.SelectPeer(ctx, cmd)
        if err == nil {
            distribution[result.Peer.ID]++
        }
    }
    
    for peerID, count := range distribution {
        fmt.Printf("  节点 %s: %d 个键 (%.1f%%)\n", 
            peerID, count, float64(count)/1000*100)
    }
    
    // 扩容：添加新节点
    fmt.Println("\n扩容：添加新节点...")
    newPeers := consistent_hash.AddPeersCommand{
        Peers: []consistent_hash.PeerRequest{
            {ID: "server3", Address: "192.168.1.3:8080", Weight: 100},
            {ID: "server4", Address: "192.168.1.4:8080", Weight: 100},
        },
    }
    
    service.AddPeers(ctx, newPeers)
    
    // 查看扩容后的负载分布
    fmt.Println("扩容后的负载分布:")
    distribution = make(map[string]int)
    
    for i := 0; i < 1000; i++ {
        key := fmt.Sprintf("key_%d", i)
        cmd := consistent_hash.PeerSelectionCommand{Key: key}
        result, err := service.SelectPeer(ctx, cmd)
        if err == nil {
            distribution[result.Peer.ID]++
        }
    }
    
    for peerID, count := range distribution {
        fmt.Printf("  节点 %s: %d 个键 (%.1f%%)\n", 
            peerID, count, float64(count)/1000*100)
    }
    
    // 缩容：移除节点
    fmt.Println("\n缩容：移除节点...")
    removeCmd := consistent_hash.RemovePeersCommand{
        PeerIDs: []string{"server1"},
    }
    
    err = service.RemovePeers(ctx, removeCmd)
    if err != nil {
        log.Printf("移除节点失败: %v", err)
    } else {
        fmt.Println("节点移除成功")
    }
    
    // 查看最终状态
    peers, err = service.GetAllPeers(ctx)
    if err == nil {
        fmt.Printf("最终集群节点数: %d\n", len(peers))
    }
}
```

### 4. 健康检查和监控

```go
func demonstrateHealthCheckAndMonitoring() {
    service := consistent_hash.NewConsistentHashApplicationService(peerPicker)
    ctx := context.Background()
    
    // 添加节点
    addCmd := consistent_hash.AddPeersCommand{
        Peers: []consistent_hash.PeerRequest{
            {ID: "server1", Address: "192.168.1.1:8080", Weight: 100},
            {ID: "server2", Address: "192.168.1.2:8080", Weight: 150},
            {ID: "server3", Address: "192.168.1.3:8080", Weight: 120},
        },
    }
    
    service.AddPeers(ctx, addCmd)
    
    // 健康检查
    fmt.Println("执行健康检查...")
    healthResult, err := service.CheckHealth(ctx)
    if err != nil {
        log.Printf("健康检查失败: %v", err)
        return
    }
    
    fmt.Printf("系统健康状态: %v\n", healthResult.IsHealthy)
    fmt.Printf("健康检查消息: %s\n", healthResult.Message)
    
    // 获取统计信息
    fmt.Println("获取统计信息...")
    stats, err := service.GetHashStats(ctx)
    if err != nil {
        log.Printf("获取统计信息失败: %v", err)
        return
    }
    
    fmt.Printf("统计信息:\n")
    fmt.Printf("  总节点数: %d\n", stats.TotalPeers)
    fmt.Printf("  虚拟节点数: %d\n", stats.VirtualNodes)
    fmt.Printf("  副本数: %d\n", stats.Replicas)
    fmt.Printf("  负载均衡度: %.4f\n", stats.LoadBalance)
    
    // 定期监控
    fmt.Println("启动定期监控...")
    go func() {
        ticker := time.NewTicker(30 * time.Second)
        defer ticker.Stop()
        
        for {
            select {
            case <-ticker.C:
                health, err := service.CheckHealth(ctx)
                if err != nil {
                    log.Printf("监控检查失败: %v", err)
                    continue
                }
                
                if !health.IsHealthy {
                    log.Printf("警告: 系统不健康 - %s", health.Message)
                } else {
                    fmt.Printf("监控: 系统正常运行\n")
                }
                
                // 检查节点状态
                peers, err := service.GetAllPeers(ctx)
                if err == nil {
                    aliveCount := 0
                    for _, peer := range peers {
                        if peer.IsAlive {
                            aliveCount++
                        }
                    }
                    
                    fmt.Printf("监控: 存活节点 %d/%d\n", aliveCount, len(peers))
                    
                    if aliveCount < len(peers)/2 {
                        log.Printf("警告: 超过一半的节点不可用")
                    }
                }
            }
        }
    }()
    
    // 模拟运行一段时间
    time.Sleep(2 * time.Minute)
}
```

### 5. 负载均衡测试

```go
func demonstrateLoadBalancing() {
    service := consistent_hash.NewConsistentHashApplicationService(peerPicker)
    ctx := context.Background()
    
    // 添加不同权重的节点
    addCmd := consistent_hash.AddPeersCommand{
        Peers: []consistent_hash.PeerRequest{
            {ID: "high_capacity", Address: "192.168.1.1:8080", Weight: 200},
            {ID: "medium_capacity", Address: "192.168.1.2:8080", Weight: 100},
            {ID: "low_capacity", Address: "192.168.1.3:8080", Weight: 50},
        },
    }
    
    service.AddPeers(ctx, addCmd)
    
    // 测试大量键的分布
    fmt.Println("测试负载均衡...")
    distribution := make(map[string]int)
    totalKeys := 10000
    
    for i := 0; i < totalKeys; i++ {
        key := fmt.Sprintf("load_test_key_%d", i)
        cmd := consistent_hash.PeerSelectionCommand{Key: key}
        result, err := service.SelectPeer(ctx, cmd)
        if err == nil {
            distribution[result.Peer.ID]++
        }
    }
    
    fmt.Printf("负载分布结果 (总键数: %d):\n", totalKeys)
    totalWeight := 200 + 100 + 50 // 总权重
    
    for peerID, count := range distribution {
        percentage := float64(count) / float64(totalKeys) * 100
        
        // 计算期望百分比
        var expectedPercentage float64
        switch peerID {
        case "high_capacity":
            expectedPercentage = float64(200) / float64(totalWeight) * 100
        case "medium_capacity":
            expectedPercentage = float64(100) / float64(totalWeight) * 100
        case "low_capacity":
            expectedPercentage = float64(50) / float64(totalWeight) * 100
        }
        
        fmt.Printf("  节点 %s:\n", peerID)
        fmt.Printf("    实际: %d 个键 (%.2f%%)\n", count, percentage)
        fmt.Printf("    期望: %.2f%%\n", expectedPercentage)
        fmt.Printf("    偏差: %.2f%%\n", percentage-expectedPercentage)
    }
    
    // 测试副本选择的均匀性
    fmt.Println("\n测试副本选择的均匀性...")
    replicaDistribution := make(map[string]int)
    
    for i := 0; i < 1000; i++ {
        key := fmt.Sprintf("replica_test_key_%d", i)
        cmd := consistent_hash.MultiplePeerSelectionCommand{
            Key:   key,
            Count: 2, // 选择2个副本
        }
        
        result, err := service.SelectMultiplePeers(ctx, cmd)
        if err == nil {
            for _, peer := range result.Peers {
                replicaDistribution[peer.ID]++
            }
        }
    }
    
    fmt.Printf("副本分布结果 (1000个键，每个2个副本):\n")
    for peerID, count := range replicaDistribution {
        percentage := float64(count) / 2000 * 100 // 总共2000个副本
        fmt.Printf("  节点 %s: %d 个副本 (%.2f%%)\n", peerID, count, percentage)
    }
}
```

## 设计原则

### 1. 应用服务职责

- 协调领域服务和基础设施
- 实现具体的一致性哈希业务用例
- 提供输入验证和错误处理
- 转换数据传输对象

### 2. 数据传输对象设计

- 清晰的输入输出结构
- JSON标签支持序列化
- 业务语义的字段命名
- 合理的验证规则

### 3. 错误处理

- 统一的错误包装和传播
- 有意义的错误消息
- 区分业务错误和技术错误

### 4. 可扩展性

- 支持动态添加和移除节点
- 灵活的权重配置
- 可插拔的哈希算法

## 注意事项

### 1. 输入验证

```go
// ✅ 推荐：在应用服务层进行输入验证
func (s *ConsistentHashApplicationService) validateMultiplePeerSelectionCommand(cmd MultiplePeerSelectionCommand) error {
    if cmd.Key == "" {
        return fmt.Errorf("键不能为空")
    }
    if cmd.Count <= 0 {
        return fmt.Errorf("节点数量必须大于0")
    }
    if cmd.Count > 100 {
        return fmt.Errorf("节点数量不能超过100")
    }
    return nil
}

// ❌ 避免：跳过输入验证
func (s *ConsistentHashApplicationService) SelectMultiplePeers(ctx context.Context, cmd MultiplePeerSelectionCommand) (*MultiplePeerSelectionResult, error) {
    // 直接调用底层服务，没有验证
    return s.peerPicker.PickPeers(cmd.Key, cmd.Count)
}
```

### 2. 节点管理

```go
// ✅ 推荐：批量操作节点
addCmd := AddPeersCommand{
    Peers: []PeerRequest{
        {ID: "server1", Address: "192.168.1.1:8080", Weight: 100},
        {ID: "server2", Address: "192.168.1.2:8080", Weight: 100},
    },
}
service.AddPeers(ctx, addCmd)

// ❌ 避免：逐个添加节点（可能导致多次哈希环重建）
for _, peer := range peers {
    singleCmd := AddPeersCommand{Peers: []PeerRequest{peer}}
    service.AddPeers(ctx, singleCmd)
}
```

### 3. 权重设置

```go
// ✅ 推荐：根据节点容量设置合理权重
peers := []PeerRequest{
    {ID: "high_spec", Address: "192.168.1.1:8080", Weight: 200}, // 高配置
    {ID: "medium_spec", Address: "192.168.1.2:8080", Weight: 100}, // 中配置
    {ID: "low_spec", Address: "192.168.1.3:8080", Weight: 50},   // 低配置
}

// ❌ 避免：所有节点使用相同权重（忽略节点差异）
peers := []PeerRequest{
    {ID: "server1", Address: "192.168.1.1:8080", Weight: 100},
    {ID: "server2", Address: "192.168.1.2:8080", Weight: 100}, // 实际容量可能不同
}
```

### 4. 健康检查

```go
// ✅ 推荐：定期进行健康检查
go func() {
    ticker := time.NewTicker(30 * time.Second)
    defer ticker.Stop()
    
    for {
        select {
        case <-ticker.C:
            health, err := service.CheckHealth(ctx)
            if err != nil || !health.IsHealthy {
                log.Printf("系统健康检查失败: %v", health.Message)
                // 触发告警或自动恢复
            }
        }
    }
}()

// ❌ 避免：忽略健康状态
// 不进行健康检查，可能导致请求发送到故障节点
```

### 5. 副本数量选择

```go
// ✅ 推荐：根据可用性要求选择合适的副本数
// 高可用性数据：3个副本
cmd := MultiplePeerSelectionCommand{Key: "critical_data", Count: 3}

// 一般数据：2个副本
cmd := MultiplePeerSelectionCommand{Key: "normal_data", Count: 2}

// ❌ 避免：副本数量超过节点总数
totalPeers := len(service.GetAllPeers(ctx))
if cmd.Count > totalPeers {
    // 这会导致错误或重复选择
}
```
