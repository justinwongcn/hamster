package cache

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewLRUPolicy(t *testing.T) {
	tests := []struct {
		name     string
		wantSize int
	}{
		{
			name:     "创建空的LRU策略",
			wantSize: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			policy := NewLRUPolicy()
			assert.NotNil(t, policy)

			size, err := policy.Size(context.Background())
			require.NoError(t, err)
			assert.Equal(t, tt.wantSize, size)
		})
	}
}

func TestLRUPolicy_KeyAccessed(t *testing.T) {
	tests := []struct {
		name        string
		operations  []string
		wantSize    int
		checkKeys   []string
		wantHasKeys []bool
		testEvict   bool
		wantEvicted string
	}{
		{
			name:        "添加新key到LRU策略",
			operations:  []string{"key1"},
			wantSize:    1,
			checkKeys:   []string{"key1"},
			wantHasKeys: []bool{true},
		},
		{
			name:        "处理重复key访问",
			operations:  []string{"key1", "key2", "key1"},
			wantSize:    2,
			checkKeys:   []string{"key1", "key2"},
			wantHasKeys: []bool{true, true},
			testEvict:   true,
			wantEvicted: "key2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			policy := NewLRUPolicy()

			for _, key := range tt.operations {
				err := policy.KeyAccessed(ctx, key)
				require.NoError(t, err)
			}

			size, err := policy.Size(ctx)
			require.NoError(t, err)
			assert.Equal(t, tt.wantSize, size)

			for i, key := range tt.checkKeys {
				has, err := policy.Has(ctx, key)
				require.NoError(t, err)
				assert.Equal(t, tt.wantHasKeys[i], has)
			}

			if tt.testEvict {
				evicted, err := policy.Evict(ctx)
				require.NoError(t, err)
				assert.Equal(t, tt.wantEvicted, evicted)
			}
		})
	}
}

func TestLRUPolicy_Evict(t *testing.T) {
	tests := []struct {
		name        string
		setupKeys   []string
		wantEvicted string
		wantSize    int
		checkKey    string
		wantHas     bool
	}{
		{
			name:        "空策略返回空字符串",
			setupKeys:   []string{},
			wantEvicted: "",
			wantSize:    0,
			checkKey:    "",
			wantHas:     false,
		},
		{
			name:        "淘汰最近最少使用的key",
			setupKeys:   []string{"key1", "key2"},
			wantEvicted: "key1",
			wantSize:    1,
			checkKey:    "key1",
			wantHas:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			policy := NewLRUPolicy()

			for _, key := range tt.setupKeys {
				err := policy.KeyAccessed(ctx, key)
				require.NoError(t, err)
			}

			evicted, err := policy.Evict(ctx)
			require.NoError(t, err)
			assert.Equal(t, tt.wantEvicted, evicted)

			size, err := policy.Size(ctx)
			require.NoError(t, err)
			assert.Equal(t, tt.wantSize, size)

			if tt.checkKey != "" {
				has, err := policy.Has(ctx, tt.checkKey)
				require.NoError(t, err)
				assert.Equal(t, tt.wantHas, has)
			}
		})
	}
}

// TestLRUPolicy_EvictOrder 测试LRU淘汰顺序
func TestLRUPolicy_EvictOrder(t *testing.T) {
	tests := []struct {
		name        string
		setupKeys   []string
		evictCount  int
		wantEvicted []string
	}{
		{
			name:        "维持正确的淘汰顺序",
			setupKeys:   []string{"key1", "key2", "key3"},
			evictCount:  2,
			wantEvicted: []string{"key1", "key2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			policy := NewLRUPolicy()

			for _, key := range tt.setupKeys {
				err := policy.KeyAccessed(ctx, key)
				require.NoError(t, err)
			}

			var evicted []string
			for i := 0; i < tt.evictCount; i++ {
				key, err := policy.Evict(ctx)
				require.NoError(t, err)
				evicted = append(evicted, key)
			}

			assert.Equal(t, tt.wantEvicted, evicted)
		})
	}
}

func TestLRUPolicy_Remove(t *testing.T) {
	tests := []struct {
		name      string
		setupKeys []string
		removeKey string
		wantSize  int
		checkKey  string
		wantHas   bool
	}{
		{
			name:      "移除存在的key",
			setupKeys: []string{"key1"},
			removeKey: "key1",
			wantSize:  0,
			checkKey:  "key1",
			wantHas:   false,
		},
		{
			name:      "移除不存在的key不影响其他key",
			setupKeys: []string{"key1"},
			removeKey: "key2",
			wantSize:  1,
			checkKey:  "key1",
			wantHas:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			policy := NewLRUPolicy()

			for _, key := range tt.setupKeys {
				err := policy.KeyAccessed(ctx, key)
				require.NoError(t, err)
			}

			err := policy.Remove(ctx, tt.removeKey)
			require.NoError(t, err)

			size, err := policy.Size(ctx)
			require.NoError(t, err)
			assert.Equal(t, tt.wantSize, size)

			has, err := policy.Has(ctx, tt.checkKey)
			require.NoError(t, err)
			assert.Equal(t, tt.wantHas, has)
		})
	}
}

func TestLRUPolicy_Has(t *testing.T) {
	tests := []struct {
		name      string
		setupKeys []string
		checkKey  string
		wantHas   bool
	}{
		{
			name:      "检查存在的key返回true",
			setupKeys: []string{"key1"},
			checkKey:  "key1",
			wantHas:   true,
		},
		{
			name:      "检查不存在的key返回false",
			setupKeys: []string{},
			checkKey:  "key1",
			wantHas:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			policy := NewLRUPolicy()

			for _, key := range tt.setupKeys {
				err := policy.KeyAccessed(ctx, key)
				require.NoError(t, err)
			}

			has, err := policy.Has(ctx, tt.checkKey)
			require.NoError(t, err)
			assert.Equal(t, tt.wantHas, has)
		})
	}
}

func TestLRUPolicy_CombinedOperations(t *testing.T) {
	tests := []struct {
		name       string
		operations []struct {
			action string
			key    string
		}
		wantEvicted string
		wantSize    int
		checkKeys   map[string]bool
	}{
		{
			name: "组合操作测试LRU顺序",
			operations: []struct {
				action string
				key    string
			}{
				{"access", "key1"},
				{"access", "key2"},
				{"access", "key3"},
				{"access", "key1"}, // 移动key1到最新位置
				{"evict", ""},      // 应该淘汰key2
				{"remove", "key3"}, // 删除key3
			},
			wantEvicted: "key2",
			wantSize:    1,
			checkKeys: map[string]bool{
				"key1": true,
				"key2": false,
				"key3": false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			policy := NewLRUPolicy()
			var evicted string

			for _, op := range tt.operations {
				switch op.action {
				case "access":
					err := policy.KeyAccessed(ctx, op.key)
					require.NoError(t, err)
				case "evict":
					var err error
					evicted, err = policy.Evict(ctx)
					require.NoError(t, err)
				case "remove":
					err := policy.Remove(ctx, op.key)
					require.NoError(t, err)
				}
			}

			if tt.wantEvicted != "" {
				assert.Equal(t, tt.wantEvicted, evicted)
			}

			size, err := policy.Size(ctx)
			require.NoError(t, err)
			assert.Equal(t, tt.wantSize, size)

			for key, expected := range tt.checkKeys {
				has, err := policy.Has(ctx, key)
				require.NoError(t, err)
				assert.Equal(t, expected, has)
			}
		})
	}
}

func TestLRUPolicy_Size(t *testing.T) {
	ctx := context.Background()
	policy := NewLRUPolicy()

	// 初始大小应该为0
	size, err := policy.Size(ctx)
	require.NoError(t, err)
	assert.Equal(t, 0, size)

	// 添加key后大小增加
	err = policy.KeyAccessed(ctx, "key1")
	require.NoError(t, err)
	size, err = policy.Size(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, size)

	// 重复访问不增加大小
	err = policy.KeyAccessed(ctx, "key1")
	require.NoError(t, err)
	size, err = policy.Size(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, size)

	// 淘汰后大小减少
	_, err = policy.Evict(ctx)
	require.NoError(t, err)
	size, err = policy.Size(ctx)
	require.NoError(t, err)
	assert.Equal(t, 0, size)
}

func TestLRUPolicy_Clear(t *testing.T) {
	ctx := context.Background()
	policy := NewLRUPolicy()

	// 添加一些key
	err := policy.KeyAccessed(ctx, "key1")
	require.NoError(t, err)
	err = policy.KeyAccessed(ctx, "key2")
	require.NoError(t, err)
	err = policy.KeyAccessed(ctx, "key3")
	require.NoError(t, err)

	// 验证有key存在
	size, err := policy.Size(ctx)
	require.NoError(t, err)
	assert.Equal(t, 3, size)

	// 清空
	err = policy.Clear(ctx)
	require.NoError(t, err)

	// 验证已清空
	size, err = policy.Size(ctx)
	require.NoError(t, err)
	assert.Equal(t, 0, size)

	// 验证key不存在
	has, err := policy.Has(ctx, "key1")
	require.NoError(t, err)
	assert.False(t, has)
}

func TestLRUPolicy_Capacity(t *testing.T) {
	ctx := context.Background()

	t.Run("unlimited capacity", func(t *testing.T) {
		policy := NewLRUPolicy() // 无限制容量

		// 添加多个key
		for i := range 100 {
			err := policy.KeyAccessed(ctx, fmt.Sprintf("key%d", i))
			require.NoError(t, err)
		}

		size, err := policy.Size(ctx)
		require.NoError(t, err)
		assert.Equal(t, 100, size)
	})

	t.Run("limited capacity", func(t *testing.T) {
		policy := NewLRUPolicy(3) // 限制容量为3

		// 添加超过容量的key
		err := policy.KeyAccessed(ctx, "key1")
		require.NoError(t, err)
		err = policy.KeyAccessed(ctx, "key2")
		require.NoError(t, err)
		err = policy.KeyAccessed(ctx, "key3")
		require.NoError(t, err)
		err = policy.KeyAccessed(ctx, "key4") // 应该触发自动淘汰
		require.NoError(t, err)

		size, err := policy.Size(ctx)
		require.NoError(t, err)
		assert.Equal(t, 3, size) // 大小应该保持在3

		// key1应该被淘汰了
		has, err := policy.Has(ctx, "key1")
		require.NoError(t, err)
		assert.False(t, has)

		// 其他key应该还在
		has, err = policy.Has(ctx, "key4")
		require.NoError(t, err)
		assert.True(t, has)
	})
}
