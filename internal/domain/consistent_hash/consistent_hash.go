package consistent_hash

import (
	"errors"
	"fmt"
	"hash/crc32"
	"sort"
)

var (
	// ErrNoPeers 没有可用节点错误
	ErrNoPeers = errors.New("没有可用的节点")
	// ErrInvalidKey 无效键错误
	ErrInvalidKey = errors.New("无效的键")
	// ErrInvalidPeer 无效节点错误
	ErrInvalidPeer = errors.New("无效的节点")
	// ErrInvalidReplicas 无效虚拟节点倍数错误
	ErrInvalidReplicas = errors.New("无效的虚拟节点倍数")
)

// Hash 哈希函数类型
// 采用依赖注入的方式，允许替换成自定义的Hash函数
// 默认为 crc32.ChecksumIEEE 算法
type Hash func(data []byte) uint32

// ConsistentHash 一致性哈希算法接口
// 定义一致性哈希的核心操作
type ConsistentHash interface {
	// Add 添加节点到哈希环
	// peers: 要添加的节点列表
	Add(peers ...string)
	
	// Remove 从哈希环中移除节点
	// peers: 要移除的节点列表
	Remove(peers ...string)
	
	// Get 根据键获取对应的节点
	// key: 要查找的键
	// 返回: 对应的节点名称和错误信息
	Get(key string) (string, error)
	
	// GetMultiple 获取多个节点（用于副本）
	// key: 要查找的键
	// count: 需要的节点数量
	// 返回: 节点列表和错误信息
	GetMultiple(key string, count int) ([]string, error)
	
	// Peers 获取所有节点
	// 返回: 所有节点的列表
	Peers() []string
	
	// IsEmpty 检查是否为空
	// 返回: 是否没有节点
	IsEmpty() bool
	
	// Stats 获取统计信息
	// 返回: 统计信息
	Stats() HashStats
}

// PeerPicker 分布式节点选择器接口
// 抽象分布式节点的选择逻辑
type PeerPicker interface {
	// PickPeer 根据键选择节点
	// key: 要查找的键
	// 返回: 选中的节点和错误信息
	PickPeer(key string) (Peer, error)
	
	// PickPeers 选择多个节点（用于副本）
	// key: 要查找的键
	// count: 需要的节点数量
	// 返回: 选中的节点列表和错误信息
	PickPeers(key string, count int) ([]Peer, error)
	
	// AddPeers 添加节点
	// peers: 要添加的节点列表
	AddPeers(peers ...Peer)
	
	// RemovePeers 移除节点
	// peers: 要移除的节点列表
	RemovePeers(peers ...Peer)
	
	// GetAllPeers 获取所有节点
	// 返回: 所有节点的列表
	GetAllPeers() []Peer
	
	// IsHealthy 检查节点选择器是否健康
	// 返回: 是否健康和错误信息
	IsHealthy() (bool, error)
}

// Peer 节点接口
// 代表一个分布式系统中的节点
type Peer interface {
	// ID 获取节点唯一标识
	ID() string
	
	// Address 获取节点地址
	Address() string
	
	// IsAlive 检查节点是否存活
	IsAlive() bool
	
	// Weight 获取节点权重
	Weight() int
	
	// Equals 比较两个节点是否相等
	Equals(other Peer) bool
}

// HashKey 哈希键值对象
// 封装哈希键的业务规则和验证逻辑
type HashKey struct {
	value string
}

// NewHashKey 创建新的哈希键
// key: 键值字符串
// 返回: HashKey实例和错误信息
func NewHashKey(key string) (HashKey, error) {
	if key == "" {
		return HashKey{}, fmt.Errorf("%w: 哈希键不能为空", ErrInvalidKey)
	}
	if len(key) > 500 {
		return HashKey{}, fmt.Errorf("%w: 哈希键长度不能超过500个字符", ErrInvalidKey)
	}
	return HashKey{value: key}, nil
}

// String 返回哈希键的字符串表示
func (k HashKey) String() string {
	return k.value
}

// Bytes 返回哈希键的字节表示
func (k HashKey) Bytes() []byte {
	return []byte(k.value)
}

// Hash 计算哈希键的哈希值
// hashFunc: 哈希函数
// 返回: 哈希值
func (k HashKey) Hash(hashFunc Hash) uint32 {
	return hashFunc(k.Bytes())
}

// Equals 比较两个哈希键是否相等
func (k HashKey) Equals(other HashKey) bool {
	return k.value == other.value
}

// PeerInfo 节点信息值对象
// 封装节点的基本信息
type PeerInfo struct {
	id      string
	address string
	weight  int
	alive   bool
}

// NewPeerInfo 创建新的节点信息
// id: 节点ID
// address: 节点地址
// weight: 节点权重
// 返回: PeerInfo实例和错误信息
func NewPeerInfo(id, address string, weight int) (PeerInfo, error) {
	if id == "" {
		return PeerInfo{}, fmt.Errorf("%w: 节点ID不能为空", ErrInvalidPeer)
	}
	if address == "" {
		return PeerInfo{}, fmt.Errorf("%w: 节点地址不能为空", ErrInvalidPeer)
	}
	if weight < 0 {
		return PeerInfo{}, fmt.Errorf("%w: 节点权重不能为负数", ErrInvalidPeer)
	}
	return PeerInfo{
		id:      id,
		address: address,
		weight:  weight,
		alive:   true,
	}, nil
}

// ID 获取节点唯一标识
func (p PeerInfo) ID() string {
	return p.id
}

// Address 获取节点地址
func (p PeerInfo) Address() string {
	return p.address
}

// IsAlive 检查节点是否存活
func (p PeerInfo) IsAlive() bool {
	return p.alive
}

// Weight 获取节点权重
func (p PeerInfo) Weight() int {
	return p.weight
}

// Equals 比较两个节点是否相等
func (p PeerInfo) Equals(other Peer) bool {
	return p.id == other.ID()
}

// SetAlive 设置节点存活状态
func (p PeerInfo) SetAlive(alive bool) PeerInfo {
	p.alive = alive
	return p
}

// HashStats 哈希统计信息值对象
// 封装一致性哈希的统计数据
type HashStats struct {
	totalPeers     int
	virtualNodes   int
	replicas       int
	keyDistribution map[string]int // 每个节点分配到的键数量
}

// NewHashStats 创建新的哈希统计信息
// totalPeers: 总节点数
// virtualNodes: 虚拟节点数
// replicas: 虚拟节点倍数
// keyDistribution: 键分布情况
func NewHashStats(totalPeers, virtualNodes, replicas int, keyDistribution map[string]int) HashStats {
	return HashStats{
		totalPeers:      totalPeers,
		virtualNodes:    virtualNodes,
		replicas:        replicas,
		keyDistribution: keyDistribution,
	}
}

// TotalPeers 获取总节点数
func (s HashStats) TotalPeers() int {
	return s.totalPeers
}

// VirtualNodes 获取虚拟节点数
func (s HashStats) VirtualNodes() int {
	return s.virtualNodes
}

// Replicas 获取虚拟节点倍数
func (s HashStats) Replicas() int {
	return s.replicas
}

// KeyDistribution 获取键分布情况
func (s HashStats) KeyDistribution() map[string]int {
	result := make(map[string]int)
	for k, v := range s.keyDistribution {
		result[k] = v
	}
	return result
}

// LoadBalance 计算负载均衡度
// 返回标准差，值越小表示负载越均衡
func (s HashStats) LoadBalance() float64 {
	if len(s.keyDistribution) == 0 {
		return 0
	}
	
	// 计算平均值
	total := 0
	for _, count := range s.keyDistribution {
		total += count
	}
	avg := float64(total) / float64(len(s.keyDistribution))
	
	// 计算方差
	variance := 0.0
	for _, count := range s.keyDistribution {
		diff := float64(count) - avg
		variance += diff * diff
	}
	variance /= float64(len(s.keyDistribution))
	
	// 返回标准差
	return variance
}

// VirtualNodeConfig 虚拟节点配置值对象
// 封装虚拟节点的配置参数
type VirtualNodeConfig struct {
	replicas int
	hashFunc Hash
}

// NewVirtualNodeConfig 创建新的虚拟节点配置
// replicas: 虚拟节点倍数
// hashFunc: 哈希函数（可选，默认为crc32.ChecksumIEEE）
// 返回: VirtualNodeConfig实例和错误信息
func NewVirtualNodeConfig(replicas int, hashFunc Hash) (VirtualNodeConfig, error) {
	if replicas <= 0 {
		return VirtualNodeConfig{}, fmt.Errorf("%w: 虚拟节点倍数必须大于0", ErrInvalidReplicas)
	}
	if replicas > 1000 {
		return VirtualNodeConfig{}, fmt.Errorf("%w: 虚拟节点倍数不能超过1000", ErrInvalidReplicas)
	}
	
	if hashFunc == nil {
		hashFunc = crc32.ChecksumIEEE
	}
	
	return VirtualNodeConfig{
		replicas: replicas,
		hashFunc: hashFunc,
	}, nil
}

// Replicas 获取虚拟节点倍数
func (c VirtualNodeConfig) Replicas() int {
	return c.replicas
}

// HashFunc 获取哈希函数
func (c VirtualNodeConfig) HashFunc() Hash {
	return c.hashFunc
}

// GenerateVirtualNodeKeys 生成虚拟节点的键
// peer: 真实节点名称
// 返回: 虚拟节点键列表
func (c VirtualNodeConfig) GenerateVirtualNodeKeys(peer string) []string {
	keys := make([]string, c.replicas)
	for i := 0; i < c.replicas; i++ {
		keys[i] = fmt.Sprintf("%s#%d", peer, i)
	}
	return keys
}

// HashRing 哈希环值对象
// 封装哈希环的数据结构和操作
type HashRing struct {
	keys     []uint32          // 排序的哈希值列表
	hashMap  map[uint32]string // 虚拟节点哈希值到真实节点的映射
	config   VirtualNodeConfig // 虚拟节点配置
}

// NewHashRing 创建新的哈希环
// config: 虚拟节点配置
// 返回: HashRing实例
func NewHashRing(config VirtualNodeConfig) HashRing {
	return HashRing{
		keys:    make([]uint32, 0),
		hashMap: make(map[uint32]string),
		config:  config,
	}
}

// AddPeer 添加节点到哈希环
// peer: 要添加的节点名称
func (r HashRing) AddPeer(peer string) HashRing {
	// 为每个真实节点创建多个虚拟节点
	virtualKeys := r.config.GenerateVirtualNodeKeys(peer)
	
	newKeys := make([]uint32, len(r.keys))
	copy(newKeys, r.keys)
	
	newHashMap := make(map[uint32]string)
	for k, v := range r.hashMap {
		newHashMap[k] = v
	}
	
	for _, vKey := range virtualKeys {
		hash := r.config.HashFunc()([]byte(vKey))
		newKeys = append(newKeys, hash)
		newHashMap[hash] = peer
	}
	
	// 保持哈希环有序
	sort.Slice(newKeys, func(i, j int) bool {
		return newKeys[i] < newKeys[j]
	})
	
	return HashRing{
		keys:    newKeys,
		hashMap: newHashMap,
		config:  r.config,
	}
}

// RemovePeer 从哈希环中移除节点
// peer: 要移除的节点名称
func (r HashRing) RemovePeer(peer string) HashRing {
	virtualKeys := r.config.GenerateVirtualNodeKeys(peer)
	
	newHashMap := make(map[uint32]string)
	for k, v := range r.hashMap {
		newHashMap[k] = v
	}
	
	// 移除虚拟节点
	toRemove := make(map[uint32]bool)
	for _, vKey := range virtualKeys {
		hash := r.config.HashFunc()([]byte(vKey))
		delete(newHashMap, hash)
		toRemove[hash] = true
	}
	
	// 重建keys列表
	newKeys := make([]uint32, 0, len(r.keys))
	for _, key := range r.keys {
		if !toRemove[key] {
			newKeys = append(newKeys, key)
		}
	}
	
	return HashRing{
		keys:    newKeys,
		hashMap: newHashMap,
		config:  r.config,
	}
}

// GetPeer 根据键获取对应的节点
// key: 要查找的键
// 返回: 对应的节点名称和是否找到
func (r HashRing) GetPeer(key string) (string, bool) {
	if len(r.keys) == 0 {
		return "", false
	}
	
	hash := r.config.HashFunc()([]byte(key))
	
	// 在哈希环上顺时针查找第一个大于等于hash的节点
	idx := sort.Search(len(r.keys), func(i int) bool {
		return r.keys[i] >= hash
	})
	
	// 如果没找到，说明应该选择第一个节点（环形结构）
	if idx == len(r.keys) {
		idx = 0
	}
	
	peer, exists := r.hashMap[r.keys[idx]]
	return peer, exists
}

// GetMultiplePeers 获取多个不同的节点
// key: 要查找的键
// count: 需要的节点数量
// 返回: 节点列表
func (r HashRing) GetMultiplePeers(key string, count int) []string {
	if len(r.keys) == 0 || count <= 0 {
		return []string{}
	}
	
	hash := r.config.HashFunc()([]byte(key))
	
	// 在哈希环上顺时针查找
	idx := sort.Search(len(r.keys), func(i int) bool {
		return r.keys[i] >= hash
	})
	
	if idx == len(r.keys) {
		idx = 0
	}
	
	seen := make(map[string]bool)
	result := make([]string, 0, count)
	
	for len(result) < count && len(seen) < len(r.hashMap) {
		if peer, exists := r.hashMap[r.keys[idx]]; exists && !seen[peer] {
			result = append(result, peer)
			seen[peer] = true
		}
		idx = (idx + 1) % len(r.keys)
	}
	
	return result
}

// IsEmpty 检查哈希环是否为空
func (r HashRing) IsEmpty() bool {
	return len(r.keys) == 0
}

// GetAllPeers 获取所有真实节点
func (r HashRing) GetAllPeers() []string {
	seen := make(map[string]bool)
	result := make([]string, 0)
	
	for _, peer := range r.hashMap {
		if !seen[peer] {
			result = append(result, peer)
			seen[peer] = true
		}
	}
	
	return result
}

// Stats 获取统计信息
func (r HashRing) Stats() HashStats {
	peers := r.GetAllPeers()
	keyDistribution := make(map[string]int)
	
	for _, peer := range peers {
		keyDistribution[peer] = 0
	}
	
	// 统计每个真实节点对应的虚拟节点数量
	for _, peer := range r.hashMap {
		keyDistribution[peer]++
	}
	
	return NewHashStats(len(peers), len(r.keys), r.config.Replicas(), keyDistribution)
}
