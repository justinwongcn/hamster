package cache

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFIFOPolicy_KeyAccessed(t *testing.T) {
	tests := []struct {
		name        string
		operations  []string
		wantSize    int
		checkKeys   []string
		wantHasKeys []bool
	}{
		{
			name:        "添加新key到FIFO策略",
			operations:  []string{"key1", "key2"},
			wantSize:    2,
			checkKeys:   []string{"key1", "key2"},
			wantHasKeys: []bool{true, true},
		},
		{
			name:        "重复访问key不改变位置",
			operations:  []string{"key1", "key2", "key1"},
			wantSize:    2,
			checkKeys:   []string{"key1", "key2"},
			wantHasKeys: []bool{true, true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			fifo := NewFIFOPolicy()

			for _, key := range tt.operations {
				err := fifo.KeyAccessed(ctx, key)
				require.NoError(t, err)
			}

			size, err := fifo.Size(ctx)
			require.NoError(t, err)
			assert.Equal(t, tt.wantSize, size)

			for i, key := range tt.checkKeys {
				has, err := fifo.Has(ctx, key)
				require.NoError(t, err)
				assert.Equal(t, tt.wantHasKeys[i], has)
			}
		})
	}
}

func TestFIFOPolicy_Evict(t *testing.T) {
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
			name:        "淘汰最早添加的key",
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
			fifo := NewFIFOPolicy()

			for _, key := range tt.setupKeys {
				err := fifo.KeyAccessed(ctx, key)
				require.NoError(t, err)
			}

			evicted, err := fifo.Evict(ctx)
			require.NoError(t, err)
			assert.Equal(t, tt.wantEvicted, evicted)

			size, err := fifo.Size(ctx)
			require.NoError(t, err)
			assert.Equal(t, tt.wantSize, size)

			if tt.checkKey != "" {
				has, err := fifo.Has(ctx, tt.checkKey)
				require.NoError(t, err)
				assert.Equal(t, tt.wantHas, has)
			}
		})
	}
}

func TestFIFOPolicy_Remove(t *testing.T) {
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
			fifo := NewFIFOPolicy()

			for _, key := range tt.setupKeys {
				err := fifo.KeyAccessed(ctx, key)
				require.NoError(t, err)
			}

			err := fifo.Remove(ctx, tt.removeKey)
			require.NoError(t, err)

			size, err := fifo.Size(ctx)
			require.NoError(t, err)
			assert.Equal(t, tt.wantSize, size)

			has, err := fifo.Has(ctx, tt.checkKey)
			require.NoError(t, err)
			assert.Equal(t, tt.wantHas, has)
		})
	}
}

func TestFIFOPolicy_Has(t *testing.T) {
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
			fifo := NewFIFOPolicy()

			for _, key := range tt.setupKeys {
				err := fifo.KeyAccessed(ctx, key)
				require.NoError(t, err)
			}

			has, err := fifo.Has(ctx, tt.checkKey)
			require.NoError(t, err)
			assert.Equal(t, tt.wantHas, has)
		})
	}
}

// TestFIFOPolicy_Size 测试Size方法
func TestFIFOPolicy_Size(t *testing.T) {
	tests := []struct {
		name      string
		setupKeys []string
		wantSize  int
	}{
		{
			name:      "初始大小为0",
			setupKeys: []string{},
			wantSize:  0,
		},
		{
			name:      "添加key后大小增加",
			setupKeys: []string{"key1", "key2"},
			wantSize:  2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			policy := NewFIFOPolicy()

			for _, key := range tt.setupKeys {
				err := policy.KeyAccessed(ctx, key)
				require.NoError(t, err)
			}

			size, err := policy.Size(ctx)
			require.NoError(t, err)
			assert.Equal(t, tt.wantSize, size)
		})
	}
}

// TestFIFOPolicy_Clear 测试Clear方法
func TestFIFOPolicy_Clear(t *testing.T) {
	tests := []struct {
		name      string
		setupKeys []string
		checkKey  string
		wantSize  int
		wantHas   bool
	}{
		{
			name:      "清空策略",
			setupKeys: []string{"key1", "key2"},
			checkKey:  "key1",
			wantSize:  0,
			wantHas:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			policy := NewFIFOPolicy()

			for _, key := range tt.setupKeys {
				err := policy.KeyAccessed(ctx, key)
				require.NoError(t, err)
			}

			err := policy.Clear(ctx)
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

// TestFIFOPolicy_Capacity 测试容量限制功能
func TestFIFOPolicy_Capacity(t *testing.T) {
	tests := []struct {
		name         string
		capacity     []int
		operations   []string
		wantSize     int
		evictedKey   string
		remainingKey string
	}{
		{
			name:         "无限制容量",
			capacity:     []int{},
			operations:   []string{"key0", "key1", "key2", "key3", "key4", "key5", "key6", "key7", "key8", "key9"},
			wantSize:     10,
			evictedKey:   "",
			remainingKey: "",
		},
		{
			name:         "限制容量触发淘汰",
			capacity:     []int{3},
			operations:   []string{"key1", "key2", "key3", "key4"},
			wantSize:     3,
			evictedKey:   "key1",
			remainingKey: "key4",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			var policy *FIFOPolicy

			if len(tt.capacity) > 0 {
				policy = NewFIFOPolicy(tt.capacity[0])
			} else {
				policy = NewFIFOPolicy()
			}

			for _, key := range tt.operations {
				err := policy.KeyAccessed(ctx, key)
				require.NoError(t, err)
			}

			size, err := policy.Size(ctx)
			require.NoError(t, err)
			assert.Equal(t, tt.wantSize, size)

			if tt.evictedKey != "" {
				has, err := policy.Has(ctx, tt.evictedKey)
				require.NoError(t, err)
				assert.False(t, has)
			}

			if tt.remainingKey != "" {
				has, err := policy.Has(ctx, tt.remainingKey)
				require.NoError(t, err)
				assert.True(t, has)
			}
		})
	}
}

// TestFIFOPolicy_DuplicateKeyAccess 测试重复访问key
func TestFIFOPolicy_DuplicateKeyAccess(t *testing.T) {
	tests := []struct {
		name       string
		operations []string
		wantSize   int
	}{
		{
			name:       "重复访问key不改变大小",
			operations: []string{"key1", "key1"},
			wantSize:   1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			policy := NewFIFOPolicy()

			for _, key := range tt.operations {
				err := policy.KeyAccessed(ctx, key)
				require.NoError(t, err)
			}

			size, err := policy.Size(ctx)
			require.NoError(t, err)
			assert.Equal(t, tt.wantSize, size)
		})
	}
}

// TestFIFOPolicy_RemoveFromMiddle 测试从中间删除节点
func TestFIFOPolicy_RemoveFromMiddle(t *testing.T) {
	tests := []struct {
		name          string
		setupKeys     []string
		removeKey     string
		wantSize      int
		removedKey    string
		remainingKeys []string
	}{
		{
			name:          "从中间删除节点",
			setupKeys:     []string{"key1", "key2", "key3"},
			removeKey:     "key2",
			wantSize:      2,
			removedKey:    "key2",
			remainingKeys: []string{"key1", "key3"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			policy := NewFIFOPolicy()

			for _, key := range tt.setupKeys {
				err := policy.KeyAccessed(ctx, key)
				require.NoError(t, err)
			}

			err := policy.Remove(ctx, tt.removeKey)
			require.NoError(t, err)

			// 验证删除的key不存在
			has, err := policy.Has(ctx, tt.removedKey)
			require.NoError(t, err)
			assert.False(t, has)

			// 验证剩余的key存在
			for _, key := range tt.remainingKeys {
				has, err := policy.Has(ctx, key)
				require.NoError(t, err)
				assert.True(t, has)
			}

			// 验证大小
			size, err := policy.Size(ctx)
			require.NoError(t, err)
			assert.Equal(t, tt.wantSize, size)
		})
	}
}

// TestFIFOPolicy_RemovePositions 测试删除不同位置的节点
func TestFIFOPolicy_RemovePositions(t *testing.T) {
	tests := []struct {
		name         string
		setupKeys    []string
		removeKey    string
		removedKey   string
		remainingKey string
	}{
		{
			name:         "删除头节点",
			setupKeys:    []string{"key1", "key2"},
			removeKey:    "key1",
			removedKey:   "key1",
			remainingKey: "key2",
		},
		{
			name:         "删除尾节点",
			setupKeys:    []string{"key1", "key2"},
			removeKey:    "key2",
			removedKey:   "key2",
			remainingKey: "key1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			policy := NewFIFOPolicy()

			for _, key := range tt.setupKeys {
				err := policy.KeyAccessed(ctx, key)
				require.NoError(t, err)
			}

			err := policy.Remove(ctx, tt.removeKey)
			require.NoError(t, err)

			// 验证删除成功
			has, err := policy.Has(ctx, tt.removedKey)
			require.NoError(t, err)
			assert.False(t, has)

			// 验证其他key还在
			has, err = policy.Has(ctx, tt.remainingKey)
			require.NoError(t, err)
			assert.True(t, has)
		})
	}
}

// TestFIFOPolicy_EdgeCases 测试边界情况
func TestFIFOPolicy_EdgeCases(t *testing.T) {
	tests := []struct {
		name       string
		setupKeys  []string
		operation  string
		operateKey string
		wantResult string
		wantSize   int
		checkKey   string
		wantHas    bool
	}{
		{
			name:       "空策略淘汰返回空字符串",
			setupKeys:  []string{},
			operation:  "evict",
			wantResult: "",
			wantSize:   0,
		},
		{
			name:       "删除唯一节点",
			setupKeys:  []string{"key1"},
			operation:  "remove",
			operateKey: "key1",
			wantSize:   0,
			checkKey:   "key1",
			wantHas:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			policy := NewFIFOPolicy()

			for _, key := range tt.setupKeys {
				err := policy.KeyAccessed(ctx, key)
				require.NoError(t, err)
			}

			switch tt.operation {
			case "evict":
				result, err := policy.Evict(ctx)
				require.NoError(t, err)
				assert.Equal(t, tt.wantResult, result)
			case "remove":
				err := policy.Remove(ctx, tt.operateKey)
				require.NoError(t, err)
			}

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

// TestFIFOPolicy_CapacityEviction 测试容量限制下的自动淘汰
func TestFIFOPolicy_CapacityEviction(t *testing.T) {
	ctx := context.Background()
	policy := NewFIFOPolicy(2) // 限制容量为2

	// 添加第一个key
	err := policy.KeyAccessed(ctx, "key1")
	require.NoError(t, err)

	// 添加第二个key
	err = policy.KeyAccessed(ctx, "key2")
	require.NoError(t, err)

	// 添加第三个key，应该触发自动淘汰
	err = policy.KeyAccessed(ctx, "key3")
	require.NoError(t, err)

	// 验证key1被淘汰
	has, err := policy.Has(ctx, "key1")
	require.NoError(t, err)
	assert.False(t, has)

	// 验证key3存在
	has, err = policy.Has(ctx, "key3")
	require.NoError(t, err)
	assert.True(t, has)
}
