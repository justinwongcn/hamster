# Infrastructure Consistent Hash 一致性哈希基础设施包

一致性哈希基础设施包提供了完整的一致性哈希算法实现，包括哈希环管理、节点选择、负载均衡和故障转移等功能。该包是分布式系统中实现数据分片和负载均衡的核心组件，为上层应用提供高效、可靠的节点选择服务。

## 📁 包结构

```
consistent_hash/
├── consistent_hash_map.go          # 一致性哈希映射实现
├── singleflight_peer_picker.go     # SingleFlight节点选择器
├── consistent_hash_test.go         # 一致性哈希测试
├── consistent_hash_map.md          # 哈希映射详细文档
├── singleflight_peer_picker.md     # 节点选择器详细文档
└── README.md                       # 包级别文档
```

## 🚀 主要功能

### 1. ConsistentHashMap - 一致性哈希映射

#### 核心特性
- **虚拟节点**: 支持虚拟节点提高负载均衡
- **自定义哈希**: 可插拔的哈希函数
- **动态扩缩容**: 支持节点的动态添加和移除
- **负载统计**: 提供详细的负载分布统计

#### 主要方法
```go
// 添加节点
hashMap.AddPeers(peer1, peer2, peer3)

// 移除节点
hashMap.RemovePeers(peer1)

// 选择节点
peer := hashMap.GetPeer(key)

// 选择多个节点
peers := hashMap.GetPeers(key, count)

// 获取统计信息
stats := hashMap.GetStats()
```

### 2. SingleflightPeerPicker - 节点选择器

#### 核心特性
- **SingleFlight优化**: 相同键的并发请求只执行一次
- **故障转移**: 自动检测节点故障并选择替代节点
- **健康检查**: 实时的节点健康状态管理
- **缓存管理**: 智能的选择结果缓存

#### 主要方法
```go
// 选择单个节点
peer, err := picker.PickPeer(key)

// 选择多个节点
peers, err := picker.PickPeers(key, count)

// 添加节点
picker.AddPeers(peer1, peer2, peer3)

// 更新节点状态
picker.UpdatePeerStatus(peerID, alive)

// 健康检查
healthy, err := picker.IsHealthy()
```

## 🔧 快速上手

### 基础使用

```go
import "github.com/justinwongcn/hamster/internal/infrastructure/consistent_hash"

// 创建一致性哈希映射
hashMap := consistent_hash.NewConsistentHashMap(150, nil) // 150个虚拟节点

// 创建节点选择器
picker := consistent_hash.NewSingleflightPeerPicker(hashMap)

// 添加节点
peer1, _ := domain.NewPeerInfo("server1", "192.168.1.1:8080", 100)
peer2, _ := domain.NewPeerInfo("server2", "192.168.1.2:8080", 100)
picker.AddPeers(peer1, peer2)

// 选择节点
peer, err := picker.PickPeer("user:12345")
if err != nil {
    log.Printf("选择节点失败: %v", err)
    return
}

fmt.Printf("选择的节点: %s (%s)\n", peer.ID(), peer.Address())
```

### 高级配置

```go
// 自定义哈希函数
import "hash/crc32"

customHash := func(data []byte) uint32 {
    return crc32.ChecksumIEEE(data)
}

hashMap := consistent_hash.NewConsistentHashMap(200, customHash)

// 创建带权重的节点
heavyPeer, _ := domain.NewPeerInfo("heavy-server", "192.168.1.10:8080", 200)
lightPeer, _ := domain.NewPeerInfo("light-server", "192.168.1.11:8080", 50)

picker.AddPeers(heavyPeer, lightPeer)
```

### 故障处理

```go
// 模拟节点故障
picker.UpdatePeerStatus("server1", false) // 标记server1为故障

// 选择节点时会自动故障转移
peer, err := picker.PickPeer("user:12345")
// 会选择其他可用节点

// 节点恢复
picker.UpdatePeerStatus("server1", true) // 恢复server1
```

## 🎯 架构设计

### 1. 一致性哈希算法

#### 哈希环结构
```
     Node A (Hash: 100)
         |
    ┌────┴────┐
    │         │
Node D      Node B
(Hash: 300) (Hash: 200)
    │         │
    └────┬────┘
         │
     Node C (Hash: 250)
```

#### 虚拟节点分布
- 每个物理节点对应多个虚拟节点
- 虚拟节点均匀分布在哈希环上
- 提高负载均衡效果

### 2. SingleFlight优化

#### 并发控制
```go
// 多个goroutine同时请求同一个键
go picker.PickPeer("same-key") // 只有一个实际执行
go picker.PickPeer("same-key") // 等待第一个结果
go picker.PickPeer("same-key") // 共享第一个结果
```

#### 缓存策略
- 短期缓存选择结果
- 节点变化时清除相关缓存
- 避免重复计算

### 3. 故障转移机制

#### 故障检测
```go
// 定期健康检查
go func() {
    ticker := time.NewTicker(30 * time.Second)
    for range ticker.C {
        for _, peer := range picker.GetAllPeers() {
            alive := checkPeerHealth(peer.Address())
            picker.UpdatePeerStatus(peer.ID(), alive)
        }
    }
}()
```

#### 替代节点选择
1. 检测到节点故障
2. 从哈希环获取后续节点
3. 验证替代节点健康状态
4. 返回可用的替代节点

## 📊 性能特性

### 时间复杂度
- **添加节点**: O(V log N) - V为虚拟节点数，N为总节点数
- **移除节点**: O(V log N)
- **选择节点**: O(log N) - 二分查找
- **选择多节点**: O(K log N) - K为请求节点数

### 空间复杂度
- **哈希环**: O(V × N) - 虚拟节点总数
- **节点映射**: O(N) - 物理节点数
- **SingleFlight**: O(K) - 正在处理的键数

### 负载均衡效果
```go
// 测试负载分布
distribution := make(map[string]int)
for i := 0; i < 10000; i++ {
    key := fmt.Sprintf("key_%d", i)
    peer, _ := picker.PickPeer(key)
    distribution[peer.ID()]++
}

// 计算负载均衡度
stats := hashMap.GetStats()
fmt.Printf("负载均衡度: %.4f\n", stats.LoadBalance())
```

## 🔍 监控和调试

### 统计信息
```go
// 获取哈希环统计
stats := hashMap.GetStats()
fmt.Printf("总节点数: %d\n", stats.TotalPeers())
fmt.Printf("虚拟节点数: %d\n", stats.VirtualNodes())
fmt.Printf("负载均衡度: %.4f\n", stats.LoadBalance())

// 获取节点分布
distribution := stats.KeyDistribution()
for peerID, count := range distribution {
    fmt.Printf("节点 %s: %d 个键\n", peerID, count)
}
```

### 健康监控
```go
// 检查系统健康状态
healthy, err := picker.IsHealthy()
if !healthy {
    log.Printf("系统不健康: %v", err)
}

// 获取存活节点数
aliveCount := picker.GetAlivePeerCount()
totalCount := picker.GetPeerCount()
fmt.Printf("存活节点: %d/%d\n", aliveCount, totalCount)
```

### 性能监控
```go
// 监控选择性能
start := time.Now()
peer, err := picker.PickPeer(key)
duration := time.Since(start)

if duration > time.Millisecond {
    log.Printf("节点选择耗时过长: %v", duration)
}
```

## ⚠️ 最佳实践

### 1. 虚拟节点配置
```go
// ✅ 推荐：根据节点数量设置虚拟节点
nodeCount := 10
virtualNodes := nodeCount * 15 // 每个节点15个虚拟节点
hashMap := NewConsistentHashMap(virtualNodes, nil)

// ❌ 避免：虚拟节点过少导致负载不均
hashMap := NewConsistentHashMap(nodeCount, nil) // 每个节点只有1个虚拟节点
```

### 2. 节点权重设置
```go
// ✅ 推荐：根据节点容量设置权重
highCapacityPeer, _ := domain.NewPeerInfo("high", "addr1", 200)
mediumCapacityPeer, _ := domain.NewPeerInfo("medium", "addr2", 100)
lowCapacityPeer, _ := domain.NewPeerInfo("low", "addr3", 50)

// ❌ 避免：忽略节点差异，使用相同权重
allPeers := []*domain.PeerInfo{
    domain.NewPeerInfo("server1", "addr1", 100), // 实际容量可能不同
    domain.NewPeerInfo("server2", "addr2", 100),
}
```

### 3. 故障处理
```go
// ✅ 推荐：实现健康检查
func healthCheck(picker *SingleflightPeerPicker) {
    go func() {
        ticker := time.NewTicker(30 * time.Second)
        for range ticker.C {
            peers := picker.GetAllPeers()
            for _, peer := range peers {
                alive := pingPeer(peer.Address())
                picker.UpdatePeerStatus(peer.ID(), alive)
            }
        }
    }()
}

// ✅ 推荐：处理选择失败
peer, err := picker.PickPeer(key)
if err != nil {
    // 记录错误并使用降级策略
    log.Printf("节点选择失败: %v", err)
    peer = getDefaultPeer() // 降级到默认节点
}
```

### 4. 缓存管理
```go
// ✅ 推荐：在节点变化时清除缓存
func (p *SingleflightPeerPicker) AddPeers(peers ...Peer) {
    // 添加节点
    p.addPeersInternal(peers...)
    
    // 清除可能受影响的缓存
    p.clearRelevantCache()
}

// ✅ 推荐：定期清理长时间未使用的缓存
go func() {
    ticker := time.NewTicker(time.Hour)
    for range ticker.C {
        picker.cleanupStaleCache()
    }
}()
```

## 🧪 测试指南

### 单元测试
```bash
# 运行所有测试
go test ./internal/infrastructure/consistent_hash/

# 运行特定测试
go test -run TestConsistentHashMap ./internal/infrastructure/consistent_hash/

# 查看测试覆盖率
go test -cover ./internal/infrastructure/consistent_hash/
```

### 负载均衡测试
```go
func TestLoadBalance(t *testing.T) {
    hashMap := NewConsistentHashMap(150, nil)
    
    // 添加节点
    for i := 0; i < 5; i++ {
        peer, _ := domain.NewPeerInfo(fmt.Sprintf("server%d", i), 
            fmt.Sprintf("addr%d", i), 100)
        hashMap.AddPeers(peer)
    }
    
    // 测试分布
    distribution := make(map[string]int)
    for i := 0; i < 10000; i++ {
        key := fmt.Sprintf("key_%d", i)
        peer := hashMap.GetPeer(key)
        distribution[peer.ID()]++
    }
    
    // 验证负载均衡
    expectedCount := 10000 / 5
    for peerID, count := range distribution {
        deviation := float64(count-expectedCount) / float64(expectedCount)
        if math.Abs(deviation) > 0.2 { // 允许20%偏差
            t.Errorf("节点 %s 负载偏差过大: %f", peerID, deviation)
        }
    }
}
```

### 故障转移测试
```go
func TestFailover(t *testing.T) {
    picker := NewSingleflightPeerPicker(hashMap)
    
    // 添加节点
    peer1, _ := domain.NewPeerInfo("server1", "addr1", 100)
    peer2, _ := domain.NewPeerInfo("server2", "addr2", 100)
    picker.AddPeers(peer1, peer2)
    
    key := "test_key"
    
    // 正常选择
    selectedPeer, err := picker.PickPeer(key)
    require.NoError(t, err)
    
    // 模拟节点故障
    picker.UpdatePeerStatus(selectedPeer.ID(), false)
    
    // 故障转移
    newPeer, err := picker.PickPeer(key)
    require.NoError(t, err)
    assert.NotEqual(t, selectedPeer.ID(), newPeer.ID())
}
```

## 🔄 扩展指南

### 添加新的哈希函数
```go
// 实现自定义哈希函数
func customHashFunc(data []byte) uint32 {
    // 自定义哈希算法实现
    return hash
}

// 使用自定义哈希函数
hashMap := NewConsistentHashMap(150, customHashFunc)
```

### 扩展节点选择策略
```go
// 实现自定义选择策略
type CustomPeerPicker struct {
    *SingleflightPeerPicker
    customLogic CustomLogic
}

func (c *CustomPeerPicker) PickPeer(key string) (Peer, error) {
    // 自定义选择逻辑
    if c.customLogic.ShouldUseCustom(key) {
        return c.customLogic.SelectPeer(key)
    }
    
    // 回退到默认逻辑
    return c.SingleflightPeerPicker.PickPeer(key)
}
```

## 📈 性能优化

### 1. 虚拟节点优化
- 合理设置虚拟节点数量
- 避免过多虚拟节点影响性能
- 根据节点数量动态调整

### 2. 缓存优化
- 合理设置缓存过期时间
- 及时清理无效缓存
- 使用LRU等策略管理缓存

### 3. 并发优化
- 使用读写锁分离读写操作
- 减少锁持有时间
- 使用原子操作优化计数器

Infrastructure Consistent Hash包为分布式系统提供了高效、可靠的一致性哈希解决方案，支持动态扩缩容、故障转移和负载均衡等关键特性。
