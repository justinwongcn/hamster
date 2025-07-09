package hamster

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// MockCache is a mock implementation of the Cache interface for testing
type MockCache struct {
	data      map[string]any
	onEvicted func(key string, val any)
}

func NewMockCache() *MockCache {
	return &MockCache{
		data: make(map[string]any),
	}
}

func (m *MockCache) Set(ctx context.Context, key string, val any, expiration time.Duration) error {
	m.data[key] = val
	return nil
}

func (m *MockCache) Get(ctx context.Context, key string) (any, error) {
	val, exists := m.data[key]
	if !exists {
		return nil, assert.AnError
	}
	return val, nil
}

func (m *MockCache) Delete(ctx context.Context, key string) error {
	delete(m.data, key)
	return nil
}

func (m *MockCache) LoadAndDelete(ctx context.Context, key string) (any, error) {
	val, exists := m.data[key]
	if !exists {
		return nil, assert.AnError
	}
	delete(m.data, key)
	return val, nil
}

func (m *MockCache) OnEvicted(fn func(key string, val any)) {
	m.onEvicted = fn
}

func TestCacheInterface(t *testing.T) {
	// Test that MockCache implements Cache interface
	var cache Cache = NewMockCache()
	assert.NotNil(t, cache)

	ctx := context.Background()

	// Test Set
	err := cache.Set(ctx, "test_key", "test_value", time.Hour)
	assert.NoError(t, err)

	// Test Get
	value, err := cache.Get(ctx, "test_key")
	assert.NoError(t, err)
	assert.Equal(t, "test_value", value)

	// Test Delete
	err = cache.Delete(ctx, "test_key")
	assert.NoError(t, err)

	// Test Get after delete
	_, err = cache.Get(ctx, "test_key")
	assert.Error(t, err)
}

func TestCacheInterface_LoadAndDelete(t *testing.T) {
	var cache Cache = NewMockCache()
	ctx := context.Background()

	// Set a value
	err := cache.Set(ctx, "test_key", "test_value", time.Hour)
	assert.NoError(t, err)

	// LoadAndDelete
	value, err := cache.LoadAndDelete(ctx, "test_key")
	assert.NoError(t, err)
	assert.Equal(t, "test_value", value)

	// Verify it's deleted
	_, err = cache.Get(ctx, "test_key")
	assert.Error(t, err)
}

func TestCacheInterface_OnEvicted(t *testing.T) {
	mockCache := NewMockCache()
	var cache Cache = mockCache

	evictedKeys := make([]string, 0)
	evictedValues := make([]any, 0)

	// Set eviction callback
	cache.OnEvicted(func(key string, val any) {
		evictedKeys = append(evictedKeys, key)
		evictedValues = append(evictedValues, val)
	})

	// Verify callback is set
	assert.NotNil(t, mockCache.onEvicted)

	// Simulate eviction
	mockCache.onEvicted("evicted_key", "evicted_value")

	assert.Len(t, evictedKeys, 1)
	assert.Equal(t, "evicted_key", evictedKeys[0])
	assert.Equal(t, "evicted_value", evictedValues[0])
}

func TestCacheInterface_WithDifferentValueTypes(t *testing.T) {
	var cache Cache = NewMockCache()
	ctx := context.Background()

	// Test with string
	err := cache.Set(ctx, "string_key", "string_value", time.Hour)
	assert.NoError(t, err)

	value, err := cache.Get(ctx, "string_key")
	assert.NoError(t, err)
	assert.Equal(t, "string_value", value)

	// Test with int
	err = cache.Set(ctx, "int_key", 42, time.Hour)
	assert.NoError(t, err)

	value, err = cache.Get(ctx, "int_key")
	assert.NoError(t, err)
	assert.Equal(t, 42, value)

	// Test with struct
	type TestStruct struct {
		Name string
		Age  int
	}
	testStruct := TestStruct{Name: "John", Age: 30}

	err = cache.Set(ctx, "struct_key", testStruct, time.Hour)
	assert.NoError(t, err)

	value, err = cache.Get(ctx, "struct_key")
	assert.NoError(t, err)
	assert.Equal(t, testStruct, value)

	// Test with slice
	testSlice := []string{"a", "b", "c"}
	err = cache.Set(ctx, "slice_key", testSlice, time.Hour)
	assert.NoError(t, err)

	value, err = cache.Get(ctx, "slice_key")
	assert.NoError(t, err)
	assert.Equal(t, testSlice, value)

	// Test with map
	testMap := map[string]int{"one": 1, "two": 2}
	err = cache.Set(ctx, "map_key", testMap, time.Hour)
	assert.NoError(t, err)

	value, err = cache.Get(ctx, "map_key")
	assert.NoError(t, err)
	assert.Equal(t, testMap, value)
}

func TestCacheInterface_WithContext(t *testing.T) {
	var cache Cache = NewMockCache()

	// Test with background context
	ctx := context.Background()
	err := cache.Set(ctx, "bg_key", "bg_value", time.Hour)
	assert.NoError(t, err)

	// Test with timeout context
	timeoutCtx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	err = cache.Set(timeoutCtx, "timeout_key", "timeout_value", time.Hour)
	assert.NoError(t, err)

	value, err := cache.Get(timeoutCtx, "timeout_key")
	assert.NoError(t, err)
	assert.Equal(t, "timeout_value", value)

	// Test with canceled context
	cancelCtx, cancelFunc := context.WithCancel(context.Background())
	cancelFunc() // Cancel immediately

	err = cache.Set(cancelCtx, "cancel_key", "cancel_value", time.Hour)
	// MockCache doesn't check context cancellation, so this should still work
	assert.NoError(t, err)
}

func TestCacheInterface_WithDifferentExpirations(t *testing.T) {
	var cache Cache = NewMockCache()
	ctx := context.Background()

	// Test with different expiration times
	expirations := []time.Duration{
		0,                    // No expiration
		time.Second,          // 1 second
		time.Minute,          // 1 minute
		time.Hour,            // 1 hour
		24 * time.Hour,       // 1 day
		365 * 24 * time.Hour, // 1 year
	}

	for i, expiration := range expirations {
		key := fmt.Sprintf("key_%d", i)
		value := fmt.Sprintf("value_%d", i)

		err := cache.Set(ctx, key, value, expiration)
		assert.NoError(t, err)

		retrievedValue, err := cache.Get(ctx, key)
		assert.NoError(t, err)
		assert.Equal(t, value, retrievedValue)
	}
}

func TestCacheInterface_EdgeCases(t *testing.T) {
	var cache Cache = NewMockCache()
	ctx := context.Background()

	// Test with empty key
	err := cache.Set(ctx, "", "empty_key_value", time.Hour)
	assert.NoError(t, err)

	value, err := cache.Get(ctx, "")
	assert.NoError(t, err)
	assert.Equal(t, "empty_key_value", value)

	// Test with nil value
	err = cache.Set(ctx, "nil_key", nil, time.Hour)
	assert.NoError(t, err)

	value, err = cache.Get(ctx, "nil_key")
	assert.NoError(t, err)
	assert.Nil(t, value)

	// Test with very long key
	longKey := string(make([]byte, 1000))
	err = cache.Set(ctx, longKey, "long_key_value", time.Hour)
	assert.NoError(t, err)

	value, err = cache.Get(ctx, longKey)
	assert.NoError(t, err)
	assert.Equal(t, "long_key_value", value)
}

func TestCacheInterface_ConcurrentAccess(t *testing.T) {
	var cache Cache = NewMockCache()
	ctx := context.Background()

	// Note: MockCache is not thread-safe, but we test the interface
	// Real implementations should be thread-safe

	// Set multiple values
	for i := 0; i < 10; i++ {
		key := fmt.Sprintf("concurrent_key_%d", i)
		value := fmt.Sprintf("concurrent_value_%d", i)

		err := cache.Set(ctx, key, value, time.Hour)
		assert.NoError(t, err)
	}

	// Get multiple values
	for i := 0; i < 10; i++ {
		key := fmt.Sprintf("concurrent_key_%d", i)
		expectedValue := fmt.Sprintf("concurrent_value_%d", i)

		value, err := cache.Get(ctx, key)
		assert.NoError(t, err)
		assert.Equal(t, expectedValue, value)
	}
}
