package cache

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRandomPolicy_KeyAccessed(t *testing.T) {
	tests := []struct {
		name        string
		operations  []string
		wantSize    int
		checkKeys   []string
		wantHasKeys []bool
	}{
		{
			name:        "添加新key到随机策略",
			operations:  []string{"key1", "key2"},
			wantSize:    2,
			checkKeys:   []string{"key1", "key2"},
			wantHasKeys: []bool{true, true},
		},
		{
			name:        "重复访问key不改变大小",
			operations:  []string{"key1", "key2", "key1"},
			wantSize:    2,
			checkKeys:   []string{"key1", "key2"},
			wantHasKeys: []bool{true, true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			policy := NewRandomPolicy()

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
		})
	}
}

func TestRandomPolicy_Evict(t *testing.T) {
	tests := []struct {
		name         string
		setupKeys    []string
		wantEvicted  string
		wantSize     int
		checkEvicted bool
	}{
		{
			name:         "空策略返回空字符串",
			setupKeys:    []string{},
			wantEvicted:  "",
			wantSize:     0,
			checkEvicted: false,
		},
		{
			name:         "淘汰随机key",
			setupKeys:    []string{"key1", "key2", "key3"},
			wantSize:     2,
			checkEvicted: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			policy := NewRandomPolicy()

			for _, key := range tt.setupKeys {
				err := policy.KeyAccessed(ctx, key)
				require.NoError(t, err)
			}

			evicted, err := policy.Evict(ctx)
			require.NoError(t, err)

			if tt.wantEvicted != "" {
				assert.Equal(t, tt.wantEvicted, evicted)
			} else if tt.checkEvicted {
				assert.Contains(t, tt.setupKeys, evicted)
			}

			size, err := policy.Size(ctx)
			require.NoError(t, err)
			assert.Equal(t, tt.wantSize, size)

			if tt.checkEvicted && evicted != "" {
				has, err := policy.Has(ctx, evicted)
				require.NoError(t, err)
				assert.False(t, has)
			}
		})
	}
}

func TestRandomPolicy_Remove(t *testing.T) {
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
			setupKeys: []string{"key1", "key2"},
			removeKey: "key1",
			wantSize:  1,
			checkKey:  "key1",
			wantHas:   false,
		},
		{
			name:      "移除不存在的key",
			setupKeys: []string{"key1", "key2"},
			removeKey: "key3",
			wantSize:  2,
			checkKey:  "key1",
			wantHas:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			policy := NewRandomPolicy()

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

func TestRandomPolicy_Has(t *testing.T) {
	tests := []struct {
		name      string
		setupKeys []string
		checkKey  string
		wantHas   bool
	}{
		{
			name:      "检查存在的key",
			setupKeys: []string{"key1"},
			checkKey:  "key1",
			wantHas:   true,
		},
		{
			name:      "检查不存在的key",
			setupKeys: []string{"key1"},
			checkKey:  "key2",
			wantHas:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			policy := NewRandomPolicy()

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

func TestRandomPolicy_Size(t *testing.T) {
	tests := []struct {
		name       string
		operations []struct {
			action string
			key    string
		}
		wantSize int
	}{
		{
			name:       "初始大小为0",
			operations: []struct{ action, key string }{},
			wantSize:   0,
		},
		{
			name: "添加key后大小增加",
			operations: []struct{ action, key string }{
				{"access", "key1"},
			},
			wantSize: 1,
		},
		{
			name: "重复访问不增加大小",
			operations: []struct{ action, key string }{
				{"access", "key1"},
				{"access", "key1"},
			},
			wantSize: 1,
		},
		{
			name: "淘汰后大小减少",
			operations: []struct{ action, key string }{
				{"access", "key1"},
				{"evict", ""},
			},
			wantSize: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			policy := NewRandomPolicy()

			for _, op := range tt.operations {
				switch op.action {
				case "access":
					err := policy.KeyAccessed(ctx, op.key)
					require.NoError(t, err)
				case "evict":
					_, err := policy.Evict(ctx)
					require.NoError(t, err)
				}
			}

			size, err := policy.Size(ctx)
			require.NoError(t, err)
			assert.Equal(t, tt.wantSize, size)
		})
	}
}

func TestRandomPolicy_Clear(t *testing.T) {
	tests := []struct {
		name      string
		setupKeys []string
		checkKey  string
		wantSize  int
		wantHas   bool
	}{
		{
			name:      "清空策略",
			setupKeys: []string{"key1", "key2", "key3"},
			checkKey:  "key1",
			wantSize:  0,
			wantHas:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			policy := NewRandomPolicy()

			for _, key := range tt.setupKeys {
				err := policy.KeyAccessed(ctx, key)
				require.NoError(t, err)
			}

			// 验证有key存在
			size, err := policy.Size(ctx)
			require.NoError(t, err)
			assert.Equal(t, len(tt.setupKeys), size)

			// 清空
			err = policy.Clear(ctx)
			require.NoError(t, err)

			// 验证已清空
			size, err = policy.Size(ctx)
			require.NoError(t, err)
			assert.Equal(t, tt.wantSize, size)

			// 验证key不存在
			has, err := policy.Has(ctx, tt.checkKey)
			require.NoError(t, err)
			assert.Equal(t, tt.wantHas, has)
		})
	}
}

func TestRandomPolicy_Capacity(t *testing.T) {
	tests := []struct {
		name         string
		capacity     []int
		operations   []string
		wantSize     int
		checkEvicted bool
	}{
		{
			name:         "无限制容量",
			capacity:     []int{},
			operations:   []string{"key0", "key1", "key2", "key3", "key4", "key5", "key6", "key7", "key8", "key9"},
			wantSize:     10,
			checkEvicted: false,
		},
		{
			name:         "限制容量触发淘汰",
			capacity:     []int{3},
			operations:   []string{"key1", "key2", "key3", "key4"},
			wantSize:     3,
			checkEvicted: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			var policy *RandomPolicy

			if len(tt.capacity) > 0 {
				policy = NewRandomPolicy(tt.capacity[0])
			} else {
				policy = NewRandomPolicy()
			}

			for _, key := range tt.operations {
				err := policy.KeyAccessed(ctx, key)
				require.NoError(t, err)
			}

			size, err := policy.Size(ctx)
			require.NoError(t, err)
			assert.Equal(t, tt.wantSize, size)

			if tt.checkEvicted {
				// 验证总共只有3个key（随机策略可能淘汰任意key）
				keys := []string{"key1", "key2", "key3", "key4"}
				existingCount := 0
				for _, key := range keys {
					has, err := policy.Has(ctx, key)
					require.NoError(t, err)
					if has {
						existingCount++
					}
				}
				assert.Equal(t, tt.wantSize, existingCount)
			}
		})
	}
}
