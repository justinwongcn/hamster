package hash

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()
	
	assert.Equal(t, 150, config.Replicas)
	assert.Nil(t, config.HashFunction)
	assert.True(t, config.EnableSingleflight)
}

func TestWithReplicas(t *testing.T) {
	config := DefaultConfig()
	option := WithReplicas(200)
	option(config)
	
	assert.Equal(t, 200, config.Replicas)
}

func TestWithHashFunction(t *testing.T) {
	config := DefaultConfig()
	hashFunc := func(data []byte) uint32 { return 42 }
	option := WithHashFunction(hashFunc)
	option(config)
	
	assert.NotNil(t, config.HashFunction)
	assert.Equal(t, uint32(42), config.HashFunction([]byte("test")))
}

func TestWithSingleflight(t *testing.T) {
	config := DefaultConfig()
	option := WithSingleflight(false)
	option(config)
	
	assert.False(t, config.EnableSingleflight)
}

func TestNewService(t *testing.T) {
	tests := []struct {
		name    string
		options []Option
		wantErr bool
	}{
		{
			name:    "default config",
			options: nil,
			wantErr: false,
		},
		{
			name: "with custom options",
			options: []Option{
				WithReplicas(100),
				WithSingleflight(true),
			},
			wantErr: false,
		},
		{
			name: "with hash function",
			options: []Option{
				WithHashFunction(func(data []byte) uint32 { return 42 }),
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service, err := NewService(tt.options...)
			
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, service)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, service)
				assert.NotNil(t, service.appService)
			}
		})
	}
}

func TestNewServiceWithConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name:    "nil config",
			config:  nil,
			wantErr: true,
		},
		{
			name:    "valid config",
			config:  DefaultConfig(),
			wantErr: false,
		},
		{
			name: "custom config",
			config: &Config{
				Replicas:           100,
				HashFunction:       nil,
				EnableSingleflight: false,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service, err := NewServiceWithConfig(tt.config)
			
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, service)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, service)
				assert.NotNil(t, service.appService)
			}
		})
	}
}

func TestService_AddPeer(t *testing.T) {
	service, err := NewService()
	require.NoError(t, err)
	
	ctx := context.Background()
	peer := Peer{
		ID:      "server1",
		Address: "192.168.1.1:8080",
		Weight:  100,
	}

	err = service.AddPeer(ctx, peer)
	assert.NoError(t, err)
}

func TestService_AddPeers(t *testing.T) {
	service, err := NewService()
	require.NoError(t, err)
	
	ctx := context.Background()
	peers := []Peer{
		{ID: "server1", Address: "192.168.1.1:8080", Weight: 100},
		{ID: "server2", Address: "192.168.1.2:8080", Weight: 100},
		{ID: "server3", Address: "192.168.1.3:8080", Weight: 150},
	}

	err = service.AddPeers(ctx, peers)
	assert.NoError(t, err)
}

func TestService_RemovePeer(t *testing.T) {
	service, err := NewService()
	require.NoError(t, err)
	
	ctx := context.Background()
	
	// Add a peer first
	peer := Peer{ID: "server1", Address: "192.168.1.1:8080", Weight: 100}
	err = service.AddPeer(ctx, peer)
	require.NoError(t, err)

	// Remove the peer
	err = service.RemovePeer(ctx, "server1")
	assert.NoError(t, err)
}

func TestService_RemovePeers(t *testing.T) {
	service, err := NewService()
	require.NoError(t, err)
	
	ctx := context.Background()
	
	// Add peers first
	peers := []Peer{
		{ID: "server1", Address: "192.168.1.1:8080", Weight: 100},
		{ID: "server2", Address: "192.168.1.2:8080", Weight: 100},
	}
	err = service.AddPeers(ctx, peers)
	require.NoError(t, err)

	// Remove the peers
	err = service.RemovePeers(ctx, []string{"server1", "server2"})
	assert.NoError(t, err)
}

func TestService_SelectPeer(t *testing.T) {
	service, err := NewService()
	require.NoError(t, err)
	
	ctx := context.Background()
	
	// Add peers first
	peers := []Peer{
		{ID: "server1", Address: "192.168.1.1:8080", Weight: 100},
		{ID: "server2", Address: "192.168.1.2:8080", Weight: 100},
	}
	err = service.AddPeers(ctx, peers)
	require.NoError(t, err)

	// Select a peer
	peer, err := service.SelectPeer(ctx, "test_key")
	assert.NoError(t, err)
	assert.NotNil(t, peer)
	assert.NotEmpty(t, peer.ID)
	assert.NotEmpty(t, peer.Address)
}

func TestService_SelectPeerNoPeers(t *testing.T) {
	service, err := NewService()
	require.NoError(t, err)
	
	ctx := context.Background()

	// Try to select a peer without adding any
	_, err = service.SelectPeer(ctx, "test_key")
	assert.Error(t, err)
}

func TestService_SelectPeers(t *testing.T) {
	service, err := NewService()
	require.NoError(t, err)
	
	ctx := context.Background()
	
	// Add peers first
	peers := []Peer{
		{ID: "server1", Address: "192.168.1.1:8080", Weight: 100},
		{ID: "server2", Address: "192.168.1.2:8080", Weight: 100},
		{ID: "server3", Address: "192.168.1.3:8080", Weight: 100},
	}
	err = service.AddPeers(ctx, peers)
	require.NoError(t, err)

	// Select multiple peers
	selectedPeers, err := service.SelectPeers(ctx, "test_key", 2)
	assert.NoError(t, err)
	assert.Len(t, selectedPeers, 2)
	
	for _, peer := range selectedPeers {
		assert.NotEmpty(t, peer.ID)
		assert.NotEmpty(t, peer.Address)
	}
}

func TestService_SelectPeersMoreThanAvailable(t *testing.T) {
	service, err := NewService()
	require.NoError(t, err)
	
	ctx := context.Background()
	
	// Add only one peer
	peer := Peer{ID: "server1", Address: "192.168.1.1:8080", Weight: 100}
	err = service.AddPeer(ctx, peer)
	require.NoError(t, err)

	// Try to select more peers than available
	selectedPeers, err := service.SelectPeers(ctx, "test_key", 3)
	// Should not error, but return only available peers
	assert.NoError(t, err)
	assert.LessOrEqual(t, len(selectedPeers), 1)
}

func TestService_GetStats(t *testing.T) {
	service, err := NewService()
	require.NoError(t, err)
	
	ctx := context.Background()
	
	// Add some peers first
	peers := []Peer{
		{ID: "server1", Address: "192.168.1.1:8080", Weight: 100},
		{ID: "server2", Address: "192.168.1.2:8080", Weight: 100},
	}
	err = service.AddPeers(ctx, peers)
	require.NoError(t, err)

	stats, err := service.GetStats(ctx)
	assert.NoError(t, err)
	assert.NotNil(t, stats)
	assert.GreaterOrEqual(t, stats.TotalPeers, 0)
	assert.GreaterOrEqual(t, stats.VirtualNodes, 0)
	assert.GreaterOrEqual(t, stats.Replicas, 0)
}

func TestService_HealthCheck(t *testing.T) {
	service, err := NewService()
	require.NoError(t, err)
	
	ctx := context.Background()

	status, err := service.HealthCheck(ctx)
	assert.NoError(t, err)
	assert.NotNil(t, status)
	// Health status can be true or false, both are valid
}

func TestPeerStruct(t *testing.T) {
	peer := Peer{
		ID:      "test_server",
		Address: "192.168.1.100:8080",
		Weight:  200,
		IsAlive: true,
	}
	
	assert.Equal(t, "test_server", peer.ID)
	assert.Equal(t, "192.168.1.100:8080", peer.Address)
	assert.Equal(t, 200, peer.Weight)
	assert.True(t, peer.IsAlive)
}

func TestStatsStruct(t *testing.T) {
	stats := Stats{
		TotalPeers:      3,
		VirtualNodes:    450,
		Replicas:        150,
		KeyDistribution: map[string]int{"server1": 150, "server2": 150, "server3": 150},
		LoadBalance:     0.95,
	}
	
	assert.Equal(t, 3, stats.TotalPeers)
	assert.Equal(t, 450, stats.VirtualNodes)
	assert.Equal(t, 150, stats.Replicas)
	assert.Equal(t, 0.95, stats.LoadBalance)
	assert.Len(t, stats.KeyDistribution, 3)
}

func TestHealthStatusStruct(t *testing.T) {
	status := HealthStatus{
		IsHealthy: true,
		Message:   "All systems operational",
	}
	
	assert.True(t, status.IsHealthy)
	assert.Equal(t, "All systems operational", status.Message)
}
