package cache

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewLRUPolicy(t *testing.T) {
	t.Run("should create empty LRU policy", func(t *testing.T) {
		policy := NewLRUPolicy()
		assert.NotNil(t, policy)

		size, err := policy.Size(context.Background())
		require.NoError(t, err)
		assert.Equal(t, 0, size)
	})
}

func TestLRUPolicy_KeyAccessed(t *testing.T) {
	ctx := context.Background()

	t.Run("should add new key to LRU policy", func(t *testing.T) {
		policy := NewLRUPolicy()
		err := policy.KeyAccessed(ctx, "key1")
		require.NoError(t, err)

		size, err := policy.Size(ctx)
		require.NoError(t, err)
		assert.Equal(t, 1, size)

		has, err := policy.Has(ctx, "key1")
		require.NoError(t, err)
		assert.True(t, has)
	})

	t.Run("should handle duplicate key access", func(t *testing.T) {
		policy := NewLRUPolicy()
		err := policy.KeyAccessed(ctx, "key1")
		require.NoError(t, err)
		err = policy.KeyAccessed(ctx, "key2")
		require.NoError(t, err)
		err = policy.KeyAccessed(ctx, "key1") // 重复访问
		require.NoError(t, err)

		size, err := policy.Size(ctx)
		require.NoError(t, err)
		assert.Equal(t, 2, size) // 应该还是2个key

		// key1应该是最近使用的，所以key2应该先被淘汰
		evicted, err := policy.Evict(ctx)
		require.NoError(t, err)
		assert.Equal(t, "key2", evicted)
	})
}

func TestLRUPolicy_Evict(t *testing.T) {
	ctx := context.Background()

	t.Run("should return empty string when policy is empty", func(t *testing.T) {
		policy := NewLRUPolicy()
		evicted, err := policy.Evict(ctx)
		require.NoError(t, err)
		assert.Equal(t, "", evicted)
	})

	t.Run("should evict least recently used key", func(t *testing.T) {
		policy := NewLRUPolicy()
		err := policy.KeyAccessed(ctx, "key1")
		require.NoError(t, err)
		err = policy.KeyAccessed(ctx, "key2")
		require.NoError(t, err)

		size, err := policy.Size(ctx)
		require.NoError(t, err)
		assert.Equal(t, 2, size)

		evicted, err := policy.Evict(ctx)
		require.NoError(t, err)
		assert.Equal(t, "key1", evicted)

		size, err = policy.Size(ctx)
		require.NoError(t, err)
		assert.Equal(t, 1, size)

		has, err := policy.Has(ctx, "key1")
		require.NoError(t, err)
		assert.False(t, has)
	})

	t.Run("should maintain correct order after eviction", func(t *testing.T) {
		policy := NewLRUPolicy()
		err := policy.KeyAccessed(ctx, "key1")
		require.NoError(t, err)
		err = policy.KeyAccessed(ctx, "key2")
		require.NoError(t, err)
		err = policy.KeyAccessed(ctx, "key3")
		require.NoError(t, err)

		evicted, err := policy.Evict(ctx)
		require.NoError(t, err)
		assert.Equal(t, "key1", evicted)

		evicted, err = policy.Evict(ctx)
		require.NoError(t, err)
		assert.Equal(t, "key2", evicted)
	})
}

func TestLRUPolicy_Remove(t *testing.T) {
	ctx := context.Background()

	t.Run("should remove existing key", func(t *testing.T) {
		policy := NewLRUPolicy()
		err := policy.KeyAccessed(ctx, "key1")
		require.NoError(t, err)
		err = policy.Remove(ctx, "key1")
		require.NoError(t, err)

		size, err := policy.Size(ctx)
		require.NoError(t, err)
		assert.Equal(t, 0, size)

		has, err := policy.Has(ctx, "key1")
		require.NoError(t, err)
		assert.False(t, has)
	})

	t.Run("should do nothing when key does not exist", func(t *testing.T) {
		policy := NewLRUPolicy()
		err := policy.KeyAccessed(ctx, "key1")
		require.NoError(t, err)
		err = policy.Remove(ctx, "key2")
		require.NoError(t, err)

		size, err := policy.Size(ctx)
		require.NoError(t, err)
		assert.Equal(t, 1, size)

		has, err := policy.Has(ctx, "key1")
		require.NoError(t, err)
		assert.True(t, has)
	})
}

func TestLRUPolicy_Has(t *testing.T) {
	ctx := context.Background()

	t.Run("should return true for existing key", func(t *testing.T) {
		policy := NewLRUPolicy()
		err := policy.KeyAccessed(ctx, "key1")
		require.NoError(t, err)

		has, err := policy.Has(ctx, "key1")
		require.NoError(t, err)
		assert.True(t, has)
	})

	t.Run("should return false for non-existent key", func(t *testing.T) {
		policy := NewLRUPolicy()
		has, err := policy.Has(ctx, "key1")
		require.NoError(t, err)
		assert.False(t, has)
	})
}

func TestLRUPolicy_CombinedOperations(t *testing.T) {
	ctx := context.Background()
	policy := NewLRUPolicy()

	// Add initial keys
	err := policy.KeyAccessed(ctx, "key1")
	require.NoError(t, err)
	err = policy.KeyAccessed(ctx, "key2")
	require.NoError(t, err)
	err = policy.KeyAccessed(ctx, "key3")
	require.NoError(t, err)

	// Access key1 to move it to end
	err = policy.KeyAccessed(ctx, "key1")
	require.NoError(t, err)

	// Evict key2 (LRU)
	evicted, err := policy.Evict(ctx)
	require.NoError(t, err)
	assert.Equal(t, "key2", evicted)

	// Remove key3
	err = policy.Remove(ctx, "key3")
	require.NoError(t, err)

	// Verify final state
	size, err := policy.Size(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, size)

	has, err := policy.Has(ctx, "key1")
	require.NoError(t, err)
	assert.True(t, has)

	has, err = policy.Has(ctx, "key2")
	require.NoError(t, err)
	assert.False(t, has)

	has, err = policy.Has(ctx, "key3")
	require.NoError(t, err)
	assert.False(t, has)
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
		for i := 0; i < 100; i++ {
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
