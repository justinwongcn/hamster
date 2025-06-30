# consistent_hash.go - 一致性哈希领域模型

## 文件概述

`consistent_hash.go` 定义了一致性哈希算法的完整领域模型，包括核心接口、值对象和领域服务。该文件遵循DDD设计原则，将一致性哈希的业务逻辑封装在领域层，为分布式系统的负载均衡和数据分片提供核心抽象。

## 核心功能

### 1. 错误定义

```go
var (
    ErrNoPeers         = errors.New("没有可用的节点")
    ErrInvalidKey      = errors.New("无效的键")
    ErrInvalidPeer     = errors.New("无效的节点")
    ErrInvalidReplicas = errors.New("无效的虚拟节点倍数")
)
```

定义了一致性哈希相关的领域错误，便于错误识别和处理。

### 2. Hash 函数类型

```go
type Hash func(data []byte) uint32
```

**设计特点：**

- 采用依赖注入方式，支持自定义哈希函数
- 默认使用crc32.ChecksumIEEE算法
- 支持运行时替换哈希函数

### 3. ConsistentHash 核心接口

```go
type ConsistentHash interface {
    Add(peers ...string)
    Remove(peers ...string)
    Get(key string) (string, error)
    GetMultiple(key string, count int) ([]string, error)
    Peers() []string
    IsEmpty() bool
    Stats() HashStats
}
```

**核心方法：**

- **Add**: 添加节点到哈希环
- **Remove**: 从哈希环中移除节点
- **Get**: 根据键获取对应的节点
- **GetMultiple**: 获取多个节点（用于副本）
- **Peers**: 获取所有节点
- **IsEmpty**: 检查是否为空
- **Stats**: 获取统计信息

### 4. PeerPicker 节点选择器接口

```go
type PeerPicker interface {
    PickPeer(key string) (Peer, error)
    PickPeers(key string, count int) ([]Peer, error)
    AddPeers(peers ...Peer)
    RemovePeers(peers ...Peer)
    GetAllPeers() []Peer
    IsHealthy() (bool, error)
}
```

**抽象特点：**

- 封装分布式节点选择逻辑
- 支持节点健康检查
- 提供高层次的节点管理接口

### 5. Peer 节点接口

```go
type Peer interface {
    ID() string
    Address() string
    IsAlive() bool
    Weight() int
    Equals(other Peer) bool
}
```

**节点属性：**

- **ID**: 节点唯一标识
- **Address**: 节点地址
- **IsAlive**: 存活状态
- **Weight**: 节点权重
- **Equals**: 节点比较

### 6. HashKey 哈希键值对象

#### 结构定义

```go
type HashKey struct {
    value string
}
```

#### 构造函数

```go
func NewHashKey(key string) (HashKey, error)
```

**验证规则：**

- 键不能为空
- 键长度不能超过500个字符

**示例：**

```go
key, err := NewHashKey("user:12345")
if err != nil {
    log.Printf("键创建失败: %v", err)
    return
}

fmt.Printf("键值: %s\n", key.String())
fmt.Printf("字节表示: %v\n", key.Bytes())

// 计算哈希值
hashFunc := crc32.ChecksumIEEE
hashValue := key.Hash(hashFunc)
fmt.Printf("哈希值: %d\n", hashValue)
```

#### 操作方法

- `String()`: 返回字符串表示
- `Bytes()`: 返回字节表示
- `Hash(hashFunc Hash)`: 计算哈希值
- `Equals(other HashKey)`: 比较键是否相等

### 7. PeerInfo 节点信息值对象

#### 结构定义

```go
type PeerInfo struct {
    id      string
    address string
    weight  int
    alive   bool
}
```

#### 构造函数

```go
func NewPeerInfo(id, address string, weight int) (PeerInfo, error)
```

**验证规则：**

- 节点ID不能为空
- 节点地址不能为空
- 节点权重不能为负数

**示例：**

```go
peer, err := NewPeerInfo("server1", "192.168.1.1:8080", 100)
if err != nil {
    log.Printf("节点创建失败: %v", err)
    return
}

fmt.Printf("节点ID: %s\n", peer.ID())
fmt.Printf("节点地址: %s\n", peer.Address())
fmt.Printf("节点权重: %d\n", peer.Weight())
fmt.Printf("是否存活: %v\n", peer.IsAlive())

// 设置存活状态
updatedPeer := peer.SetAlive(false)
fmt.Printf("更新后存活状态: %v\n", updatedPeer.IsAlive())
```

### 8. HashStats 统计信息值对象

#### 结构定义

```go
type HashStats struct {
    totalPeers      int
    virtualNodes    int
    replicas        int
    keyDistribution map[string]int
}
```

#### 分析方法

```go
func (s HashStats) LoadBalance() float64
```

计算负载均衡度，返回标准差，值越小表示负载越均衡。

**示例：**

```go
stats := hashRing.Stats()

fmt.Printf("统计信息:\n")
fmt.Printf("  总节点数: %d\n", stats.TotalPeers())
fmt.Printf("  虚拟节点数: %d\n", stats.VirtualNodes())
fmt.Printf("  虚拟节点倍数: %d\n", stats.Replicas())
fmt.Printf("  负载均衡度: %.4f\n", stats.LoadBalance())

// 查看键分布
distribution := stats.KeyDistribution()
for peer, count := range distribution {
    fmt.Printf("  %s: %d个虚拟节点\n", peer, count)
}
```

### 9. VirtualNodeConfig 虚拟节点配置值对象

#### 结构定义

```go
type VirtualNodeConfig struct {
    replicas int
    hashFunc Hash
}
```

#### 构造函数

```go
func NewVirtualNodeConfig(replicas int, hashFunc Hash) (VirtualNodeConfig, error)
```

**验证规则：**

- 虚拟节点倍数必须大于0
- 虚拟节点倍数不能超过1000
- 哈希函数为nil时使用默认函数

**示例：**

```go
// 使用默认哈希函数
config, err := NewVirtualNodeConfig(150, nil)
if err != nil {
    log.Printf("配置创建失败: %v", err)
    return
}

// 使用自定义哈希函数
import "github.com/cespare/xxhash/v2"
customHash := func(data []byte) uint32 {
    return uint32(xxhash.Sum64(data))
}
config, err = NewVirtualNodeConfig(100, customHash)

// 生成虚拟节点键
virtualKeys := config.GenerateVirtualNodeKeys("server1")
fmt.Printf("虚拟节点键: %v\n", virtualKeys)
// 输出: [server1#0 server1#1 server1#2 ...]
```

### 10. HashRing 哈希环值对象

#### 结构定义

```go
type HashRing struct {
    keys    []uint32          // 排序的哈希值列表
    hashMap map[uint32]string // 虚拟节点到真实节点的映射
    config  VirtualNodeConfig // 虚拟节点配置
}
```

#### 核心操作

##### 添加节点

```go
func (r HashRing) AddPeer(peer string) HashRing
```

**实现逻辑：**

1. 为真实节点生成多个虚拟节点键
2. 计算每个虚拟节点的哈希值
3. 添加到哈希环和映射表
4. 保持哈希环有序

##### 移除节点

```go
func (r HashRing) RemovePeer(peer string) HashRing
```

##### 获取节点

```go
func (r HashRing) GetPeer(key string) (string, bool)
func (r HashRing) GetMultiplePeers(key string, count int) []string
```

**查找算法：**

1. 计算键的哈希值
2. 在哈希环上顺时针查找第一个大于等于该哈希值的虚拟节点
3. 如果没找到，选择第一个虚拟节点（环形结构）
4. 返回对应的真实节点

## 使用示例

### 1. 基本哈希环操作

```go
package main

import (
    "fmt"
    "log"
    
    "github.com/justinwongcn/hamster/internal/domain/consistent_hash"
)

func main() {
    // 创建虚拟节点配置
    config, err := consistent_hash.NewVirtualNodeConfig(150, nil)
    if err != nil {
        log.Fatal(err)
    }
    
    // 创建哈希环
    ring := consistent_hash.NewHashRing(config)
    
    // 添加节点
    ring = ring.AddPeer("server1")
    ring = ring.AddPeer("server2")
    ring = ring.AddPeer("server3")
    
    // 查找节点
    key := "user:12345"
    if peer, found := ring.GetPeer(key); found {
        fmt.Printf("键 %s 分配到节点: %s\n", key, peer)
    }
    
    // 获取多个节点（用于副本）
    replicas := ring.GetMultiplePeers(key, 2)
    fmt.Printf("副本节点: %v\n", replicas)
    
    // 查看统计信息
    stats := ring.Stats()
    fmt.Printf("总节点数: %d\n", stats.TotalPeers())
    fmt.Printf("虚拟节点数: %d\n", stats.VirtualNodes())
    fmt.Printf("负载均衡度: %.4f\n", stats.LoadBalance())
}
```

### 2. 节点管理

```go
func demonstrateNodeManagement() {
    config, _ := consistent_hash.NewVirtualNodeConfig(100, nil)
    ring := consistent_hash.NewHashRing(config)
    
    // 批量添加节点
    servers := []string{"server1", "server2", "server3", "server4"}
    for _, server := range servers {
        ring = ring.AddPeer(server)
        fmt.Printf("添加节点: %s\n", server)
    }
    
    // 测试键分布
    testKeys := []string{
        "user:1", "user:2", "user:3", "user:4", "user:5",
        "data:a", "data:b", "data:c", "data:d", "data:e",
    }
    
    distribution := make(map[string][]string)
    for _, key := range testKeys {
        if peer, found := ring.GetPeer(key); found {
            distribution[peer] = append(distribution[peer], key)
        }
    }
    
    fmt.Println("\n键分布情况:")
    for peer, keys := range distribution {
        fmt.Printf("  %s: %v\n", peer, keys)
    }
    
    // 移除一个节点
    fmt.Println("\n移除 server2...")
    ring = ring.RemovePeer("server2")
    
    // 重新测试分布
    newDistribution := make(map[string][]string)
    for _, key := range testKeys {
        if peer, found := ring.GetPeer(key); found {
            newDistribution[peer] = append(newDistribution[peer], key)
        }
    }
    
    fmt.Println("移除节点后的键分布:")
    for peer, keys := range newDistribution {
        fmt.Printf("  %s: %v\n", peer, keys)
    }
}
```

### 3. 节点信息管理

```go
func demonstratePeerInfo() {
    // 创建节点信息
    peers := []consistent_hash.PeerInfo{}
    
    peerConfigs := []struct {
        id      string
        address string
        weight  int
    }{
        {"server1", "192.168.1.1:8080", 100},
        {"server2", "192.168.1.2:8080", 150},
        {"server3", "192.168.1.3:8080", 80},
    }
    
    for _, config := range peerConfigs {
        peer, err := consistent_hash.NewPeerInfo(config.id, config.address, config.weight)
        if err != nil {
            log.Printf("创建节点失败: %v", err)
            continue
        }
        peers = append(peers, peer)
    }
    
    // 显示节点信息
    fmt.Println("节点信息:")
    for _, peer := range peers {
        fmt.Printf("  ID: %s, 地址: %s, 权重: %d, 存活: %v\n",
            peer.ID(), peer.Address(), peer.Weight(), peer.IsAlive())
    }
    
    // 模拟节点故障
    fmt.Println("\n模拟 server2 故障...")
    for i, peer := range peers {
        if peer.ID() == "server2" {
            peers[i] = peer.SetAlive(false)
            break
        }
    }
    
    // 显示更新后的状态
    fmt.Println("更新后的节点状态:")
    for _, peer := range peers {
        status := "存活"
        if !peer.IsAlive() {
            status = "故障"
        }
        fmt.Printf("  %s: %s\n", peer.ID(), status)
    }
}
```

### 4. 负载均衡分析

```go
func analyzeLoadBalance() {
    // 测试不同虚拟节点数量的效果
    replicasCounts := []int{10, 50, 150, 500}
    
    for _, replicas := range replicasCounts {
        fmt.Printf("\n测试虚拟节点数量: %d\n", replicas)
        
        config, _ := consistent_hash.NewVirtualNodeConfig(replicas, nil)
        ring := consistent_hash.NewHashRing(config)
        
        // 添加节点
        for i := 1; i <= 5; i++ {
            ring = ring.AddPeer(fmt.Sprintf("server%d", i))
        }
        
        // 生成大量测试键
        keyDistribution := make(map[string]int)
        for i := 0; i < 10000; i++ {
            key := fmt.Sprintf("key_%d", i)
            if peer, found := ring.GetPeer(key); found {
                keyDistribution[peer]++
            }
        }
        
        // 计算负载均衡度
        total := 10000
        nodeCount := 5
        avg := float64(total) / float64(nodeCount)
        
        variance := 0.0
        for _, count := range keyDistribution {
            diff := float64(count) - avg
            variance += diff * diff
            fmt.Printf("  %s: %d (%.2f%%)\n", 
                fmt.Sprintf("server%d", count), count, float64(count)/float64(total)*100)
        }
        variance /= float64(nodeCount)
        stdDev := variance // 简化计算
        
        fmt.Printf("  标准差: %.2f (越小越均衡)\n", stdDev)
    }
}
```

### 5. 自定义哈希函数

```go
func demonstrateCustomHashFunction() {
    // 定义自定义哈希函数
    customHash := func(data []byte) uint32 {
        // 简单的自定义哈希算法（仅作示例）
        hash := uint32(0)
        for _, b := range data {
            hash = hash*31 + uint32(b)
        }
        return hash
    }
    
    // 创建配置
    config, err := consistent_hash.NewVirtualNodeConfig(100, customHash)
    if err != nil {
        log.Printf("配置创建失败: %v", err)
        return
    }
    
    ring := consistent_hash.NewHashRing(config)
    ring = ring.AddPeer("server1")
    ring = ring.AddPeer("server2")
    
    // 测试哈希函数
    testKeys := []string{"test1", "test2", "test3"}
    
    fmt.Println("使用自定义哈希函数:")
    for _, key := range testKeys {
        hashKey, _ := consistent_hash.NewHashKey(key)
        hashValue := hashKey.Hash(customHash)
        
        if peer, found := ring.GetPeer(key); found {
            fmt.Printf("  键: %s, 哈希值: %d, 节点: %s\n", key, hashValue, peer)
        }
    }
}
```

## 设计原则

### 1. 值对象不变性

- 所有值对象创建后不可修改
- 通过构造函数进行验证
- 提供只读访问方法

### 2. 接口分离

- 将核心算法和节点管理分离
- 支持不同的实现策略
- 便于测试和扩展

### 3. 领域封装

- 将一致性哈希的数学计算封装在领域层
- 隐藏实现细节，暴露业务概念
- 提供类型安全的操作

### 4. 依赖注入

- 支持自定义哈希函数
- 便于测试和性能优化
- 提高系统的灵活性

## 注意事项

### 1. 虚拟节点数量选择

```go
// ✅ 推荐：根据节点数量选择合适的虚拟节点倍数
func calculateOptimalReplicas(nodeCount int) int {
    if nodeCount <= 3 {
        return 150 // 少量节点需要更多虚拟节点
    } else if nodeCount <= 10 {
        return 100
    } else {
        return 50  // 大量节点可以减少虚拟节点
    }
}
```

### 2. 哈希函数选择

```go
// ✅ 推荐：使用高质量的哈希函数
import "github.com/cespare/xxhash/v2"

func xxHash(data []byte) uint32 {
    return uint32(xxhash.Sum64(data))
}

config, _ := consistent_hash.NewVirtualNodeConfig(150, xxHash)
```

### 3. 键的使用

```go
// ✅ 推荐：使用有意义的键名
key, err := consistent_hash.NewHashKey("user:profile:12345")

// ❌ 避免：键名过长
longKey := strings.Repeat("a", 501)
key, err := consistent_hash.NewHashKey(longKey) // 会返回错误
```

### 4. 节点管理

```go
// ✅ 推荐：验证节点信息
peer, err := consistent_hash.NewPeerInfo("server1", "192.168.1.1:8080", 100)
if err != nil {
    log.Printf("节点创建失败: %v", err)
    return
}

// ✅ 推荐：监控负载均衡
stats := ring.Stats()
if stats.LoadBalance() > threshold {
    log.Println("警告: 负载不均衡，考虑调整虚拟节点数量")
}
```
