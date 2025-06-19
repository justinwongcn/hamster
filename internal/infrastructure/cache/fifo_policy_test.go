package cache

import (
	"testing"
	"github.com/stretchr/testify/assert"
)

func TestFIFOPolicy(t *testing.T) {
	t.Run("KeyAccessed", func(t *testing.T) {
		fifo := NewFIFOPolicy()
		
		// 添加新key
		fifo.KeyAccessed("key1")
		assert.Equal(t, 1, fifo.list.Len())
		assert.True(t, fifo.Has("key1"))
		
		// 添加另一个key
		fifo.KeyAccessed("key2")
		assert.Equal(t, 2, fifo.list.Len())
		assert.True(t, fifo.Has("key2"))
		
		// 重复添加已存在key
		fifo.KeyAccessed("key1")
		assert.Equal(t, 2, fifo.list.Len()) // 长度不变
	})

	t.Run("Evict", func(t *testing.T) {
		fifo := NewFIFOPolicy()
		
		// 空链表
		assert.Equal(t, "", fifo.Evict())
		
		// 添加key后淘汰
		fifo.KeyAccessed("key1")
		fifo.KeyAccessed("key2")
		assert.Equal(t, "key1", fifo.Evict()) // 最早添加的key
		assert.Equal(t, 1, fifo.list.Len())
		assert.False(t, fifo.Has("key1"))
	})

	t.Run("Remove", func(t *testing.T) {
		fifo := NewFIFOPolicy()
		fifo.KeyAccessed("key1")
		fifo.KeyAccessed("key2")
		
		// 移除存在的key
		fifo.Remove("key1")
		assert.Equal(t, 1, fifo.list.Len())
		assert.False(t, fifo.Has("key1"))
		
		// 移除不存在的key
		fifo.Remove("key3")
		assert.Equal(t, 1, fifo.list.Len()) // 长度不变
	})

	t.Run("Has", func(t *testing.T) {
		fifo := NewFIFOPolicy()
		fifo.KeyAccessed("key1")
		
		assert.True(t, fifo.Has("key1"))
		assert.False(t, fifo.Has("key2"))
	})
}