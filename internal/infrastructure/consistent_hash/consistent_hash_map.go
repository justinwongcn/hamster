package consistent_hash

import (
	"fmt"
	"hash/crc32"
	"sort"
	"sync"

	domainHash "github.com/justinwongcn/hamster/internal/domain/consistent_hash"
)

// ConsistentHashMap 一致性哈希算法的主数据结构
// 包含4个成员变量：Hash函数hash；虚拟节点倍数replicas；哈希环keys；虚拟节点与真实节点的映射表hashMap
type ConsistentHashMap struct {
	hash     domainHash.Hash   // Hash函数
	replicas int               // 虚拟节点倍数
	keys     []uint32          // 哈希环（排序的哈希值列表）
	hashMap  map[uint32]string // 虚拟节点与真实节点的映射表，键是虚拟节点的哈希值，值是真实节点的名称
	mu       sync.RWMutex      // 读写锁保护
}

// NewConsistentHashMap 构造函数，允许自定义虚拟节点倍数和Hash函数
// replicas: 虚拟节点倍数
// hashFunc: Hash函数，如果为nil则使用默认的crc32.ChecksumIEEE
// 返回: ConsistentHashMap实例
func NewConsistentHashMap(replicas int, hashFunc domainHash.Hash) *ConsistentHashMap {
	if hashFunc == nil {
		hashFunc = crc32.ChecksumIEEE
	}

	return &ConsistentHashMap{
		hash:     hashFunc,
		replicas: replicas,
		keys:     make([]uint32, 0),
		hashMap:  make(map[uint32]string),
	}
}

// Add 添加节点到哈希环
// peers: 要添加的节点列表
func (m *ConsistentHashMap) Add(peers ...string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, peer := range peers {
		// 为每个真实节点创建replicas个虚拟节点
		for i := 0; i < m.replicas; i++ {
			// 生成虚拟节点的键
			virtualKey := m.generateVirtualNodeKey(peer, i)
			// 计算虚拟节点的哈希值
			hash := m.hash([]byte(virtualKey))
			// 添加到哈希环
			m.keys = append(m.keys, hash)
			// 建立虚拟节点到真实节点的映射
			m.hashMap[hash] = peer
		}
	}

	// 保持哈希环有序
	sort.Slice(m.keys, func(i, j int) bool {
		return m.keys[i] < m.keys[j]
	})
}

// Remove 从哈希环中移除节点
// peers: 要移除的节点列表
func (m *ConsistentHashMap) Remove(peers ...string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, peer := range peers {
		// 移除该节点的所有虚拟节点
		for i := 0; i < m.replicas; i++ {
			virtualKey := m.generateVirtualNodeKey(peer, i)
			hash := m.hash([]byte(virtualKey))

			// 从映射表中删除
			delete(m.hashMap, hash)

			// 从哈希环中删除
			for j, key := range m.keys {
				if key == hash {
					m.keys = append(m.keys[:j], m.keys[j+1:]...)
					break
				}
			}
		}
	}
}

// Get 根据键获取对应的节点
// key: 要查找的键
// 返回: 对应的节点名称和错误信息
func (m *ConsistentHashMap) Get(key string) (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if len(m.keys) == 0 {
		return "", domainHash.ErrNoPeers
	}

	// 计算键的哈希值
	hash := m.hash([]byte(key))

	// 在哈希环上顺时针查找第一个大于等于hash的虚拟节点
	idx := sort.Search(len(m.keys), func(i int) bool {
		return m.keys[i] >= hash
	})

	// 如果没找到，说明应该选择第一个节点（环形结构）
	if idx == len(m.keys) {
		idx = 0
	}

	// 返回对应的真实节点
	return m.hashMap[m.keys[idx]], nil
}

// GetMultiple 获取多个节点（用于副本）
// key: 要查找的键
// count: 需要的节点数量
// 返回: 节点列表和错误信息
func (m *ConsistentHashMap) GetMultiple(key string, count int) ([]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if len(m.keys) == 0 {
		return nil, domainHash.ErrNoPeers
	}

	if count <= 0 {
		return []string{}, nil
	}

	// 计算键的哈希值
	hash := m.hash([]byte(key))

	// 在哈希环上顺时针查找
	idx := sort.Search(len(m.keys), func(i int) bool {
		return m.keys[i] >= hash
	})

	if idx == len(m.keys) {
		idx = 0
	}

	// 收集不同的真实节点
	seen := make(map[string]bool)
	result := make([]string, 0, count)

	// 在哈希环上顺时针遍历，直到找到足够的不同节点
	for len(result) < count && len(seen) < len(m.getAllRealPeers()) {
		peer := m.hashMap[m.keys[idx]]
		if !seen[peer] {
			result = append(result, peer)
			seen[peer] = true
		}
		idx = (idx + 1) % len(m.keys)
	}

	return result, nil
}

// Peers 获取所有节点
// 返回: 所有节点的列表
func (m *ConsistentHashMap) Peers() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.getAllRealPeers()
}

// IsEmpty 检查是否为空
// 返回: 是否没有节点
func (m *ConsistentHashMap) IsEmpty() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return len(m.keys) == 0
}

// Stats 获取统计信息
// 返回: 统计信息
func (m *ConsistentHashMap) Stats() domainHash.HashStats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	peers := m.getAllRealPeers()
	keyDistribution := make(map[string]int)

	// 初始化每个节点的计数
	for _, peer := range peers {
		keyDistribution[peer] = 0
	}

	// 统计每个真实节点对应的虚拟节点数量
	for _, peer := range m.hashMap {
		keyDistribution[peer]++
	}

	return domainHash.NewHashStats(len(peers), len(m.keys), m.replicas, keyDistribution)
}

// generateVirtualNodeKey 生成虚拟节点的键
// peer: 真实节点名称
// index: 虚拟节点索引
// 返回: 虚拟节点键
func (m *ConsistentHashMap) generateVirtualNodeKey(peer string, index int) string {
	return fmt.Sprintf("%s#%d", peer, index)
}

// getAllRealPeers 获取所有真实节点（去重）
// 返回: 真实节点列表
func (m *ConsistentHashMap) getAllRealPeers() []string {
	seen := make(map[string]bool)
	result := make([]string, 0)

	for _, peer := range m.hashMap {
		if !seen[peer] {
			result = append(result, peer)
			seen[peer] = true
		}
	}

	// 保持结果有序，便于测试
	sort.Strings(result)
	return result
}

// GetHashFunc 获取哈希函数（用于测试）
// 返回: 哈希函数
func (m *ConsistentHashMap) GetHashFunc() domainHash.Hash {
	return m.hash
}

// GetReplicas 获取虚拟节点倍数（用于测试）
// 返回: 虚拟节点倍数
func (m *ConsistentHashMap) GetReplicas() int {
	return m.replicas
}

// GetKeys 获取哈希环的键列表（用于测试）
// 返回: 哈希环键列表的副本
func (m *ConsistentHashMap) GetKeys() []uint32 {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]uint32, len(m.keys))
	copy(result, m.keys)
	return result
}

// GetHashMap 获取虚拟节点映射表（用于测试）
// 返回: 映射表的副本
func (m *ConsistentHashMap) GetHashMap() map[uint32]string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make(map[uint32]string)
	for k, v := range m.hashMap {
		result[k] = v
	}
	return result
}

// Clone 克隆一致性哈希映射
// 返回: 新的一致性哈希映射实例
func (m *ConsistentHashMap) Clone() *ConsistentHashMap {
	m.mu.RLock()
	defer m.mu.RUnlock()

	newMap := &ConsistentHashMap{
		hash:     m.hash,
		replicas: m.replicas,
		keys:     make([]uint32, len(m.keys)),
		hashMap:  make(map[uint32]string),
	}

	copy(newMap.keys, m.keys)
	for k, v := range m.hashMap {
		newMap.hashMap[k] = v
	}

	return newMap
}

// Reset 重置一致性哈希映射
func (m *ConsistentHashMap) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.keys = make([]uint32, 0)
	m.hashMap = make(map[uint32]string)
}

// GetVirtualNodeCount 获取指定节点的虚拟节点数量
// peer: 节点名称
// 返回: 虚拟节点数量
func (m *ConsistentHashMap) GetVirtualNodeCount(peer string) int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	count := 0
	for _, p := range m.hashMap {
		if p == peer {
			count++
		}
	}
	return count
}

// GetLoadDistribution 获取负载分布情况
// testKeys: 测试键列表
// 返回: 每个节点分配到的键数量
func (m *ConsistentHashMap) GetLoadDistribution(testKeys []string) map[string]int {
	distribution := make(map[string]int)

	for _, key := range testKeys {
		peer, err := m.Get(key)
		if err == nil {
			distribution[peer]++
		}
	}

	return distribution
}
