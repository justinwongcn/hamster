package cache

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	domainCache "github.com/justinwongcn/hamster/internal/domain/cache"
)

// TestBloomFilterConfig 测试布隆过滤器配置
func TestBloomFilterConfig(t *testing.T) {
	tests := []struct {
		name              string
		expectedElements  uint64
		falsePositiveRate float64
		wantErr           bool
		wantBitArraySize  uint64
		wantHashFunctions uint64
	}{
		{
			name:              "正常配置_1000元素_1%假阳性率",
			expectedElements:  1000,
			falsePositiveRate: 0.01,
			wantErr:           false,
			wantBitArraySize:  9586, // 理论计算值
			wantHashFunctions: 7,    // 理论计算值
		},
		{
			name:              "正常配置_10000元素_0.1%假阳性率",
			expectedElements:  10000,
			falsePositiveRate: 0.001,
			wantErr:           false,
			wantBitArraySize:  143776,
			wantHashFunctions: 10,
		},
		{
			name:              "无效配置_元素数量为0",
			expectedElements:  0,
			falsePositiveRate: 0.01,
			wantErr:           true,
		},
		{
			name:              "无效配置_假阳性率为0",
			expectedElements:  1000,
			falsePositiveRate: 0,
			wantErr:           true,
		},
		{
			name:              "无效配置_假阳性率为1",
			expectedElements:  1000,
			falsePositiveRate: 1,
			wantErr:           true,
		},
		{
			name:              "无效配置_假阳性率大于1",
			expectedElements:  1000,
			falsePositiveRate: 1.5,
			wantErr:           true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config, err := domainCache.NewBloomFilterConfig(tt.expectedElements, tt.falsePositiveRate)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.expectedElements, config.ExpectedElements())
			assert.Equal(t, tt.falsePositiveRate, config.FalsePositiveRate())

			// 验证计算的位数组大小和哈希函数数量在合理范围内
			assert.Greater(t, config.BitArraySize(), uint64(0))
			assert.Greater(t, config.HashFunctions(), uint64(0))
			assert.Greater(t, config.MemoryUsage(), uint64(0))
		})
	}
}

// TestInMemoryBloomFilter 测试内存布隆过滤器基本功能
func TestInMemoryBloomFilter(t *testing.T) {
	tests := []struct {
		name            string
		config          domainCache.BloomFilterConfig
		addKeys         []string
		testKeys        []string
		expectedResults []bool // true表示应该返回true（可能存在），false表示应该返回false（一定不存在）
	}{
		{
			name: "基本添加和查询",
			config: func() domainCache.BloomFilterConfig {
				config, _ := domainCache.NewBloomFilterConfig(100, 0.01)
				return config
			}(),
			addKeys:         []string{"key1", "key2", "key3"},
			testKeys:        []string{"key1", "key2", "key3", "key4", "key5"},
			expectedResults: []bool{true, true, true, false, false},
		},
		{
			name: "大量数据测试",
			config: func() domainCache.BloomFilterConfig {
				config, _ := domainCache.NewBloomFilterConfig(1000, 0.01)
				return config
			}(),
			addKeys: func() []string {
				keys := make([]string, 500)
				for i := 0; i < 500; i++ {
					keys[i] = fmt.Sprintf("key_%d", i)
				}
				return keys
			}(),
			testKeys:        []string{"key_0", "key_100", "key_499", "key_500", "key_1000"},
			expectedResults: []bool{true, true, true, false, false},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bf := NewInMemoryBloomFilter(tt.config)
			ctx := context.Background()

			// 添加键
			for _, key := range tt.addKeys {
				err := bf.Add(ctx, key)
				require.NoError(t, err)
			}

			// 测试查询
			for i, key := range tt.testKeys {
				result := bf.HasKey(ctx, key)
				if tt.expectedResults[i] {
					assert.True(t, result, "键 %s 应该返回 true（可能存在）", key)
				} else {
					assert.False(t, result, "键 %s 应该返回 false（一定不存在）", key)
				}
			}

			// 测试统计信息
			stats, err := bf.Stats(ctx)
			require.NoError(t, err)
			assert.Equal(t, uint64(len(tt.addKeys)), stats.AddedElements())
			assert.Greater(t, stats.SetBits(), uint64(0))
			assert.Greater(t, stats.LoadFactor(), 0.0)
		})
	}
}

// TestBloomFilterFalsePositiveRate 测试假阳性率
func TestBloomFilterFalsePositiveRate(t *testing.T) {
	config, err := domainCache.NewBloomFilterConfig(1000, 0.01)
	require.NoError(t, err)

	bf := NewInMemoryBloomFilter(config)
	ctx := context.Background()

	// 添加1000个键
	addedKeys := make(map[string]bool)
	for i := 0; i < 1000; i++ {
		key := fmt.Sprintf("added_key_%d", i)
		addedKeys[key] = true
		err := bf.Add(ctx, key)
		require.NoError(t, err)
	}

	// 测试10000个未添加的键
	falsePositives := 0
	testCount := 10000
	for i := 0; i < testCount; i++ {
		key := fmt.Sprintf("test_key_%d", i)
		if !addedKeys[key] && bf.HasKey(ctx, key) {
			falsePositives++
		}
	}

	// 计算实际假阳性率
	actualFPR := float64(falsePositives) / float64(testCount)

	// 实际假阳性率应该接近期望值（允许一定误差）
	assert.Less(t, actualFPR, 0.05, "实际假阳性率应该小于5%")

	// 获取估算的假阳性率
	estimatedFPR, err := bf.EstimateFalsePositiveRate(ctx)
	require.NoError(t, err)
	assert.Greater(t, estimatedFPR, 0.0)
}

// TestBloomFilterClear 测试清空功能
func TestBloomFilterClear(t *testing.T) {
	config, err := domainCache.NewBloomFilterConfig(100, 0.01)
	require.NoError(t, err)

	bf := NewInMemoryBloomFilter(config)
	ctx := context.Background()

	// 添加一些键
	keys := []string{"key1", "key2", "key3"}
	for _, key := range keys {
		err := bf.Add(ctx, key)
		require.NoError(t, err)
	}

	// 验证键存在
	for _, key := range keys {
		assert.True(t, bf.HasKey(ctx, key))
	}

	// 清空布隆过滤器
	err = bf.Clear(ctx)
	require.NoError(t, err)

	// 验证键不存在
	for _, key := range keys {
		assert.False(t, bf.HasKey(ctx, key))
	}

	// 验证统计信息被重置
	stats, err := bf.Stats(ctx)
	require.NoError(t, err)
	assert.Equal(t, uint64(0), stats.AddedElements())
	assert.Equal(t, uint64(0), stats.SetBits())
}

// TestBloomFilterConcurrency 测试并发安全
func TestBloomFilterConcurrency(t *testing.T) {
	config, err := domainCache.NewBloomFilterConfig(1000, 0.01)
	require.NoError(t, err)

	bf := NewInMemoryBloomFilter(config)
	ctx := context.Background()

	const numGoroutines = 10
	const keysPerGoroutine = 100

	// 并发添加键
	var wg sync.WaitGroup
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < keysPerGoroutine; j++ {
				key := fmt.Sprintf("key_%d_%d", id, j)
				_ = bf.Add(ctx, key)
			}
		}(i)
	}

	wg.Wait()

	// 验证所有键都能被找到
	for i := 0; i < numGoroutines; i++ {
		for j := 0; j < keysPerGoroutine; j++ {
			key := fmt.Sprintf("key_%d_%d", i, j)
			assert.True(t, bf.HasKey(ctx, key), "键 %s 应该存在", key)
		}
	}

	// 验证统计信息
	stats, err := bf.Stats(ctx)
	require.NoError(t, err)
	assert.Equal(t, uint64(numGoroutines*keysPerGoroutine), stats.AddedElements())
}

// TestBloomFilterKey 测试布隆过滤器键值对象
func TestBloomFilterKey(t *testing.T) {
	tests := []struct {
		name    string
		keyStr  string
		wantErr bool
	}{
		{
			name:    "正常键",
			keyStr:  "normal_key",
			wantErr: false,
		},
		{
			name:    "空键",
			keyStr:  "",
			wantErr: true,
		},
		{
			name:    "长键",
			keyStr:  string(make([]byte, 1001)), // 超过1000字符
			wantErr: true,
		},
		{
			name:    "边界长度键",
			keyStr:  string(make([]byte, 1000)), // 正好1000字符
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key, err := domainCache.NewBloomFilterKey(tt.keyStr)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.keyStr, key.String())
			assert.Equal(t, []byte(tt.keyStr), key.Bytes())

			// 测试哈希函数
			hash1 := key.Hash(0)
			hash2 := key.Hash(1)
			assert.NotEqual(t, hash1, hash2, "不同种子应该产生不同的哈希值")

			// 测试相等性
			key2, _ := domainCache.NewBloomFilterKey(tt.keyStr)
			assert.True(t, key.Equals(key2))
		})
	}
}

// TestBloomFilterCache 测试布隆过滤器缓存
func TestBloomFilterCache(t *testing.T) {
	tests := []struct {
		name               string
		setupCache         func() *MockCache
		setupBloomFilter   func() *InMemoryBloomFilter
		setupLoadFunc      func() (func(ctx context.Context, key string) (any, error), *[]string)
		preAddToBloom      []string
		testKey            string
		wantValue          any
		wantErr            error
		wantLoadCalled     bool
		wantBloomFilterHit bool
	}{
		{
			name: "缓存命中_不查询布隆过滤器",
			setupCache: func() *MockCache {
				return &MockCache{store: map[string]any{"key1": "cached_value"}}
			},
			setupBloomFilter: func() *InMemoryBloomFilter {
				config, _ := domainCache.NewBloomFilterConfig(100, 0.01)
				return NewInMemoryBloomFilter(config)
			},
			setupLoadFunc: func() (func(ctx context.Context, key string) (any, error), *[]string) {
				called := make([]string, 0)
				loadFunc := func(ctx context.Context, key string) (any, error) {
					called = append(called, key)
					return "loaded_value", nil
				}
				return loadFunc, &called
			},
			testKey:            "key1",
			wantValue:          "cached_value",
			wantErr:            nil,
			wantLoadCalled:     false,
			wantBloomFilterHit: false,
		},
		{
			name: "缓存未命中_布隆过滤器返回false_直接返回不存在",
			setupCache: func() *MockCache {
				return &MockCache{store: make(map[string]any)}
			},
			setupBloomFilter: func() *InMemoryBloomFilter {
				config, _ := domainCache.NewBloomFilterConfig(100, 0.01)
				return NewInMemoryBloomFilter(config)
			},
			setupLoadFunc: func() (func(ctx context.Context, key string) (any, error), *[]string) {
				called := make([]string, 0)
				loadFunc := func(ctx context.Context, key string) (any, error) {
					called = append(called, key)
					return "loaded_value", nil
				}
				return loadFunc, &called
			},
			testKey:            "nonexistent_key",
			wantValue:          nil,
			wantErr:            ErrKeyNotFound,
			wantLoadCalled:     false,
			wantBloomFilterHit: false,
		},
		{
			name: "缓存未命中_布隆过滤器返回true_调用LoadFunc成功",
			setupCache: func() *MockCache {
				return &MockCache{store: make(map[string]any)}
			},
			setupBloomFilter: func() *InMemoryBloomFilter {
				config, _ := domainCache.NewBloomFilterConfig(100, 0.01)
				return NewInMemoryBloomFilter(config)
			},
			setupLoadFunc: func() (func(ctx context.Context, key string) (any, error), *[]string) {
				called := make([]string, 0)
				loadFunc := func(ctx context.Context, key string) (any, error) {
					called = append(called, key)
					return "loaded_value", nil
				}
				return loadFunc, &called
			},
			preAddToBloom:      []string{"key1"},
			testKey:            "key1",
			wantValue:          "loaded_value",
			wantErr:            nil,
			wantLoadCalled:     true,
			wantBloomFilterHit: true,
		},
		{
			name: "缓存未命中_布隆过滤器返回true_LoadFunc失败",
			setupCache: func() *MockCache {
				return &MockCache{store: make(map[string]any)}
			},
			setupBloomFilter: func() *InMemoryBloomFilter {
				config, _ := domainCache.NewBloomFilterConfig(100, 0.01)
				return NewInMemoryBloomFilter(config)
			},
			setupLoadFunc: func() (func(ctx context.Context, key string) (any, error), *[]string) {
				called := make([]string, 0)
				loadFunc := func(ctx context.Context, key string) (any, error) {
					called = append(called, key)
					return nil, fmt.Errorf("load failed")
				}
				return loadFunc, &called
			},
			preAddToBloom:      []string{"key1"},
			testKey:            "key1",
			wantValue:          nil,
			wantErr:            fmt.Errorf("load failed"),
			wantLoadCalled:     true,
			wantBloomFilterHit: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockCache := tt.setupCache()
			bloomFilter := tt.setupBloomFilter()
			loadFunc, loadCalled := tt.setupLoadFunc()

			// 预先添加键到布隆过滤器
			for _, key := range tt.preAddToBloom {
				_ = bloomFilter.Add(context.Background(), key)
			}

			bfc := NewBloomFilterCache(BloomFilterCacheConfig{
				Repository:     mockCache,
				BloomFilter:    bloomFilter,
				LoadFunc:       loadFunc,
				Expiration:     time.Minute,
				AutoAddToBloom: true,
			})

			val, err := bfc.Get(context.Background(), tt.testKey)

			// 检查返回值
			if tt.wantErr != nil {
				assert.Error(t, err)
				if err != nil {
					assert.Contains(t, err.Error(), tt.wantErr.Error())
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantValue, val)
			}

			// 检查LoadFunc是否被调用
			if tt.wantLoadCalled {
				assert.Contains(t, *loadCalled, tt.testKey)
			} else {
				assert.NotContains(t, *loadCalled, tt.testKey)
			}
		})
	}
}
