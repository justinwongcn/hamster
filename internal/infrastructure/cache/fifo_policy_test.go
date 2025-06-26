package cache

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFIFOPolicy_KeyAccessed(t *testing.T) {
	ctx := context.Background()

	t.Run("should add new key to FIFO policy", func(t *testing.T) {
		fifo := NewFIFOPolicy()

		// 添加新key
		err := fifo.KeyAccessed(ctx, "key1")
		require.NoError(t, err)

		size, err := fifo.Size(ctx)
		require.NoError(t, err)
		assert.Equal(t, 1, size)

		has, err := fifo.Has(ctx, "key1")
		require.NoError(t, err)
		assert.True(t, has)

		// 添加另一个key
		err = fifo.KeyAccessed(ctx, "key2")
		require.NoError(t, err)

		size, err = fifo.Size(ctx)
		require.NoError(t, err)
		assert.Equal(t, 2, size)

		has, err = fifo.Has(ctx, "key2")
		require.NoError(t, err)
		assert.True(t, has)

		// 重复访问key1，不应该改变位置（FIFO特性）
		err = fifo.KeyAccessed(ctx, "key1")
		require.NoError(t, err)

		size, err = fifo.Size(ctx)
		require.NoError(t, err)
		assert.Equal(t, 2, size) // 长度不变
	})
}

func TestFIFOPolicy_Evict(t *testing.T) {
	ctx := context.Background()

	t.Run("should return empty string when policy is empty", func(t *testing.T) {
		fifo := NewFIFOPolicy()
		evicted, err := fifo.Evict(ctx)
		require.NoError(t, err)
		assert.Equal(t, "", evicted)
	})

	t.Run("should evict first in key", func(t *testing.T) {
		fifo := NewFIFOPolicy()

		// 添加key后淘汰
		err := fifo.KeyAccessed(ctx, "key1")
		require.NoError(t, err)
		err = fifo.KeyAccessed(ctx, "key2")
		require.NoError(t, err)

		evicted, err := fifo.Evict(ctx)
		require.NoError(t, err)
		assert.Equal(t, "key1", evicted) // 最早添加的key

		size, err := fifo.Size(ctx)
		require.NoError(t, err)
		assert.Equal(t, 1, size)

		has, err := fifo.Has(ctx, "key1")
		require.NoError(t, err)
		assert.False(t, has)
	})
}

func TestFIFOPolicy_Remove(t *testing.T) {
	ctx := context.Background()

	t.Run("should remove existing key", func(t *testing.T) {
		fifo := NewFIFOPolicy()
		err := fifo.KeyAccessed(ctx, "key1")
		require.NoError(t, err)
		err = fifo.KeyAccessed(ctx, "key2")
		require.NoError(t, err)

		// 移除存在的key
		err = fifo.Remove(ctx, "key1")
		require.NoError(t, err)

		size, err := fifo.Size(ctx)
		require.NoError(t, err)
		assert.Equal(t, 1, size)

		has, err := fifo.Has(ctx, "key1")
		require.NoError(t, err)
		assert.False(t, has)

		// 移除不存在的key
		err = fifo.Remove(ctx, "key3")
		require.NoError(t, err)

		size, err = fifo.Size(ctx)
		require.NoError(t, err)
		assert.Equal(t, 1, size) // 长度不变
	})
}

func TestFIFOPolicy_Has(t *testing.T) {
	ctx := context.Background()

	t.Run("should return correct existence status", func(t *testing.T) {
		fifo := NewFIFOPolicy()
		err := fifo.KeyAccessed(ctx, "key1")
		require.NoError(t, err)

		has, err := fifo.Has(ctx, "key1")
		require.NoError(t, err)
		assert.True(t, has)

		has, err = fifo.Has(ctx, "key2")
		require.NoError(t, err)
		assert.False(t, has)
	})
}

// TestFIFOPolicy_Size 测试Size方法
func TestFIFOPolicy_Size(t *testing.T) {
	ctx := context.Background()
	policy := NewFIFOPolicy()

	// 初始大小应该为0
	size, err := policy.Size(ctx)
	require.NoError(t, err)
	assert.Equal(t, 0, size)

	// 添加一些key
	err = policy.KeyAccessed(ctx, "key1")
	require.NoError(t, err)
	err = policy.KeyAccessed(ctx, "key2")
	require.NoError(t, err)

	size, err = policy.Size(ctx)
	require.NoError(t, err)
	assert.Equal(t, 2, size)
}

// TestFIFOPolicy_Clear 测试Clear方法
func TestFIFOPolicy_Clear(t *testing.T) {
	ctx := context.Background()
	policy := NewFIFOPolicy()

	// 添加一些key
	err := policy.KeyAccessed(ctx, "key1")
	require.NoError(t, err)
	err = policy.KeyAccessed(ctx, "key2")
	require.NoError(t, err)

	// 清空
	err = policy.Clear(ctx)
	require.NoError(t, err)

	// 验证大小为0
	size, err := policy.Size(ctx)
	require.NoError(t, err)
	assert.Equal(t, 0, size)

	// 验证key不存在
	has, err := policy.Has(ctx, "key1")
	require.NoError(t, err)
	assert.False(t, has)
}

// TestFIFOPolicy_Capacity 测试容量限制功能
func TestFIFOPolicy_Capacity(t *testing.T) {
	ctx := context.Background()

	t.Run("unlimited capacity", func(t *testing.T) {
		policy := NewFIFOPolicy() // 无限制容量

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
		policy := NewFIFOPolicy(3) // 限制容量为3

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

		// key1应该被淘汰（FIFO）
		has, err := policy.Has(ctx, "key1")
		require.NoError(t, err)
		assert.False(t, has)

		// key4应该还在
		has, err = policy.Has(ctx, "key4")
		require.NoError(t, err)
		assert.True(t, has)
	})
}

// TestFIFOPolicy_DuplicateKeyAccess 测试重复访问key
func TestFIFOPolicy_DuplicateKeyAccess(t *testing.T) {
	ctx := context.Background()
	policy := NewFIFOPolicy()

	// 添加key
	err := policy.KeyAccessed(ctx, "key1")
	require.NoError(t, err)

	// 重复添加同一个key
	err = policy.KeyAccessed(ctx, "key1")
	require.NoError(t, err)

	// 大小应该还是1
	size, err := policy.Size(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, size)
}

// TestFIFOPolicy_RemoveFromMiddle 测试从中间删除节点
func TestFIFOPolicy_RemoveFromMiddle(t *testing.T) {
	ctx := context.Background()
	policy := NewFIFOPolicy()

	// 添加多个key
	err := policy.KeyAccessed(ctx, "key1")
	require.NoError(t, err)
	err = policy.KeyAccessed(ctx, "key2")
	require.NoError(t, err)
	err = policy.KeyAccessed(ctx, "key3")
	require.NoError(t, err)

	// 删除中间的key
	err = policy.Remove(ctx, "key2")
	require.NoError(t, err)

	// 验证删除成功
	has, err := policy.Has(ctx, "key2")
	require.NoError(t, err)
	assert.False(t, has)

	// 验证其他key还在
	has, err = policy.Has(ctx, "key1")
	require.NoError(t, err)
	assert.True(t, has)

	has, err = policy.Has(ctx, "key3")
	require.NoError(t, err)
	assert.True(t, has)

	// 验证大小
	size, err := policy.Size(ctx)
	require.NoError(t, err)
	assert.Equal(t, 2, size)
}

// TestFIFOPolicy_RemoveHead 测试删除头节点
func TestFIFOPolicy_RemoveHead(t *testing.T) {
	ctx := context.Background()
	policy := NewFIFOPolicy()

	// 添加多个key
	err := policy.KeyAccessed(ctx, "key1")
	require.NoError(t, err)
	err = policy.KeyAccessed(ctx, "key2")
	require.NoError(t, err)

	// 删除头节点
	err = policy.Remove(ctx, "key1")
	require.NoError(t, err)

	// 验证删除成功
	has, err := policy.Has(ctx, "key1")
	require.NoError(t, err)
	assert.False(t, has)

	// 验证其他key还在
	has, err = policy.Has(ctx, "key2")
	require.NoError(t, err)
	assert.True(t, has)
}

// TestFIFOPolicy_RemoveTail 测试删除尾节点
func TestFIFOPolicy_RemoveTail(t *testing.T) {
	ctx := context.Background()
	policy := NewFIFOPolicy()

	// 添加多个key
	err := policy.KeyAccessed(ctx, "key1")
	require.NoError(t, err)
	err = policy.KeyAccessed(ctx, "key2")
	require.NoError(t, err)

	// 删除尾节点
	err = policy.Remove(ctx, "key2")
	require.NoError(t, err)

	// 验证删除成功
	has, err := policy.Has(ctx, "key2")
	require.NoError(t, err)
	assert.False(t, has)

	// 验证其他key还在
	has, err = policy.Has(ctx, "key1")
	require.NoError(t, err)
	assert.True(t, has)
}

// TestFIFOPolicy_EvictEmpty 测试空策略的淘汰
func TestFIFOPolicy_EvictEmpty(t *testing.T) {
	ctx := context.Background()
	policy := NewFIFOPolicy()

	// 空策略淘汰应该返回空字符串
	key, err := policy.Evict(ctx)
	require.NoError(t, err)
	assert.Equal(t, "", key)
}

// TestFIFOPolicy_RemoveSingleNode 测试删除唯一节点
func TestFIFOPolicy_RemoveSingleNode(t *testing.T) {
	ctx := context.Background()
	policy := NewFIFOPolicy()

	// 添加一个key
	err := policy.KeyAccessed(ctx, "key1")
	require.NoError(t, err)

	// 删除这个key
	err = policy.Remove(ctx, "key1")
	require.NoError(t, err)

	// 验证删除成功
	has, err := policy.Has(ctx, "key1")
	require.NoError(t, err)
	assert.False(t, has)

	// 验证大小为0
	size, err := policy.Size(ctx)
	require.NoError(t, err)
	assert.Equal(t, 0, size)
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
