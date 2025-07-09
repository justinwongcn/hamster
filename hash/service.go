package hash

import (
	"context"
	"fmt"

	appHash "github.com/justinwongcn/hamster/internal/application/consistent_hash"
	domainHash "github.com/justinwongcn/hamster/internal/domain/consistent_hash"
	infraHash "github.com/justinwongcn/hamster/internal/infrastructure/consistent_hash"
)

// Service 一致性哈希服务公共接口
type Service struct {
	appService *appHash.ConsistentHashApplicationService
}

// Config 一致性哈希配置
type Config struct {
	// Replicas 虚拟节点数量
	Replicas int

	// HashFunction 哈希函数
	HashFunction func(data []byte) uint32

	// EnableSingleflight 是否启用单飞模式
	EnableSingleflight bool
}

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	return &Config{
		Replicas:           150,
		HashFunction:       nil, // 使用默认哈希函数
		EnableSingleflight: true,
	}
}

// Option 配置选项函数
type Option func(*Config)

// WithReplicas 设置虚拟节点数量
func WithReplicas(replicas int) Option {
	return func(c *Config) {
		c.Replicas = replicas
	}
}

// WithHashFunction 设置哈希函数
func WithHashFunction(fn func(data []byte) uint32) Option {
	return func(c *Config) {
		c.HashFunction = fn
	}
}

// WithSingleflight 设置是否启用单飞模式
func WithSingleflight(enable bool) Option {
	return func(c *Config) {
		c.EnableSingleflight = enable
	}
}

// NewService 创建一致性哈希服务
func NewService(options ...Option) (*Service, error) {
	config := DefaultConfig()
	for _, option := range options {
		option(config)
	}

	return NewServiceWithConfig(config)
}

// NewServiceWithConfig 使用配置创建一致性哈希服务
func NewServiceWithConfig(config *Config) (*Service, error) {
	if config == nil {
		return nil, fmt.Errorf("配置不能为空")
	}

	// 创建一致性哈希映射
	hashMap := infraHash.NewConsistentHashMap(config.Replicas, config.HashFunction)

	// 创建节点选择器
	var peerPicker domainHash.PeerPicker
	if config.EnableSingleflight {
		peerPicker = infraHash.NewSingleflightPeerPicker(hashMap)
	} else {
		// 暂时只支持 singleflight 模式
		peerPicker = infraHash.NewSingleflightPeerPicker(hashMap)
	}

	// 创建应用服务
	appService := appHash.NewConsistentHashApplicationService(peerPicker)

	return &Service{
		appService: appService,
	}, nil
}

// Peer 节点信息
type Peer struct {
	ID      string `json:"id"`
	Address string `json:"address"`
	Weight  int    `json:"weight"`
	IsAlive bool   `json:"is_alive"`
}

// AddPeer 添加单个节点
func (s *Service) AddPeer(ctx context.Context, peer Peer) error {
	cmd := appHash.AddPeersCommand{
		Peers: []appHash.PeerRequest{
			{
				ID:      peer.ID,
				Address: peer.Address,
				Weight:  peer.Weight,
			},
		},
	}

	return s.appService.AddPeers(ctx, cmd)
}

// AddPeers 添加多个节点
func (s *Service) AddPeers(ctx context.Context, peers []Peer) error {
	peerRequests := make([]appHash.PeerRequest, len(peers))
	for i, peer := range peers {
		peerRequests[i] = appHash.PeerRequest{
			ID:      peer.ID,
			Address: peer.Address,
			Weight:  peer.Weight,
		}
	}

	cmd := appHash.AddPeersCommand{Peers: peerRequests}
	return s.appService.AddPeers(ctx, cmd)
}

// RemovePeer 移除单个节点
func (s *Service) RemovePeer(ctx context.Context, peerID string) error {
	cmd := appHash.RemovePeersCommand{
		PeerIDs: []string{peerID},
	}

	return s.appService.RemovePeers(ctx, cmd)
}

// RemovePeers 移除多个节点
func (s *Service) RemovePeers(ctx context.Context, peerIDs []string) error {
	cmd := appHash.RemovePeersCommand{PeerIDs: peerIDs}
	return s.appService.RemovePeers(ctx, cmd)
}

// SelectPeer 根据键选择节点
func (s *Service) SelectPeer(ctx context.Context, key string) (*Peer, error) {
	cmd := appHash.PeerSelectionCommand{Key: key}

	result, err := s.appService.SelectPeer(ctx, cmd)
	if err != nil {
		return nil, err
	}

	return &Peer{
		ID:      result.Peer.ID,
		Address: result.Peer.Address,
		Weight:  result.Peer.Weight,
		IsAlive: result.Peer.IsAlive,
	}, nil
}

// SelectPeers 根据键选择多个节点
func (s *Service) SelectPeers(ctx context.Context, key string, count int) ([]Peer, error) {
	cmd := appHash.MultiplePeerSelectionCommand{
		Key:   key,
		Count: count,
	}

	result, err := s.appService.SelectMultiplePeers(ctx, cmd)
	if err != nil {
		return nil, err
	}

	peers := make([]Peer, len(result.Peers))
	for i, peer := range result.Peers {
		peers[i] = Peer{
			ID:      peer.ID,
			Address: peer.Address,
			Weight:  peer.Weight,
			IsAlive: peer.IsAlive,
		}
	}

	return peers, nil
}

// GetStats 获取哈希统计信息
func (s *Service) GetStats(ctx context.Context) (*Stats, error) {
	result, err := s.appService.GetHashStats(ctx)
	if err != nil {
		return nil, err
	}

	return &Stats{
		TotalPeers:      result.TotalPeers,
		VirtualNodes:    result.VirtualNodes,
		Replicas:        result.Replicas,
		KeyDistribution: result.KeyDistribution,
		LoadBalance:     result.LoadBalance,
	}, nil
}

// HealthCheck 健康检查
func (s *Service) HealthCheck(ctx context.Context) (*HealthStatus, error) {
	result, err := s.appService.CheckHealth(ctx)
	if err != nil {
		return nil, err
	}

	return &HealthStatus{
		IsHealthy: result.IsHealthy,
		Message:   result.Message,
	}, nil
}

// Stats 哈希统计信息
type Stats struct {
	TotalPeers      int            `json:"total_peers"`
	VirtualNodes    int            `json:"virtual_nodes"`
	Replicas        int            `json:"replicas"`
	KeyDistribution map[string]int `json:"key_distribution"`
	LoadBalance     float64        `json:"load_balance"`
}

// HealthStatus 健康状态
type HealthStatus struct {
	IsHealthy bool   `json:"is_healthy"`
	Message   string `json:"message"`
}
