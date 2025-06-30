package consistent_hash

import (
	"fmt"
	"sync"

	"golang.org/x/sync/singleflight"

	domainHash "github.com/justinwongcn/hamster/internal/domain/consistent_hash"
)

// SingleflightPeerPicker 带singleflight优化的节点选择器
// 结合singleflight优化节点选择过程，确保相同key的并发请求只执行一次节点选择
type SingleflightPeerPicker struct {
	consistentHash domainHash.ConsistentHash
	peers          map[string]domainHash.Peer // 节点ID到节点实例的映射
	mu             sync.RWMutex               // 保护peers映射
	g              singleflight.Group         // singleflight组
}

// NewSingleflightPeerPicker 创建带singleflight优化的节点选择器
// consistentHash: 一致性哈希实现
// 返回: SingleflightPeerPicker实例
func NewSingleflightPeerPicker(consistentHash domainHash.ConsistentHash) *SingleflightPeerPicker {
	return &SingleflightPeerPicker{
		consistentHash: consistentHash,
		peers:          make(map[string]domainHash.Peer),
		g:              singleflight.Group{},
	}
}

// PickPeer 根据键选择节点（带singleflight优化）
// key: 要查找的键
// 返回: 选中的节点和错误信息
func (p *SingleflightPeerPicker) PickPeer(key string) (domainHash.Peer, error) {
	// 使用singleflight确保相同key的并发请求只执行一次节点选择
	result, err, _ := p.g.Do(key, func() (interface{}, error) {
		return p.pickPeerInternal(key)
	})
	
	if err != nil {
		return nil, err
	}
	
	return result.(domainHash.Peer), nil
}

// PickPeers 选择多个节点（用于副本）
// key: 要查找的键
// count: 需要的节点数量
// 返回: 选中的节点列表和错误信息
func (p *SingleflightPeerPicker) PickPeers(key string, count int) ([]domainHash.Peer, error) {
	// 为多节点选择生成唯一的singleflight key
	sfKey := fmt.Sprintf("%s#%d", key, count)
	
	result, err, _ := p.g.Do(sfKey, func() (interface{}, error) {
		return p.pickPeersInternal(key, count)
	})
	
	if err != nil {
		return nil, err
	}
	
	return result.([]domainHash.Peer), nil
}

// AddPeers 添加节点
// peers: 要添加的节点列表
func (p *SingleflightPeerPicker) AddPeers(peers ...domainHash.Peer) {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	peerIDs := make([]string, len(peers))
	for i, peer := range peers {
		peerIDs[i] = peer.ID()
		p.peers[peer.ID()] = peer
	}
	
	// 添加到一致性哈希
	p.consistentHash.Add(peerIDs...)
}

// RemovePeers 移除节点
// peers: 要移除的节点列表
func (p *SingleflightPeerPicker) RemovePeers(peers ...domainHash.Peer) {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	peerIDs := make([]string, len(peers))
	for i, peer := range peers {
		peerIDs[i] = peer.ID()
		delete(p.peers, peer.ID())
	}
	
	// 从一致性哈希中移除
	p.consistentHash.Remove(peerIDs...)
}

// GetAllPeers 获取所有节点
// 返回: 所有节点的列表
func (p *SingleflightPeerPicker) GetAllPeers() []domainHash.Peer {
	p.mu.RLock()
	defer p.mu.RUnlock()
	
	result := make([]domainHash.Peer, 0, len(p.peers))
	for _, peer := range p.peers {
		result = append(result, peer)
	}
	
	return result
}

// IsHealthy 检查节点选择器是否健康
// 返回: 是否健康和错误信息
func (p *SingleflightPeerPicker) IsHealthy() (bool, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	
	if len(p.peers) == 0 {
		return false, domainHash.ErrNoPeers
	}
	
	// 检查是否有存活的节点
	aliveCount := 0
	for _, peer := range p.peers {
		if peer.IsAlive() {
			aliveCount++
		}
	}
	
	if aliveCount == 0 {
		return false, fmt.Errorf("没有存活的节点")
	}
	
	return true, nil
}

// pickPeerInternal 内部节点选择逻辑
// key: 要查找的键
// 返回: 选中的节点和错误信息
func (p *SingleflightPeerPicker) pickPeerInternal(key string) (domainHash.Peer, error) {
	// 从一致性哈希获取节点ID
	peerID, err := p.consistentHash.Get(key)
	if err != nil {
		return nil, err
	}
	
	p.mu.RLock()
	peer, exists := p.peers[peerID]
	p.mu.RUnlock()
	
	if !exists {
		return nil, fmt.Errorf("节点 %s 不存在", peerID)
	}
	
	// 检查节点是否存活
	if !peer.IsAlive() {
		// 如果节点不存活，尝试获取其他节点
		return p.pickAlternativePeer(key, peerID)
	}
	
	return peer, nil
}

// pickPeersInternal 内部多节点选择逻辑
// key: 要查找的键
// count: 需要的节点数量
// 返回: 选中的节点列表和错误信息
func (p *SingleflightPeerPicker) pickPeersInternal(key string, count int) ([]domainHash.Peer, error) {
	// 从一致性哈希获取多个节点ID
	peerIDs, err := p.consistentHash.GetMultiple(key, count)
	if err != nil {
		return nil, err
	}
	
	p.mu.RLock()
	defer p.mu.RUnlock()
	
	result := make([]domainHash.Peer, 0, len(peerIDs))
	for _, peerID := range peerIDs {
		if peer, exists := p.peers[peerID]; exists && peer.IsAlive() {
			result = append(result, peer)
		}
	}
	
	if len(result) == 0 {
		return nil, fmt.Errorf("没有可用的节点")
	}
	
	return result, nil
}

// pickAlternativePeer 选择替代节点
// key: 原始键
// excludePeerID: 要排除的节点ID
// 返回: 替代节点和错误信息
func (p *SingleflightPeerPicker) pickAlternativePeer(key, excludePeerID string) (domainHash.Peer, error) {
	// 尝试获取多个节点，排除不可用的节点
	peerIDs, err := p.consistentHash.GetMultiple(key, 5) // 最多尝试5个节点
	if err != nil {
		return nil, err
	}
	
	p.mu.RLock()
	defer p.mu.RUnlock()
	
	for _, peerID := range peerIDs {
		if peerID == excludePeerID {
			continue // 跳过原始节点
		}
		
		if peer, exists := p.peers[peerID]; exists && peer.IsAlive() {
			return peer, nil
		}
	}
	
	return nil, fmt.Errorf("没有可用的替代节点")
}

// GetStats 获取统计信息
// 返回: 统计信息
func (p *SingleflightPeerPicker) GetStats() domainHash.HashStats {
	return p.consistentHash.Stats()
}

// ForgetKey 忘记指定key的singleflight缓存
// key: 要忘记的键
func (p *SingleflightPeerPicker) ForgetKey(key string) {
	p.g.Forget(key)
}

// ForgetMultipleKey 忘记多节点选择的singleflight缓存
// key: 原始键
// count: 节点数量
func (p *SingleflightPeerPicker) ForgetMultipleKey(key string, count int) {
	sfKey := fmt.Sprintf("%s#%d", key, count)
	p.g.Forget(sfKey)
}

// UpdatePeerStatus 更新节点状态
// peerID: 节点ID
// alive: 是否存活
// 返回: 操作错误
func (p *SingleflightPeerPicker) UpdatePeerStatus(peerID string, alive bool) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	peer, exists := p.peers[peerID]
	if !exists {
		return fmt.Errorf("节点 %s 不存在", peerID)
	}
	
	// 如果节点实现了状态更新接口，则更新状态
	if peerInfo, ok := peer.(domainHash.PeerInfo); ok {
		p.peers[peerID] = peerInfo.SetAlive(alive)
	}
	
	return nil
}

// GetPeerByID 根据ID获取节点
// peerID: 节点ID
// 返回: 节点实例和是否存在
func (p *SingleflightPeerPicker) GetPeerByID(peerID string) (domainHash.Peer, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	
	peer, exists := p.peers[peerID]
	return peer, exists
}

// GetConsistentHash 获取一致性哈希实例（用于测试）
// 返回: 一致性哈希实例
func (p *SingleflightPeerPicker) GetConsistentHash() domainHash.ConsistentHash {
	return p.consistentHash
}

// GetPeerCount 获取节点数量
// 返回: 节点数量
func (p *SingleflightPeerPicker) GetPeerCount() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	
	return len(p.peers)
}

// GetAlivePeerCount 获取存活节点数量
// 返回: 存活节点数量
func (p *SingleflightPeerPicker) GetAlivePeerCount() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	
	count := 0
	for _, peer := range p.peers {
		if peer.IsAlive() {
			count++
		}
	}
	
	return count
}
