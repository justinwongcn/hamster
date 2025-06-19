package cache

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewLRUPolicy(t *testing.T) {
	t.Run("should create empty LRU policy", func(t *testing.T) {
		policy := NewLRUPolicy()
		assert.NotNil(t, policy)
		assert.Equal(t, 0, policy.list.Len())
		assert.Equal(t, 0, len(policy.keys))
	})
}

func TestLRUPolicy_KeyAccessed(t *testing.T) {
	t.Run("should add new key to LRU policy", func(t *testing.T) {
		policy := NewLRUPolicy()
		policy.KeyAccessed("key1")
		assert.Equal(t, 1, policy.list.Len())
		assert.True(t, policy.Has("key1"))
	})

	t.Run("should move existing key to end of list", func(t *testing.T) {
		policy := NewLRUPolicy()
		policy.KeyAccessed("key1")
		policy.KeyAccessed("key2")
		policy.KeyAccessed("key1")
		lastKey, _ := policy.list.Get(policy.list.Len() - 1)
		assert.Equal(t, "key1", lastKey)
	})
}

func TestLRUPolicy_Evict(t *testing.T) {
	t.Run("should return empty string when policy is empty", func(t *testing.T) {
		policy := NewLRUPolicy()
		assert.Equal(t, "", policy.Evict())
	})

	t.Run("should evict least recently used key", func(t *testing.T) {
		policy := NewLRUPolicy()
		policy.KeyAccessed("key1")
		policy.KeyAccessed("key2")
		assert.Equal(t, 2, policy.list.Len())
		assert.Equal(t, "key1", policy.Evict())
		assert.Equal(t, 1, policy.list.Len())
		assert.False(t, policy.Has("key1"))
	})

	t.Run("should maintain correct order after eviction", func(t *testing.T) {
		policy := NewLRUPolicy()
		policy.KeyAccessed("key1")
		policy.KeyAccessed("key2")
		policy.KeyAccessed("key3")
		policy.Evict()
		assert.Equal(t, "key2", policy.Evict())
	})
}

func TestLRUPolicy_Remove(t *testing.T) {
	t.Run("should remove existing key", func(t *testing.T) {
		policy := NewLRUPolicy()
		policy.KeyAccessed("key1")
		policy.Remove("key1")
		assert.Equal(t, 0, policy.list.Len())
		assert.False(t, policy.Has("key1"))
	})

	t.Run("should do nothing when key does not exist", func(t *testing.T) {
		policy := NewLRUPolicy()
		policy.KeyAccessed("key1")
		policy.Remove("key2")
		assert.Equal(t, 1, policy.list.Len())
		assert.True(t, policy.Has("key1"))
	})
}

func TestLRUPolicy_Has(t *testing.T) {
	t.Run("should return true for existing key", func(t *testing.T) {
		policy := NewLRUPolicy()
		policy.KeyAccessed("key1")
		assert.True(t, policy.Has("key1"))
	})

	t.Run("should return false for non-existent key", func(t *testing.T) {
		policy := NewLRUPolicy()
		assert.False(t, policy.Has("key1"))
	})
}

func TestLRUPolicy_CombinedOperations(t *testing.T) {
	policy := NewLRUPolicy()

	// Add initial keys
	policy.KeyAccessed("key1")
	policy.KeyAccessed("key2")
	policy.KeyAccessed("key3")

	// Access key1 to move it to end
	policy.KeyAccessed("key1")

	// Evict key2 (LRU)
	assert.Equal(t, "key2", policy.Evict())

	// Remove key3
	policy.Remove("key3")

	// Verify final state
	assert.Equal(t, 1, policy.list.Len())
	assert.True(t, policy.Has("key1"))
	assert.False(t, policy.Has("key2"))
	assert.False(t, policy.Has("key3"))
}
