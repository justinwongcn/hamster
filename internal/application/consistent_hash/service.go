package consistent_hash

import (
	"context"
	"fmt"

	domainHash "github.com/justinwongcn/hamster/internal/domain/consistent_hash"
)

// ConsistentHashApplicationService 一致性哈希应用服务
// 协调领域服务和基础设施，实现具体的一致性哈希业务用例
type ConsistentHashApplicationService struct {
	peerPicker domainHash.PeerPicker
}

// NewConsistentHashApplicationService 创建一致性哈希应用服务
// peerPicker: 节点选择器实现
func NewConsistentHashApplicationService(peerPicker domainHash.PeerPicker) *ConsistentHashApplicationService {
	return &ConsistentHashApplicationService{
		peerPicker: peerPicker,
	}
}

// PeerSelectionCommand 节点选择命令
type PeerSelectionCommand struct {
	Key string `json:"key"`
}

// MultiplePeerSelectionCommand 多节点选择命令
type MultiplePeerSelectionCommand struct {
	Key   string `json:"key"`
	Count int    `json:"count"`
}

// AddPeersCommand 添加节点命令
type AddPeersCommand struct {
	Peers []PeerRequest `json:"peers"`
}

// RemovePeersCommand 移除节点命令
type RemovePeersCommand struct {
	PeerIDs []string `json:"peer_ids"`
}

// PeerRequest 节点请求
type PeerRequest struct {
	ID      string `json:"id"`
	Address string `json:"address"`
	Weight  int    `json:"weight"`
}

// PeerResult 节点结果
type PeerResult struct {
	ID      string `json:"id"`
	Address string `json:"address"`
	Weight  int    `json:"weight"`
	IsAlive bool   `json:"is_alive"`
}

// PeerSelectionResult 节点选择结果
type PeerSelectionResult struct {
	Key  string     `json:"key"`
	Peer PeerResult `json:"peer"`
}

// MultiplePeerSelectionResult 多节点选择结果
type MultiplePeerSelectionResult struct {
	Key   string       `json:"key"`
	Peers []PeerResult `json:"peers"`
	Count int          `json:"count"`
}

// HashStatsResult 哈希统计结果
type HashStatsResult struct {
	TotalPeers      int                `json:"total_peers"`
	VirtualNodes    int                `json:"virtual_nodes"`
	Replicas        int                `json:"replicas"`
	KeyDistribution map[string]int     `json:"key_distribution"`
	LoadBalance     float64            `json:"load_balance"`
}

// HealthCheckResult 健康检查结果
type HealthCheckResult struct {
	IsHealthy bool   `json:"is_healthy"`
	Message   string `json:"message"`
}

// SelectPeer 选择节点
// 用例：用户想要根据键选择一个节点来处理请求
func (s *ConsistentHashApplicationService) SelectPeer(ctx context.Context, cmd PeerSelectionCommand) (*PeerSelectionResult, error) {
	// 验证输入
	if err := s.validatePeerSelectionCommand(cmd); err != nil {
		return nil, fmt.Errorf("验证节点选择命令失败: %w", err)
	}

	// 选择节点
	peer, err := s.peerPicker.PickPeer(cmd.Key)
	if err != nil {
		return nil, fmt.Errorf("选择节点失败: %w", err)
	}

	return &PeerSelectionResult{
		Key:  cmd.Key,
		Peer: s.buildPeerResult(peer),
	}, nil
}

// SelectMultiplePeers 选择多个节点
// 用例：用户想要选择多个节点来实现数据副本或负载分担
func (s *ConsistentHashApplicationService) SelectMultiplePeers(ctx context.Context, cmd MultiplePeerSelectionCommand) (*MultiplePeerSelectionResult, error) {
	// 验证输入
	if err := s.validateMultiplePeerSelectionCommand(cmd); err != nil {
		return nil, fmt.Errorf("验证多节点选择命令失败: %w", err)
	}

	// 选择多个节点
	peers, err := s.peerPicker.PickPeers(cmd.Key, cmd.Count)
	if err != nil {
		return nil, fmt.Errorf("选择多个节点失败: %w", err)
	}

	peerResults := make([]PeerResult, len(peers))
	for i, peer := range peers {
		peerResults[i] = s.buildPeerResult(peer)
	}

	return &MultiplePeerSelectionResult{
		Key:   cmd.Key,
		Peers: peerResults,
		Count: len(peerResults),
	}, nil
}

// AddPeers 添加节点
// 用例：用户想要向集群中添加新的节点
func (s *ConsistentHashApplicationService) AddPeers(ctx context.Context, cmd AddPeersCommand) error {
	// 验证输入
	if err := s.validateAddPeersCommand(cmd); err != nil {
		return fmt.Errorf("验证添加节点命令失败: %w", err)
	}

	// 转换为领域对象
	peers := make([]domainHash.Peer, len(cmd.Peers))
	for i, peerReq := range cmd.Peers {
		peer, err := domainHash.NewPeerInfo(peerReq.ID, peerReq.Address, peerReq.Weight)
		if err != nil {
			return fmt.Errorf("创建节点信息失败: %w", err)
		}
		peers[i] = peer
	}

	// 添加节点
	s.peerPicker.AddPeers(peers...)

	return nil
}

// RemovePeers 移除节点
// 用例：用户想要从集群中移除节点
func (s *ConsistentHashApplicationService) RemovePeers(ctx context.Context, cmd RemovePeersCommand) error {
	// 验证输入
	if err := s.validateRemovePeersCommand(cmd); err != nil {
		return fmt.Errorf("验证移除节点命令失败: %w", err)
	}

	// 获取要移除的节点
	allPeers := s.peerPicker.GetAllPeers()
	peersToRemove := make([]domainHash.Peer, 0)

	for _, peerID := range cmd.PeerIDs {
		for _, peer := range allPeers {
			if peer.ID() == peerID {
				peersToRemove = append(peersToRemove, peer)
				break
			}
		}
	}

	if len(peersToRemove) == 0 {
		return fmt.Errorf("没有找到要移除的节点")
	}

	// 移除节点
	s.peerPicker.RemovePeers(peersToRemove...)

	return nil
}

// GetAllPeers 获取所有节点
// 用例：用户想要查看集群中的所有节点
func (s *ConsistentHashApplicationService) GetAllPeers(ctx context.Context) ([]PeerResult, error) {
	peers := s.peerPicker.GetAllPeers()
	
	results := make([]PeerResult, len(peers))
	for i, peer := range peers {
		results[i] = s.buildPeerResult(peer)
	}

	return results, nil
}

// GetHashStats 获取哈希统计信息
// 用例：用户想要查看一致性哈希的统计信息和负载分布
func (s *ConsistentHashApplicationService) GetHashStats(ctx context.Context) (*HashStatsResult, error) {
	// 检查健康状态
	healthy, err := s.peerPicker.IsHealthy()
	if !healthy {
		return nil, fmt.Errorf("节点选择器不健康: %w", err)
	}

	// 获取统计信息（这里需要扩展PeerPicker接口或使用类型断言）
	// 为了简化，我们返回基本信息
	peers := s.peerPicker.GetAllPeers()
	
	return &HashStatsResult{
		TotalPeers:      len(peers),
		VirtualNodes:    0, // 需要从具体实现获取
		Replicas:        0, // 需要从具体实现获取
		KeyDistribution: make(map[string]int),
		LoadBalance:     0.0,
	}, nil
}

// CheckHealth 检查健康状态
// 用例：用户想要检查一致性哈希系统是否健康
func (s *ConsistentHashApplicationService) CheckHealth(ctx context.Context) (*HealthCheckResult, error) {
	healthy, err := s.peerPicker.IsHealthy()
	
	result := &HealthCheckResult{
		IsHealthy: healthy,
	}

	if err != nil {
		result.Message = err.Error()
	} else {
		result.Message = "系统健康"
	}

	return result, nil
}

// validatePeerSelectionCommand 验证节点选择命令
func (s *ConsistentHashApplicationService) validatePeerSelectionCommand(cmd PeerSelectionCommand) error {
	if cmd.Key == "" {
		return fmt.Errorf("键不能为空")
	}

	return nil
}

// validateMultiplePeerSelectionCommand 验证多节点选择命令
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

// validateAddPeersCommand 验证添加节点命令
func (s *ConsistentHashApplicationService) validateAddPeersCommand(cmd AddPeersCommand) error {
	if len(cmd.Peers) == 0 {
		return fmt.Errorf("节点列表不能为空")
	}

	for i, peer := range cmd.Peers {
		if peer.ID == "" {
			return fmt.Errorf("第%d个节点的ID不能为空", i+1)
		}
		if peer.Address == "" {
			return fmt.Errorf("第%d个节点的地址不能为空", i+1)
		}
		if peer.Weight < 0 {
			return fmt.Errorf("第%d个节点的权重不能为负数", i+1)
		}
	}

	return nil
}

// validateRemovePeersCommand 验证移除节点命令
func (s *ConsistentHashApplicationService) validateRemovePeersCommand(cmd RemovePeersCommand) error {
	if len(cmd.PeerIDs) == 0 {
		return fmt.Errorf("节点ID列表不能为空")
	}

	for i, peerID := range cmd.PeerIDs {
		if peerID == "" {
			return fmt.Errorf("第%d个节点ID不能为空", i+1)
		}
	}

	return nil
}

// buildPeerResult 构建节点结果
func (s *ConsistentHashApplicationService) buildPeerResult(peer domainHash.Peer) PeerResult {
	return PeerResult{
		ID:      peer.ID(),
		Address: peer.Address(),
		Weight:  peer.Weight(),
		IsAlive: peer.IsAlive(),
	}
}
