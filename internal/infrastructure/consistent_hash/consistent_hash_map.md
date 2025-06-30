# consistent_hash_map.go - 一致性哈希映射实现

## 文件概述

`consistent_hash_map.go` 实现了一致性哈希算法的核心数据结构，包含Hash函数、虚拟节点倍数、哈希环、虚拟节点与真实节点的映射表。这是分布式系统中实现负载均衡和数据分片的关键组件，支持动态扩缩容且数据迁移量最小。

## 核心功能

### 1. ConsistentHashMap 结构体

```go
type ConsistentHashMap struct {
    hash     domainHash.Hash   // Hash函数
    replicas int               // 虚拟节点倍数
    keys     []uint32          // 哈希环（排序的哈希值列表）
    hashMap  map[uint32]string // 虚拟节点与真实节点的映射表，键是虚拟节点的哈希值，值是真实节点的名称
    mu       sync.RWMutex      // 读写锁保护
}
```

**设计特点：**

- **Hash函数**: 支持依赖注入，默认使用crc32.ChecksumIEEE
- **虚拟节点倍数**: 解决数据倾斜问题，提升负载均衡
- **哈希环**: 有序数组实现，支持二分查找
- **映射表**: 虚拟节点到真实节点的快速映射
- **线程安全**: 读写锁保护并发访问

### 2. 虚拟节点机制

虚拟节点是一致性哈希的核心优化，通过为每个真实节点创建多个虚拟节点来解决数据分布不均的问题。

**虚拟节点键生成规则：**

```go
func (m *ConsistentHashMap) generateVirtualNodeKey(peer string, index int) string {
    return fmt.Sprintf("%s#%d", peer, index)
}
```

**示例：**

- 真实节点 "server1" 有3个虚拟节点：
    - "server1#0"
    - "server1#1"
    - "server1#2"

## 主要方法

### 1. 构造函数

```go
func NewConsistentHashMap(replicas int, hashFunc domainHash.Hash) *ConsistentHashMap
```

**参数：**

- `replicas`: 虚拟节点倍数，建议50-200之间
- `hashFunc`: 哈希函数，nil时使用默认的crc32.ChecksumIEEE

**示例：**

```go
// 使用默认哈希函数，150个虚拟节点
hashMap := NewConsistentHashMap(150, nil)

// 使用自定义哈希函数
customHash := func(data []byte) uint32 {
    return xxhash.Sum32(data)
}
hashMap := NewConsistentHashMap(100, customHash)
```

### 2. 节点管理

#### Add - 添加节点

```go
func (m *ConsistentHashMap) Add(peers ...string)
```

**实现逻辑：**

1. 为每个真实节点创建replicas个虚拟节点
2. 计算虚拟节点的哈希值
3. 添加到哈希环和映射表
4. 保持哈希环有序

**示例：**

```go
hashMap.Add("server1", "server2", "server3")
```

#### Remove - 移除节点

```go
func (m *ConsistentHashMap) Remove(peers ...string)
```

**实现逻辑：**

1. 找到节点的所有虚拟节点
2. 从哈希环和映射表中删除
3. 重新整理哈希环

**示例：**

```go
hashMap.Remove("server2") // 移除server2及其所有虚拟节点
```

### 3. 节点选择

#### Get - 获取单个节点

```go
func (m *ConsistentHashMap) Get(key string) (string, error)
```

**实现逻辑：**

1. 计算键的哈希值
2. 在哈希环上顺时针查找第一个大于等于该哈希值的虚拟节点
3. 如果没找到，选择第一个虚拟节点（环形结构）
4. 返回对应的真实节点

**示例：**

```go
server, err := hashMap.Get("user:123")
if err != nil {
    return err
}
fmt.Printf("用户123分配到服务器: %s\n", server)
```

#### GetMultiple - 获取多个节点

```go
func (m *ConsistentHashMap) GetMultiple(key string, count int) ([]string, error)
```

**实现逻辑：**

1. 从Get方法确定的起始位置开始
2. 顺时针遍历哈希环
3. 收集不同的真实节点
4. 直到达到所需数量或遍历完所有节点

**示例：**

```go
// 为数据副本选择3个不同的服务器
servers, err := hashMap.GetMultiple("data:456", 3)
if err != nil {
    return err
}
fmt.Printf("数据456的副本服务器: %v\n", servers)
```

### 4. 信息查询

#### Peers - 获取所有节点

```go
func (m *ConsistentHashMap) Peers() []string
```

#### IsEmpty - 检查是否为空

```go
func (m *ConsistentHashMap) IsEmpty() bool
```

#### Stats - 获取统计信息

```go
func (m *ConsistentHashMap) Stats() domainHash.HashStats
```

**返回信息包括：**

- 总节点数
- 虚拟节点数
- 虚拟节点倍数
- 每个节点的虚拟节点分布

## 哈希环实现

### 1. 二分查找优化

```go
func (m *ConsistentHashMap) Get(key string) (string, error) {
    m.mu.RLock()
    defer m.mu.RUnlock()
    
    if len(m.keys) == 0 {
        return "", domainHash.ErrNoPeers
    }
    
    // 计算键的哈希值
    hash := m.hash([]byte(key))
    
    // 使用二分查找定位节点
    idx := sort.Search(len(m.keys), func(i int) bool {
        return m.keys[i] >= hash
    })
    
    // 环形结构：如果没找到，选择第一个节点
    if idx == len(m.keys) {
        idx = 0
    }
    
    return m.hashMap[m.keys[idx]], nil
}
```

### 2. 哈希环维护

```go
func (m *ConsistentHashMap) Add(peers ...string) {
    m.mu.Lock()
    defer m.mu.Unlock()
    
    for _, peer := range peers {
        // 为每个真实节点创建多个虚拟节点
        for i := 0; i < m.replicas; i++ {
            virtualKey := m.generateVirtualNodeKey(peer, i)
            hash := m.hash([]byte(virtualKey))
            
            m.keys = append(m.keys, hash)
            m.hashMap[hash] = peer
        }
    }
    
    // 保持哈希环有序
    sort.Slice(m.keys, func(i, j int) bool {
        return m.keys[i] < m.keys[j]
    })
}
```

## 负载均衡分析

### 1. 虚拟节点效果

不同虚拟节点倍数对负载均衡的影响：

```go
func analyzeLoadBalance(hashMap *ConsistentHashMap, testKeys []string) {
    distribution := make(map[string]int)
    
    // 统计每个节点分配到的键数量
    for _, key := range testKeys {
        server, err := hashMap.Get(key)
        if err == nil {
            distribution[server]++
        }
    }
    
    // 计算负载均衡度（标准差）
    total := len(testKeys)
    nodeCount := len(distribution)
    avg := float64(total) / float64(nodeCount)
    
    variance := 0.0
    for _, count := range distribution {
        diff := float64(count) - avg
        variance += diff * diff
    }
    variance /= float64(nodeCount)
    stdDev := math.Sqrt(variance)
    
    fmt.Printf("平均负载: %.2f, 标准差: %.2f\n", avg, stdDev)
}
```

### 2. 一致性验证

```go
func verifyConsistency(hashMap *ConsistentHashMap, testKeys []string) {
    // 记录初始映射
    initialMapping := make(map[string]string)
    for _, key := range testKeys {
        server, _ := hashMap.Get(key)
        initialMapping[key] = server
    }
    
    // 添加新节点
    hashMap.Add("new_server")
    
    // 检查映射变化
    changedCount := 0
    for _, key := range testKeys {
        server, _ := hashMap.Get(key)
        if server != initialMapping[key] {
            changedCount++
        }
    }
    
    changeRate := float64(changedCount) / float64(len(testKeys))
    fmt.Printf("添加节点后，%.2f%% 的键发生了重新映射\n", changeRate*100)
}
```

## 性能优化

### 1. 内存预分配

```go
func NewConsistentHashMap(replicas int, hashFunc domainHash.Hash) *ConsistentHashMap {
    if hashFunc == nil {
        hashFunc = crc32.ChecksumIEEE
    }
    
    // 预估初始容量，减少扩容开销
    estimatedNodes := 10 // 预估节点数
    initialCapacity := estimatedNodes * replicas
    
    return &ConsistentHashMap{
        hash:     hashFunc,
        replicas: replicas,
        keys:     make([]uint32, 0, initialCapacity),
        hashMap:  make(map[uint32]string, initialCapacity),
    }
}
```

### 2. 批量操作

```go
func (m *ConsistentHashMap) AddBatch(peers []string) {
    m.mu.Lock()
    defer m.mu.Unlock()
    
    // 批量添加，最后统一排序
    for _, peer := range peers {
        for i := 0; i < m.replicas; i++ {
            virtualKey := m.generateVirtualNodeKey(peer, i)
            hash := m.hash([]byte(virtualKey))
            
            m.keys = append(m.keys, hash)
            m.hashMap[hash] = peer
        }
    }
    
    // 一次性排序
    sort.Slice(m.keys, func(i, j int) bool {
        return m.keys[i] < m.keys[j]
    })
}

func (m *ConsistentHashMap) GetBatch(keys []string) map[string]string {
    m.mu.RLock()
    defer m.mu.RUnlock()
    
    results := make(map[string]string, len(keys))
    for _, key := range keys {
        if server, err := m.getInternal(key); err == nil {
            results[key] = server
        }
    }
    
    return results
}
```

## 使用示例

### 1. 基本使用

```go
// 创建一致性哈希映射
hashMap := NewConsistentHashMap(150, nil)

// 添加服务器节点
hashMap.Add("server1", "server2", "server3")

// 为用户分配服务器
userID := "user_12345"
server, err := hashMap.Get(userID)
if err != nil {
    log.Printf("分配服务器失败: %v", err)
    return
}

fmt.Printf("用户 %s 分配到服务器: %s\n", userID, server)

// 为数据选择多个副本服务器
dataKey := "data_67890"
servers, err := hashMap.GetMultiple(dataKey, 2)
if err != nil {
    log.Printf("选择副本服务器失败: %v", err)
    return
}

fmt.Printf("数据 %s 的副本服务器: %v\n", dataKey, servers)
```

### 2. 动态扩缩容

```go
func demonstrateScaling() {
    hashMap := NewConsistentHashMap(100, nil)
    
    // 初始节点
    hashMap.Add("server1", "server2", "server3")
    
    // 生成测试数据
    testKeys := make([]string, 1000)
    for i := 0; i < 1000; i++ {
        testKeys[i] = fmt.Sprintf("key_%d", i)
    }
    
    // 记录扩容前的分布
    beforeDistribution := make(map[string]string)
    for _, key := range testKeys {
        server, _ := hashMap.Get(key)
        beforeDistribution[key] = server
    }
    
    // 添加新服务器
    fmt.Println("添加新服务器...")
    hashMap.Add("server4")
    
    // 统计数据迁移
    migrationCount := 0
    for _, key := range testKeys {
        server, _ := hashMap.Get(key)
        if server != beforeDistribution[key] {
            migrationCount++
        }
    }
    
    migrationRate := float64(migrationCount) / float64(len(testKeys))
    fmt.Printf("数据迁移率: %.2f%% (%d/%d)\n", 
        migrationRate*100, migrationCount, len(testKeys))
}
```

### 3. 负载均衡测试

```go
func testLoadBalance() {
    // 测试不同虚拟节点数量的效果
    replicasCounts := []int{10, 50, 150, 500}
    
    for _, replicas := range replicasCounts {
        fmt.Printf("\n测试虚拟节点数量: %d\n", replicas)
        
        hashMap := NewConsistentHashMap(replicas, nil)
        hashMap.Add("server1", "server2", "server3", "server4", "server5")
        
        // 生成大量测试键
        testKeys := make([]string, 10000)
        for i := 0; i < 10000; i++ {
            testKeys[i] = fmt.Sprintf("key_%d", i)
        }
        
        // 统计分布
        distribution := make(map[string]int)
        for _, key := range testKeys {
            server, _ := hashMap.Get(key)
            distribution[server]++
        }
        
        // 计算负载均衡度
        var counts []int
        for server, count := range distribution {
            fmt.Printf("  %s: %d (%.2f%%)\n", 
                server, count, float64(count)/float64(len(testKeys))*100)
            counts = append(counts, count)
        }
        
        // 计算标准差
        avg := float64(len(testKeys)) / float64(len(distribution))
        variance := 0.0
        for _, count := range counts {
            diff := float64(count) - avg
            variance += diff * diff
        }
        variance /= float64(len(counts))
        stdDev := math.Sqrt(variance)
        
        fmt.Printf("  标准差: %.2f (越小越均衡)\n", stdDev)
    }
}
```

### 4. 性能基准测试

```go
func BenchmarkConsistentHashMap(b *testing.B) {
    hashMap := NewConsistentHashMap(150, nil)
    hashMap.Add("server1", "server2", "server3", "server4", "server5")
    
    b.Run("Get", func(b *testing.B) {
        keys := make([]string, 1000)
        for i := 0; i < 1000; i++ {
            keys[i] = fmt.Sprintf("key_%d", i)
        }
        
        b.ResetTimer()
        for i := 0; i < b.N; i++ {
            key := keys[i%1000]
            _, _ = hashMap.Get(key)
        }
    })
    
    b.Run("Add", func(b *testing.B) {
        b.ResetTimer()
        for i := 0; i < b.N; i++ {
            server := fmt.Sprintf("server_%d", i)
            hashMap.Add(server)
        }
    })
}
```

## 注意事项

### 1. 虚拟节点数量选择

```go
// ✅ 推荐：根据节点数量和负载均衡要求选择
func calculateOptimalReplicas(nodeCount int) int {
    if nodeCount <= 3 {
        return 150 // 少量节点需要更多虚拟节点
    } else if nodeCount <= 10 {
        return 100
    } else {
        return 50  // 大量节点可以减少虚拟节点
    }
}

// ❌ 避免：虚拟节点过少导致负载不均
hashMap := NewConsistentHashMap(3, nil) // 太少

// ❌ 避免：虚拟节点过多导致内存浪费
hashMap := NewConsistentHashMap(1000, nil) // 太多
```

### 2. 哈希函数选择

```go
// ✅ 推荐：使用高质量的哈希函数
import "github.com/cespare/xxhash/v2"

func xxHash(data []byte) uint32 {
    return uint32(xxhash.Sum64(data))
}

hashMap := NewConsistentHashMap(150, xxHash)

// ❌ 避免：使用质量差的哈希函数
func badHash(data []byte) uint32 {
    return uint32(len(data)) // 质量很差的哈希函数
}
```

### 3. 并发访问

```go
// ✅ 推荐：一致性哈希映射是线程安全的
go func() {
    hashMap.Add("new_server")
}()

go func() {
    server, _ := hashMap.Get("some_key")
}()

// ❌ 避免：不必要的外部同步
var mu sync.Mutex
mu.Lock()
server, _ := hashMap.Get("key") // 不必要的锁
mu.Unlock()
```

### 4. 节点管理

```go
// ✅ 推荐：批量操作提高性能
servers := []string{"server1", "server2", "server3"}
hashMap.AddBatch(servers)

// ❌ 避免：频繁的单个操作
for _, server := range servers {
    hashMap.Add(server) // 每次都会重新排序
}
```
