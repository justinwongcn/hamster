package cache

import (
	"context"
	"fmt"
	"time"

	appCache "github.com/justinwongcn/hamster/internal/application/cache"
	domainCache "github.com/justinwongcn/hamster/internal/domain/cache"
	infraCache "github.com/justinwongcn/hamster/internal/infrastructure/cache"
)

// Config 缓存配置
type Config struct {
	// MaxMemory 最大内存使用量（字节）
	MaxMemory int64

	// DefaultExpiration 默认过期时间
	DefaultExpiration time.Duration

	// CleanupInterval 清理间隔
	CleanupInterval time.Duration

	// EvictionPolicy 淘汰策略 ("lru", "lfu", "fifo")
	EvictionPolicy string

	// EnableBloomFilter 是否启用布隆过滤器
	EnableBloomFilter bool

	// BloomFilterFalsePositiveRate 布隆过滤器假阳性率
	BloomFilterFalsePositiveRate float64
}

// DefaultConfig 返回默认缓存配置
func DefaultConfig() *Config {
	return &Config{
		MaxMemory:                    1024 * 1024, // 1MB
		DefaultExpiration:            time.Hour,
		CleanupInterval:              10 * time.Minute,
		EvictionPolicy:               "lru",
		EnableBloomFilter:            false,
		BloomFilterFalsePositiveRate: 0.01,
	}
}

// Option 缓存选项函数
type Option func(*Config)

// WithMaxMemory 设置最大内存
func WithMaxMemory(maxMemory int64) Option {
	return func(c *Config) {
		c.MaxMemory = maxMemory
	}
}

// WithDefaultExpiration 设置默认过期时间
func WithDefaultExpiration(expiration time.Duration) Option {
	return func(c *Config) {
		c.DefaultExpiration = expiration
	}
}

// WithCleanupInterval 设置清理间隔
func WithCleanupInterval(interval time.Duration) Option {
	return func(c *Config) {
		c.CleanupInterval = interval
	}
}

// WithEvictionPolicy 设置淘汰策略
func WithEvictionPolicy(policy string) Option {
	return func(c *Config) {
		c.EvictionPolicy = policy
	}
}

// WithBloomFilter 启用布隆过滤器
func WithBloomFilter(enable bool, falsePositiveRate float64) Option {
	return func(c *Config) {
		c.EnableBloomFilter = enable
		c.BloomFilterFalsePositiveRate = falsePositiveRate
	}
}

// Service 缓存服务公共接口
type Service struct {
	appService *appCache.ApplicationService
}

// NewService 创建缓存服务
// 使用默认配置创建缓存服务实例
func NewService(options ...Option) (*Service, error) {
	config := DefaultConfig()
	for _, option := range options {
		option(config)
	}

	return NewServiceWithConfig(config)
}

// NewServiceWithConfig 使用配置创建缓存服务
func NewServiceWithConfig(config *Config) (*Service, error) {
	if config == nil {
		return nil, fmt.Errorf("配置不能为空")
	}

	// 创建基础设施层
	// 使用 BuildInMapCache 作为 Repository 实现
	repository := infraCache.NewBuildInMapCache(config.CleanupInterval)

	// 创建领域服务
	var evictionStrategy domainCache.EvictionStrategy
	switch config.EvictionPolicy {
	case "lru":
		evictionStrategy = domainCache.NewLRUEvictionStrategy()
	case "fifo":
		evictionStrategy = domainCache.NewFIFOEvictionStrategy()
	default:
		evictionStrategy = domainCache.NewLRUEvictionStrategy()
	}

	cacheService := domainCache.NewCacheService(evictionStrategy)

	// 创建应用服务
	appService := appCache.NewApplicationService(repository, cacheService, nil)

	return &Service{
		appService: appService,
	}, nil
}

// Set 设置缓存值
func (s *Service) Set(ctx context.Context, key string, value any, expiration time.Duration) error {
	cmd := appCache.CacheItemCommand{
		Key:        key,
		Value:      value,
		Expiration: expiration,
	}

	return s.appService.SetCacheItem(ctx, cmd)
}

// Get 获取缓存值
func (s *Service) Get(ctx context.Context, key string) (any, error) {
	query := appCache.CacheItemQuery{Key: key}

	result, err := s.appService.GetCacheItem(ctx, query)
	if err != nil {
		return nil, err
	}

	if !result.Found {
		return nil, fmt.Errorf("键 %s 未找到", key)
	}

	return result.Value, nil
}

// Delete 删除缓存值
func (s *Service) Delete(ctx context.Context, key string) error {
	query := appCache.CacheItemQuery{Key: key}
	return s.appService.DeleteCacheItem(ctx, query)
}

// LoadAndDelete 获取并删除缓存值
func (s *Service) LoadAndDelete(ctx context.Context, key string) (any, error) {
	query := appCache.CacheItemQuery{Key: key}

	// 先获取值
	result, err := s.appService.GetCacheItem(ctx, query)
	if err != nil {
		return nil, err
	}

	if !result.Found {
		return nil, fmt.Errorf("键 %s 未找到", key)
	}

	// 然后删除
	err = s.appService.DeleteCacheItem(ctx, query)
	if err != nil {
		return nil, err
	}

	return result.Value, nil
}

// OnEvicted 设置淘汰回调函数
func (s *Service) OnEvicted(fn func(key string, val any)) {
	// 这里需要通过底层仓储设置回调
	// 暂时留空，需要扩展基础设施层的接口
}

// Stats 获取缓存统计信息
func (s *Service) Stats(ctx context.Context) (*Stats, error) {
	result, err := s.appService.GetCacheStats(ctx)
	if err != nil {
		return nil, err
	}

	return &Stats{
		HitCount:    result.Hits,
		MissCount:   result.Misses,
		HitRate:     result.HitRate,
		ItemCount:   result.Size,
		MemoryUsage: 0, // 暂时不支持内存使用统计
	}, nil
}

// Clear 清空缓存
func (s *Service) Clear(ctx context.Context) error {
	// 暂时不支持清空缓存，需要扩展应用服务接口
	return fmt.Errorf("清空缓存功能暂未实现")
}

// Stats 缓存统计信息
type Stats struct {
	HitCount    int64   `json:"hit_count"`
	MissCount   int64   `json:"miss_count"`
	HitRate     float64 `json:"hit_rate"`
	ItemCount   int64   `json:"item_count"`
	MemoryUsage int64   `json:"memory_usage"`
}

// ReadThroughService 读透缓存服务
type ReadThroughService struct {
	service *Service
}

// NewReadThroughService 创建读透缓存服务
func NewReadThroughService(options ...Option) (*ReadThroughService, error) {
	service, err := NewService(options...)
	if err != nil {
		return nil, err
	}

	return &ReadThroughService{
		service: service,
	}, nil
}

// GetWithLoader 使用加载器获取缓存项
func (s *ReadThroughService) GetWithLoader(
	ctx context.Context,
	key string,
	loader func(ctx context.Context, key string) (any, error),
	expiration time.Duration,
) (any, error) {
	// 先尝试从缓存获取
	value, err := s.service.Get(ctx, key)
	if err == nil {
		return value, nil
	}

	// 缓存未命中，使用加载器加载数据
	loadedValue, err := loader(ctx, key)
	if err != nil {
		return nil, err
	}

	// 将加载的数据存入缓存
	err = s.service.Set(ctx, key, loadedValue, expiration)
	if err != nil {
		// 即使缓存设置失败，也返回加载的数据
		return loadedValue, nil
	}

	return loadedValue, nil
}
