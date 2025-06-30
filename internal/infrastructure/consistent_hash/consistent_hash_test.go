package consistent_hash

import (
	"fmt"
	"hash/crc32"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	domainHash "github.com/justinwongcn/hamster/internal/domain/consistent_hash"
)

// TestHashValueObjects 测试哈希相关的值对象
func TestHashValueObjects(t *testing.T) {
	t.Run("HashKey", func(t *testing.T) {
		tests := []struct {
			name    string
			keyStr  string
			wantErr bool
		}{
			{
				name:    "正常键",
				keyStr:  "test_key",
				wantErr: false,
			},
			{
				name:    "空键",
				keyStr:  "",
				wantErr: true,
			},
			{
				name:    "长键",
				keyStr:  string(make([]byte, 501)), // 超过500字符
				wantErr: true,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				key, err := domainHash.NewHashKey(tt.keyStr)
				if tt.wantErr {
					assert.Error(t, err)
					return
				}
				require.NoError(t, err)
				assert.Equal(t, tt.keyStr, key.String())
				assert.Equal(t, []byte(tt.keyStr), key.Bytes())
			})
		}
	})

	t.Run("PeerInfo", func(t *testing.T) {
		tests := []struct {
			name    string
			id      string
			address string
			weight  int
			wantErr bool
		}{
			{
				name:    "正常节点",
				id:      "peer1",
				address: "192.168.1.1:8080",
				weight:  100,
				wantErr: false,
			},
			{
				name:    "空ID",
				id:      "",
				address: "192.168.1.1:8080",
				weight:  100,
				wantErr: true,
			},
			{
				name:    "空地址",
				id:      "peer1",
				address: "",
				weight:  100,
				wantErr: true,
			},
			{
				name:    "负权重",
				id:      "peer1",
				address: "192.168.1.1:8080",
				weight:  -1,
				wantErr: true,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				peer, err := domainHash.NewPeerInfo(tt.id, tt.address, tt.weight)
				if tt.wantErr {
					assert.Error(t, err)
					return
				}
				require.NoError(t, err)
				assert.Equal(t, tt.id, peer.ID())
				assert.Equal(t, tt.address, peer.Address())
				assert.Equal(t, tt.weight, peer.Weight())
				assert.True(t, peer.IsAlive())
			})
		}
	})

	t.Run("VirtualNodeConfig", func(t *testing.T) {
		tests := []struct {
			name     string
			replicas int
			wantErr  bool
		}{
			{
				name:     "正常配置",
				replicas: 150,
				wantErr:  false,
			},
			{
				name:     "零倍数",
				replicas: 0,
				wantErr:  true,
			},
			{
				name:     "负倍数",
				replicas: -1,
				wantErr:  true,
			},
			{
				name:     "过大倍数",
				replicas: 1001,
				wantErr:  true,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				config, err := domainHash.NewVirtualNodeConfig(tt.replicas, nil)
				if tt.wantErr {
					assert.Error(t, err)
					return
				}
				require.NoError(t, err)
				assert.Equal(t, tt.replicas, config.Replicas())
				assert.NotNil(t, config.HashFunc())
			})
		}
	})
}

// TestConsistentHashMap 测试一致性哈希映射基本功能
func TestConsistentHashMap(t *testing.T) {
	tests := []struct {
		name     string
		replicas int
		peers    []string
		testKeys []string
	}{
		{
			name:     "基本功能测试",
			replicas: 3,
			peers:    []string{"peer1", "peer2", "peer3"},
			testKeys: []string{"key1", "key2", "key3", "key4", "key5"},
		},
		{
			name:     "单节点测试",
			replicas: 5,
			peers:    []string{"single_peer"},
			testKeys: []string{"test1", "test2", "test3"},
		},
		{
			name:     "大量节点测试",
			replicas: 10,
			peers:    []string{"peer1", "peer2", "peer3", "peer4", "peer5", "peer6", "peer7", "peer8", "peer9", "peer10"},
			testKeys: []string{"key1", "key2", "key3", "key4", "key5"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hashMap := NewConsistentHashMap(tt.replicas, nil)

			// 添加节点
			hashMap.Add(tt.peers...)

			// 验证节点数量
			peers := hashMap.Peers()
			assert.Len(t, peers, len(tt.peers))

			// 验证每个键都能找到对应的节点
			for _, key := range tt.testKeys {
				peer, err := hashMap.Get(key)
				assert.NoError(t, err)
				assert.Contains(t, tt.peers, peer)
			}

			// 验证一致性：相同的键总是返回相同的节点
			for _, key := range tt.testKeys {
				peer1, err1 := hashMap.Get(key)
				peer2, err2 := hashMap.Get(key)
				assert.NoError(t, err1)
				assert.NoError(t, err2)
				assert.Equal(t, peer1, peer2)
			}
		})
	}
}

// TestConsistentHashMap_EmptyMap 测试空映射
func TestConsistentHashMap_EmptyMap(t *testing.T) {
	hashMap := NewConsistentHashMap(3, nil)

	// 空映射应该返回错误
	_, err := hashMap.Get("any_key")
	assert.ErrorIs(t, err, domainHash.ErrNoPeers)

	assert.True(t, hashMap.IsEmpty())
	assert.Empty(t, hashMap.Peers())
}

// TestConsistentHashMap_AddRemove 测试添加和移除节点
func TestConsistentHashMap_AddRemove(t *testing.T) {
	hashMap := NewConsistentHashMap(3, nil)

	// 添加节点
	hashMap.Add("peer1", "peer2", "peer3")
	assert.Len(t, hashMap.Peers(), 3)
	assert.False(t, hashMap.IsEmpty())

	// 移除节点
	hashMap.Remove("peer2")
	peers := hashMap.Peers()
	assert.Len(t, peers, 2)
	assert.Contains(t, peers, "peer1")
	assert.Contains(t, peers, "peer3")
	assert.NotContains(t, peers, "peer2")

	// 移除所有节点
	hashMap.Remove("peer1", "peer3")
	assert.True(t, hashMap.IsEmpty())
	assert.Empty(t, hashMap.Peers())
}

// TestConsistentHashMap_Consistency 测试一致性
func TestConsistentHashMap_Consistency(t *testing.T) {
	hashMap := NewConsistentHashMap(50, nil)

	// 添加初始节点
	initialPeers := []string{"peer1", "peer2", "peer3"}
	hashMap.Add(initialPeers...)

	// 记录一些键的初始映射
	testKeys := []string{"key1", "key2", "key3", "key4", "key5", "key6", "key7", "key8", "key9", "key10"}
	initialMapping := make(map[string]string)

	for _, key := range testKeys {
		peer, err := hashMap.Get(key)
		require.NoError(t, err)
		initialMapping[key] = peer
	}

	// 添加新节点
	hashMap.Add("peer4")

	// 检查有多少键的映射发生了变化
	changedCount := 0
	for _, key := range testKeys {
		peer, err := hashMap.Get(key)
		require.NoError(t, err)
		if peer != initialMapping[key] {
			changedCount++
		}
	}

	// 一致性哈希的特点：添加节点时，只有少部分键的映射会发生变化
	// 这里我们验证变化的键数量应该少于总数的一半
	assert.Less(t, changedCount, len(testKeys)/2, "添加节点后，变化的键数量应该较少")
}

// TestConsistentHashMap_GetMultiple 测试获取多个节点
func TestConsistentHashMap_GetMultiple(t *testing.T) {
	hashMap := NewConsistentHashMap(10, nil)
	hashMap.Add("peer1", "peer2", "peer3", "peer4", "peer5")

	tests := []struct {
		name  string
		key   string
		count int
		want  int
	}{
		{
			name:  "获取3个节点",
			key:   "test_key",
			count: 3,
			want:  3,
		},
		{
			name:  "获取所有节点",
			key:   "test_key2",
			count: 5,
			want:  5,
		},
		{
			name:  "请求超过可用节点数",
			key:   "test_key3",
			count: 10,
			want:  5, // 最多只能返回5个不同的节点
		},
		{
			name:  "请求0个节点",
			key:   "test_key4",
			count: 0,
			want:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			peers, err := hashMap.GetMultiple(tt.key, tt.count)
			assert.NoError(t, err)
			assert.Len(t, peers, tt.want)

			// 验证返回的节点都是不同的
			seen := make(map[string]bool)
			for _, peer := range peers {
				assert.False(t, seen[peer], "返回的节点应该都是不同的")
				seen[peer] = true
			}
		})
	}
}

// TestConsistentHashMap_CustomHashFunction 测试自定义哈希函数
func TestConsistentHashMap_CustomHashFunction(t *testing.T) {
	// 自定义哈希函数（简单的求和）
	customHash := func(data []byte) uint32 {
		sum := uint32(0)
		for _, b := range data {
			sum += uint32(b)
		}
		return sum
	}

	hashMap1 := NewConsistentHashMap(3, customHash)
	hashMap2 := NewConsistentHashMap(3, crc32.ChecksumIEEE)

	peers := []string{"peer1", "peer2", "peer3"}
	hashMap1.Add(peers...)
	hashMap2.Add(peers...)

	// 使用不同哈希函数的映射应该可能不同
	testKey := "test_key"
	peer1, err1 := hashMap1.Get(testKey)
	peer2, err2 := hashMap2.Get(testKey)

	assert.NoError(t, err1)
	assert.NoError(t, err2)

	// 两个映射都应该返回有效的节点
	assert.Contains(t, peers, peer1)
	assert.Contains(t, peers, peer2)
}

// TestConsistentHashMap_Stats 测试统计信息
func TestConsistentHashMap_Stats(t *testing.T) {
	replicas := 5
	hashMap := NewConsistentHashMap(replicas, nil)
	hashMap.Add("peer1", "peer2", "peer3")

	stats := hashMap.Stats()

	assert.Equal(t, 3, stats.TotalPeers())
	assert.Equal(t, 3*replicas, stats.VirtualNodes())
	assert.Equal(t, replicas, stats.Replicas())

	distribution := stats.KeyDistribution()
	assert.Len(t, distribution, 3)

	// 每个节点应该有相同数量的虚拟节点
	for peer, count := range distribution {
		assert.Equal(t, replicas, count, "节点 %s 的虚拟节点数量应该等于replicas", peer)
	}
}

// TestVirtualNodes 测试虚拟节点机制
func TestVirtualNodes(t *testing.T) {
	t.Run("虚拟节点数量验证", func(t *testing.T) {
		replicas := 10
		hashMap := NewConsistentHashMap(replicas, nil)

		peers := []string{"peer1", "peer2", "peer3"}
		hashMap.Add(peers...)

		// 验证总虚拟节点数量
		totalVirtualNodes := len(hashMap.GetKeys())
		expectedTotal := len(peers) * replicas
		assert.Equal(t, expectedTotal, totalVirtualNodes)

		// 验证每个节点的虚拟节点数量
		for _, peer := range peers {
			count := hashMap.GetVirtualNodeCount(peer)
			assert.Equal(t, replicas, count, "节点 %s 应该有 %d 个虚拟节点", peer, replicas)
		}
	})

	t.Run("虚拟节点解决key倾斜问题", func(t *testing.T) {
		// 测试不同虚拟节点倍数对负载均衡的影响
		testCases := []struct {
			name     string
			replicas int
		}{
			{"低虚拟节点倍数", 3},
			{"中等虚拟节点倍数", 50},
			{"高虚拟节点倍数", 150},
		}

		peers := []string{"peer1", "peer2", "peer3"}
		// 生成大量测试键
		testKeys := make([]string, 1000)
		for i := 0; i < 1000; i++ {
			testKeys[i] = fmt.Sprintf("key_%d", i)
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				hashMap := NewConsistentHashMap(tc.replicas, nil)
				hashMap.Add(peers...)

				// 统计负载分布
				distribution := hashMap.GetLoadDistribution(testKeys)

				// 计算负载均衡度（标准差）
				total := 0
				for _, count := range distribution {
					total += count
				}
				avg := float64(total) / float64(len(distribution))

				variance := 0.0
				for _, count := range distribution {
					diff := float64(count) - avg
					variance += diff * diff
				}
				variance /= float64(len(distribution))
				stdDev := variance

				t.Logf("虚拟节点倍数: %d, 负载分布: %v, 标准差: %.2f", tc.replicas, distribution, stdDev)

				// 虚拟节点倍数越高，负载应该越均衡（标准差越小）
				// 这里我们验证每个节点至少分配到一些键
				for peer, count := range distribution {
					assert.Greater(t, count, 0, "节点 %s 应该分配到一些键", peer)
				}
			})
		}
	})

	t.Run("虚拟节点键生成", func(t *testing.T) {
		hashMap := NewConsistentHashMap(3, nil)
		hashMap.Add("test_peer")

		hashMapData := hashMap.GetHashMap()

		// 验证虚拟节点键的格式
		virtualNodeCount := 0
		for _, peer := range hashMapData {
			if peer == "test_peer" {
				virtualNodeCount++
			}
		}

		assert.Equal(t, 3, virtualNodeCount, "应该有3个虚拟节点")
	})
}

// TestSingleflightPeerPicker 测试singleflight节点选择器
func TestSingleflightPeerPicker(t *testing.T) {
	t.Run("基本功能测试", func(t *testing.T) {
		hashMap := NewConsistentHashMap(50, nil)
		picker := NewSingleflightPeerPicker(hashMap)

		// 创建测试节点
		peer1, _ := domainHash.NewPeerInfo("peer1", "192.168.1.1:8080", 100)
		peer2, _ := domainHash.NewPeerInfo("peer2", "192.168.1.2:8080", 100)
		peer3, _ := domainHash.NewPeerInfo("peer3", "192.168.1.3:8080", 100)

		// 添加节点
		picker.AddPeers(peer1, peer2, peer3)

		// 验证节点数量
		assert.Equal(t, 3, picker.GetPeerCount())
		assert.Equal(t, 3, picker.GetAlivePeerCount())

		// 测试节点选择
		selectedPeer, err := picker.PickPeer("test_key")
		assert.NoError(t, err)
		assert.NotNil(t, selectedPeer)
		assert.Contains(t, []string{"peer1", "peer2", "peer3"}, selectedPeer.ID())

		// 测试一致性：相同键应该返回相同节点
		selectedPeer2, err := picker.PickPeer("test_key")
		assert.NoError(t, err)
		assert.Equal(t, selectedPeer.ID(), selectedPeer2.ID())
	})

	t.Run("多节点选择测试", func(t *testing.T) {
		hashMap := NewConsistentHashMap(50, nil)
		picker := NewSingleflightPeerPicker(hashMap)

		// 创建测试节点
		peer1, _ := domainHash.NewPeerInfo("peer1", "192.168.1.1:8080", 100)
		peer2, _ := domainHash.NewPeerInfo("peer2", "192.168.1.2:8080", 100)
		peer3, _ := domainHash.NewPeerInfo("peer3", "192.168.1.3:8080", 100)

		picker.AddPeers(peer1, peer2, peer3)

		// 测试选择多个节点
		peers, err := picker.PickPeers("test_key", 2)
		assert.NoError(t, err)
		assert.Len(t, peers, 2)

		// 验证返回的节点都是不同的
		assert.NotEqual(t, peers[0].ID(), peers[1].ID())
	})

	t.Run("节点状态管理测试", func(t *testing.T) {
		hashMap := NewConsistentHashMap(50, nil)
		picker := NewSingleflightPeerPicker(hashMap)

		// 创建测试节点
		peer1, _ := domainHash.NewPeerInfo("peer1", "192.168.1.1:8080", 100)
		peer2, _ := domainHash.NewPeerInfo("peer2", "192.168.1.2:8080", 100)

		picker.AddPeers(peer1, peer2)

		// 验证健康状态
		healthy, err := picker.IsHealthy()
		assert.True(t, healthy)
		assert.NoError(t, err)

		// 更新节点状态
		err = picker.UpdatePeerStatus("peer1", false)
		assert.NoError(t, err)

		// 验证存活节点数量
		assert.Equal(t, 1, picker.GetAlivePeerCount())

		// 仍然应该是健康的，因为还有一个存活节点
		healthy, err = picker.IsHealthy()
		assert.True(t, healthy)
		assert.NoError(t, err)
	})

	t.Run("空节点选择器测试", func(t *testing.T) {
		hashMap := NewConsistentHashMap(50, nil)
		picker := NewSingleflightPeerPicker(hashMap)

		// 空选择器应该不健康
		healthy, err := picker.IsHealthy()
		assert.False(t, healthy)
		assert.ErrorIs(t, err, domainHash.ErrNoPeers)

		// 选择节点应该失败
		_, err = picker.PickPeer("test_key")
		assert.Error(t, err)
	})

	t.Run("节点移除测试", func(t *testing.T) {
		hashMap := NewConsistentHashMap(50, nil)
		picker := NewSingleflightPeerPicker(hashMap)

		// 创建测试节点
		peer1, _ := domainHash.NewPeerInfo("peer1", "192.168.1.1:8080", 100)
		peer2, _ := domainHash.NewPeerInfo("peer2", "192.168.1.2:8080", 100)

		picker.AddPeers(peer1, peer2)
		assert.Equal(t, 2, picker.GetPeerCount())

		// 移除节点
		picker.RemovePeers(peer1)
		assert.Equal(t, 1, picker.GetPeerCount())

		// 验证被移除的节点不存在
		_, exists := picker.GetPeerByID("peer1")
		assert.False(t, exists)

		// 验证剩余节点仍然存在
		_, exists = picker.GetPeerByID("peer2")
		assert.True(t, exists)
	})
}
