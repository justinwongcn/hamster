package cache

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRandomPolicy_KeyAccessed(t *testing.T) {
	ctx := context.Background()

	t.Run("should add new key to random policy", func(t *testing.T) {
		policy := NewRandomPolicy()

		// 添加新key
		err := policy.KeyAccessed(ctx, "key1")
		require.NoError(t, err)

		size, err := policy.Size(ctx)
		require.NoError(t, err)
		assert.Equal(t, 1, size)

		has, err := policy.Has(ctx, "key1")
		require.NoError(t, err)
		assert.True(t, has)

		// 添加另一个key
		err = policy.KeyAccessed(ctx, "key2")
		require.NoError(t, err)

		size, err = policy.Size(ctx)
		require.NoError(t, err)
		assert.Equal(t, 2, size)

		has, err = policy.Has(ctx, "key2")
		require.NoError(t, err)
		assert.True(t, has)

		// 重复访问key1，不应该改变大小
		err = policy.KeyAccessed(ctx, "key1")
		require.NoError(t, err)

		size, err = policy.Size(ctx)
		require.NoError(t, err)
		assert.Equal(t, 2, size) // 大小不变
	})
}

func TestRandomPolicy_Evict(t *testing.T) {
	ctx := context.Background()

	t.Run("should return empty string when policy is empty", func(t *testing.T) {
		policy := NewRandomPolicy()
		evicted, err := policy.Evict(ctx)
		require.NoError(t, err)
		assert.Equal(t, "", evicted)
	})

	t.Run("should evict random key", func(t *testing.T) {
		policy := NewRandomPolicy()

		// 添加多个key
		keys := []string{"key1", "key2", "key3"}
		for _, key := range keys {
			err := policy.KeyAccessed(ctx, key)
			require.NoError(t, err)
		}

		size, err := policy.Size(ctx)
		require.NoError(t, err)
		assert.Equal(t, 3, size)

		// 淘汰一个key
		evicted, err := policy.Evict(ctx)
		require.NoError(t, err)
		assert.Contains(t, keys, evicted) // 应该是其中一个key

		size, err = policy.Size(ctx)
		require.NoError(t, err)
		assert.Equal(t, 2, size)

		has, err := policy.Has(ctx, evicted)
		require.NoError(t, err)
		assert.False(t, has)
	})
}

func TestRandomPolicy_Remove(t *testing.T) {
	ctx := context.Background()

	t.Run("should remove existing key", func(t *testing.T) {
		policy := NewRandomPolicy()
		err := policy.KeyAccessed(ctx, "key1")
		require.NoError(t, err)
		err = policy.KeyAccessed(ctx, "key2")
		require.NoError(t, err)

		// 移除存在的key
		err = policy.Remove(ctx, "key1")
		require.NoError(t, err)

		size, err := policy.Size(ctx)
		require.NoError(t, err)
		assert.Equal(t, 1, size)

		has, err := policy.Has(ctx, "key1")
		require.NoError(t, err)
		assert.False(t, has)

		// 移除不存在的key
		err = policy.Remove(ctx, "key3")
		require.NoError(t, err)

		size, err = policy.Size(ctx)
		require.NoError(t, err)
		assert.Equal(t, 1, size) // 大小不变
	})
}

func TestRandomPolicy_Has(t *testing.T) {
	ctx := context.Background()

	t.Run("should return correct existence status", func(t *testing.T) {
		policy := NewRandomPolicy()
		err := policy.KeyAccessed(ctx, "key1")
		require.NoError(t, err)

		has, err := policy.Has(ctx, "key1")
		require.NoError(t, err)
		assert.True(t, has)

		has, err = policy.Has(ctx, "key2")
		require.NoError(t, err)
		assert.False(t, has)
	})
}

func TestRandomPolicy_Size(t *testing.T) {
	ctx := context.Background()
	policy := NewRandomPolicy()

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

func TestRandomPolicy_Clear(t *testing.T) {
	ctx := context.Background()
	policy := NewRandomPolicy()

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

func TestRandomPolicy_Capacity(t *testing.T) {
	ctx := context.Background()

	t.Run("unlimited capacity", func(t *testing.T) {
		policy := NewRandomPolicy() // 无限制容量

		// 添加多个key
		for i := 0; i < 10; i++ {
			err := policy.KeyAccessed(ctx, fmt.Sprintf("key%d", i))
			require.NoError(t, err)
		}

		size, err := policy.Size(ctx)
		require.NoError(t, err)
		assert.Equal(t, 10, size)
	})

	t.Run("limited capacity", func(t *testing.T) {
		policy := NewRandomPolicy(3) // 限制容量为3

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
		assert.Equal(t, 3, existingCount) // 应该只有3个key存在
	})
}
