package cache

import (
	"context"
	"fmt"
	"time"

	"golang.org/x/sync/singleflight"

	domainCache "github.com/justinwongcn/hamster/internal/domain/cache"
)

// BloomFilterCache 带布隆过滤器的读透缓存
// 使用布隆过滤器预先过滤不存在的键，减少对底层数据源的无效查询
// 当布隆过滤器返回false时，直接返回键不存在，避免调用LoadFunc
// 当布隆过滤器返回true时，继续执行正常的读透缓存逻辑
type BloomFilterCache struct {
	domainCache.Repository                    // 嵌入领域仓储接口
	bloomFilter            domainCache.BloomFilter // 布隆过滤器
	loadFunc               func(ctx context.Context, key string) (any, error) // 数据加载函数
	expiration             time.Duration      // 缓存过期时间
	autoAddToBloom         bool               // 是否自动将成功加载的键添加到布隆过滤器
	logFunc                func(format string, args ...any) // 日志函数
	g                      singleflight.Group // 防止缓存击穿
}

// BloomFilterCacheConfig 布隆过滤器缓存配置
type BloomFilterCacheConfig struct {
	Repository     domainCache.Repository                                    // 底层缓存仓储
	BloomFilter    domainCache.BloomFilter                                   // 布隆过滤器
	LoadFunc       func(ctx context.Context, key string) (any, error)       // 数据加载函数
	Expiration     time.Duration                                             // 缓存过期时间
	AutoAddToBloom bool                                                      // 是否自动将成功加载的键添加到布隆过滤器
	LogFunc        func(format string, args ...any)                         // 日志函数
}

// NewBloomFilterCache 创建带布隆过滤器的读透缓存
// config: 布隆过滤器缓存配置
// 返回: BloomFilterCache实例
func NewBloomFilterCache(config BloomFilterCacheConfig) *BloomFilterCache {
	return &BloomFilterCache{
		Repository:     config.Repository,
		bloomFilter:    config.BloomFilter,
		loadFunc:       config.LoadFunc,
		expiration:     config.Expiration,
		autoAddToBloom: config.AutoAddToBloom,
		logFunc:        config.LogFunc,
		g:              singleflight.Group{},
	}
}

// NewBloomFilterCacheSimple 创建简单的布隆过滤器缓存（兼容参考代码）
// repository: 底层缓存仓储
// bloomFilter: 布隆过滤器
// loadFunc: 数据加载函数
// 返回: BloomFilterCache实例
func NewBloomFilterCacheSimple(
	repository domainCache.Repository,
	bloomFilter domainCache.BloomFilter,
	loadFunc func(ctx context.Context, key string) (any, error),
) *BloomFilterCache {
	return NewBloomFilterCache(BloomFilterCacheConfig{
		Repository:     repository,
		BloomFilter:    bloomFilter,
		LoadFunc:       loadFunc,
		Expiration:     time.Hour, // 默认1小时过期
		AutoAddToBloom: true,      // 默认自动添加到布隆过滤器
	})
}

// Get 实现带布隆过滤器的读透缓存获取逻辑
// 1. 先从缓存获取
// 2. 如果缓存未命中，检查布隆过滤器
// 3. 如果布隆过滤器返回false，直接返回键不存在
// 4. 如果布隆过滤器返回true，调用LoadFunc加载数据
// 5. 如果加载成功且autoAddToBloom为true，将键添加到布隆过滤器
func (bfc *BloomFilterCache) Get(ctx context.Context, key string) (any, error) {
	// 先尝试从缓存获取
	cachedVal, err := bfc.Repository.Get(ctx, key)
	if err == nil {
		return cachedVal, nil
	}
	
	// 如果不是键不存在错误，直接返回错误
	if err != ErrKeyNotFound {
		return nil, err
	}
	
	// 缓存未命中，检查布隆过滤器
	if !bfc.bloomFilter.HasKey(ctx, key) {
		// 布隆过滤器返回false，键一定不存在
		if bfc.logFunc != nil {
			bfc.logFunc("布隆过滤器过滤键: %s", key)
		}
		return nil, ErrKeyNotFound
	}
	
	// 布隆过滤器返回true，可能存在，使用single flight防止缓存击穿
	loadedVal, loadErr, _ := bfc.g.Do(key, func() (any, error) {
		return bfc.handleCacheMiss(ctx, key)
	})
	
	return loadedVal, loadErr
}

// handleCacheMiss 处理缓存未命中的情况
// ctx: 上下文
// key: 缓存键
// 返回: 加载的值和错误
func (bfc *BloomFilterCache) handleCacheMiss(ctx context.Context, key string) (any, error) {
	if bfc.loadFunc == nil {
		return nil, fmt.Errorf("LoadFunc未设置")
	}
	
	// 调用LoadFunc加载数据
	newVal, err := bfc.loadFunc(ctx, key)
	if err != nil {
		return nil, err
	}
	
	// 加载成功，更新缓存
	if setErr := bfc.Repository.Set(ctx, key, newVal, bfc.expiration); setErr != nil {
		if bfc.logFunc != nil {
			bfc.logFunc("刷新缓存失败，键：%s，错误：%v", key, setErr)
		}
		// 即使缓存设置失败，也返回加载的值，但包装错误
		return newVal, fmt.Errorf("%w, 原因：%s", ErrFailedToRefreshCache, setErr.Error())
	}
	
	// 如果启用自动添加到布隆过滤器，将键添加到布隆过滤器
	if bfc.autoAddToBloom {
		if addErr := bfc.bloomFilter.Add(ctx, key); addErr != nil && bfc.logFunc != nil {
			bfc.logFunc("添加键到布隆过滤器失败，键：%s，错误：%v", key, addErr)
		}
	}
	
	return newVal, nil
}

// Set 重写Set方法，同时更新布隆过滤器
// ctx: 上下文
// key: 缓存键
// val: 缓存值
// expiration: 过期时间
// 返回: 操作错误
func (bfc *BloomFilterCache) Set(ctx context.Context, key string, val any, expiration time.Duration) error {
	// 先设置缓存
	err := bfc.Repository.Set(ctx, key, val, expiration)
	if err != nil {
		return err
	}
	
	// 如果启用自动添加到布隆过滤器，将键添加到布隆过滤器
	if bfc.autoAddToBloom {
		if addErr := bfc.bloomFilter.Add(ctx, key); addErr != nil && bfc.logFunc != nil {
			bfc.logFunc("添加键到布隆过滤器失败，键：%s，错误：%v", key, addErr)
		}
	}
	
	return nil
}

// Delete 重写Delete方法
// 注意：布隆过滤器不支持删除操作，所以只删除缓存
// ctx: 上下文
// key: 缓存键
// 返回: 操作错误
func (bfc *BloomFilterCache) Delete(ctx context.Context, key string) error {
	// 只删除缓存，布隆过滤器中的键无法删除
	// 这可能导致布隆过滤器中存在已删除键的记录，但这是布隆过滤器的固有特性
	return bfc.Repository.Delete(ctx, key)
}

// LoadAndDelete 重写LoadAndDelete方法
// ctx: 上下文
// key: 缓存键
// 返回: 缓存值和操作错误
func (bfc *BloomFilterCache) LoadAndDelete(ctx context.Context, key string) (any, error) {
	// 只操作缓存，布隆过滤器中的键无法删除
	return bfc.Repository.LoadAndDelete(ctx, key)
}

// SetLogFunc 设置日志函数
// logFunc: 日志函数
func (bfc *BloomFilterCache) SetLogFunc(logFunc func(format string, args ...any)) {
	bfc.logFunc = logFunc
}

// GetBloomFilterStats 获取布隆过滤器统计信息
// ctx: 上下文
// 返回: 布隆过滤器统计信息和错误
func (bfc *BloomFilterCache) GetBloomFilterStats(ctx context.Context) (domainCache.BloomFilterStats, error) {
	return bfc.bloomFilter.Stats(ctx)
}

// ClearBloomFilter 清空布隆过滤器
// ctx: 上下文
// 返回: 操作错误
func (bfc *BloomFilterCache) ClearBloomFilter(ctx context.Context) error {
	return bfc.bloomFilter.Clear(ctx)
}

// AddKeyToBloomFilter 手动添加键到布隆过滤器
// ctx: 上下文
// key: 要添加的键
// 返回: 操作错误
func (bfc *BloomFilterCache) AddKeyToBloomFilter(ctx context.Context, key string) error {
	return bfc.bloomFilter.Add(ctx, key)
}

// HasKeyInBloomFilter 检查键是否在布隆过滤器中
// ctx: 上下文
// key: 要检查的键
// 返回: 是否可能存在
func (bfc *BloomFilterCache) HasKeyInBloomFilter(ctx context.Context, key string) bool {
	return bfc.bloomFilter.HasKey(ctx, key)
}

// SetAutoAddToBloom 设置是否自动添加键到布隆过滤器
// autoAdd: 是否自动添加
func (bfc *BloomFilterCache) SetAutoAddToBloom(autoAdd bool) {
	bfc.autoAddToBloom = autoAdd
}

// IsAutoAddToBloomEnabled 检查是否启用自动添加到布隆过滤器
// 返回: 是否启用
func (bfc *BloomFilterCache) IsAutoAddToBloomEnabled() bool {
	return bfc.autoAddToBloom
}

// GetLoadFunc 获取数据加载函数（用于测试）
// 返回: 数据加载函数
func (bfc *BloomFilterCache) GetLoadFunc() func(ctx context.Context, key string) (any, error) {
	return bfc.loadFunc
}

// SetLoadFunc 设置数据加载函数
// loadFunc: 数据加载函数
func (bfc *BloomFilterCache) SetLoadFunc(loadFunc func(ctx context.Context, key string) (any, error)) {
	bfc.loadFunc = loadFunc
}

// GetExpiration 获取缓存过期时间
// 返回: 过期时间
func (bfc *BloomFilterCache) GetExpiration() time.Duration {
	return bfc.expiration
}

// SetExpiration 设置缓存过期时间
// expiration: 过期时间
func (bfc *BloomFilterCache) SetExpiration(expiration time.Duration) {
	bfc.expiration = expiration
}
