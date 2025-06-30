package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/justinwongcn/hamster/internal/domain/cache"
)

// ApplicationService 缓存应用服务
// 协调领域服务和基础设施，实现具体的业务用例
type ApplicationService struct {
	repository    cache.Repository
	cacheService  *cache.CacheService
	writeBackRepo cache.WriteBackRepository
}

// NewApplicationService 创建缓存应用服务
// repository: 缓存仓储
// cacheService: 缓存领域服务
// writeBackRepo: 写回缓存仓储（可选）
func NewApplicationService(repository cache.Repository, cacheService *cache.CacheService, writeBackRepo cache.WriteBackRepository) *ApplicationService {
	return &ApplicationService{
		repository:    repository,
		cacheService:  cacheService,
		writeBackRepo: writeBackRepo,
	}
}

// CacheItemCommand 缓存项命令
type CacheItemCommand struct {
	Key        string
	Value      any
	Expiration time.Duration
}

// CacheItemQuery 缓存项查询
type CacheItemQuery struct {
	Key string
}

// CacheItemResult 缓存项结果
type CacheItemResult struct {
	Key       string
	Value     any
	Found     bool
	CreatedAt time.Time
	IsDirty   bool
}

// CacheStatsResult 缓存统计结果
type CacheStatsResult struct {
	Hits      int64
	Misses    int64
	HitRate   float64
	Size      int64
	DirtyKeys []string
}

// SetCacheItem 设置缓存项
// 用例：用户想要缓存一个数据项
func (s *ApplicationService) SetCacheItem(ctx context.Context, cmd CacheItemCommand) error {
	// 验证输入
	if err := s.validateCacheItemCommand(cmd); err != nil {
		return fmt.Errorf("验证缓存项命令失败: %w", err)
	}

	// 设置缓存
	err := s.repository.Set(ctx, cmd.Key, cmd.Value, cmd.Expiration)
	if err != nil {
		return fmt.Errorf("设置缓存项失败: %w", err)
	}

	return nil
}

// GetCacheItem 获取缓存项
// 用例：用户想要获取一个缓存的数据项
func (s *ApplicationService) GetCacheItem(ctx context.Context, query CacheItemQuery) (*CacheItemResult, error) {
	// 验证输入
	if err := s.validateCacheItemQuery(query); err != nil {
		return nil, fmt.Errorf("验证缓存项查询失败: %w", err)
	}

	// 获取缓存
	value, err := s.repository.Get(ctx, query.Key)
	if err != nil {
		if err == cache.ErrKeyNotFound {
			return &CacheItemResult{
				Key:   query.Key,
				Found: false,
			}, nil
		}
		return nil, fmt.Errorf("获取缓存项失败: %w", err)
	}

	return &CacheItemResult{
		Key:       query.Key,
		Value:     value,
		Found:     true,
		CreatedAt: time.Now(), // 实际应该从缓存条目获取
		IsDirty:   false,      // 实际应该从缓存条目获取
	}, nil
}

// DeleteCacheItem 删除缓存项
// 用例：用户想要删除一个缓存的数据项
func (s *ApplicationService) DeleteCacheItem(ctx context.Context, query CacheItemQuery) error {
	// 验证输入
	if err := s.validateCacheItemQuery(query); err != nil {
		return fmt.Errorf("验证缓存项查询失败: %w", err)
	}

	// 删除缓存
	err := s.repository.Delete(ctx, query.Key)
	if err != nil {
		return fmt.Errorf("删除缓存项失败: %w", err)
	}

	return nil
}

// SetDirtyCacheItem 设置脏缓存项（仅写回模式）
// 用例：用户想要设置一个脏数据项，稍后批量写入持久化存储
func (s *ApplicationService) SetDirtyCacheItem(ctx context.Context, cmd CacheItemCommand) error {
	if s.writeBackRepo == nil {
		return fmt.Errorf("写回缓存仓储未配置")
	}

	// 验证输入
	if err := s.validateCacheItemCommand(cmd); err != nil {
		return fmt.Errorf("验证缓存项命令失败: %w", err)
	}

	// 设置脏缓存
	err := s.writeBackRepo.SetDirty(ctx, cmd.Key, cmd.Value, cmd.Expiration)
	if err != nil {
		return fmt.Errorf("设置脏缓存项失败: %w", err)
	}

	return nil
}

// FlushDirtyData 刷新脏数据
// 用例：用户想要将所有脏数据写入持久化存储
func (s *ApplicationService) FlushDirtyData(ctx context.Context, storer func(ctx context.Context, key string, val any) error) error {
	if s.writeBackRepo == nil {
		return fmt.Errorf("写回缓存仓储未配置")
	}

	if storer == nil {
		return fmt.Errorf("存储函数不能为空")
	}

	// 刷新脏数据
	err := s.writeBackRepo.Flush(ctx, storer)
	if err != nil {
		return fmt.Errorf("刷新脏数据失败: %w", err)
	}

	return nil
}

// GetCacheStats 获取缓存统计信息
// 用例：用户想要查看缓存的使用情况和性能指标
func (s *ApplicationService) GetCacheStats(ctx context.Context) (*CacheStatsResult, error) {
	// 这里应该从缓存实例获取统计信息
	// 由于当前的Repository接口没有提供统计信息，我们返回基本信息
	result := &CacheStatsResult{
		Hits:    0,
		Misses:  0,
		HitRate: 0.0,
		Size:    0,
	}

	// 如果是写回缓存，获取脏数据键
	if s.writeBackRepo != nil {
		// 这里需要扩展WriteBackRepository接口来获取脏数据键
		result.DirtyKeys = []string{} // 暂时返回空列表
	}

	return result, nil
}

// validateCacheItemCommand 验证缓存项命令
func (s *ApplicationService) validateCacheItemCommand(cmd CacheItemCommand) error {
	if err := s.cacheService.ValidateKey(cmd.Key); err != nil {
		return fmt.Errorf("无效的缓存键: %w", err)
	}

	if err := s.cacheService.ValidateExpiration(cmd.Expiration); err != nil {
		return fmt.Errorf("无效的过期时间: %w", err)
	}

	if cmd.Value == nil {
		return fmt.Errorf("缓存值不能为空")
	}

	return nil
}

// validateCacheItemQuery 验证缓存项查询
func (s *ApplicationService) validateCacheItemQuery(query CacheItemQuery) error {
	if err := s.cacheService.ValidateKey(query.Key); err != nil {
		return fmt.Errorf("无效的缓存键: %w", err)
	}

	return nil
}

// ReadThroughApplicationService 读透缓存应用服务
// 专门处理读透缓存的业务用例
type ReadThroughApplicationService struct {
	*ApplicationService
	readThroughRepo cache.ReadThroughRepository
}

// NewReadThroughApplicationService 创建读透缓存应用服务
func NewReadThroughApplicationService(
	repository cache.Repository,
	cacheService *cache.CacheService,
	readThroughRepo cache.ReadThroughRepository,
) *ReadThroughApplicationService {
	return &ReadThroughApplicationService{
		ApplicationService: NewApplicationService(repository, cacheService, nil),
		readThroughRepo:    readThroughRepo,
	}
}

// GetWithLoader 使用加载器获取缓存项
// 用例：用户想要获取数据，如果缓存未命中则从数据源加载
func (s *ReadThroughApplicationService) GetWithLoader(
	ctx context.Context,
	query CacheItemQuery,
	loader func(ctx context.Context, key string) (any, error),
	expiration time.Duration,
) (*CacheItemResult, error) {
	// 验证输入
	if err := s.validateCacheItemQuery(query); err != nil {
		return nil, fmt.Errorf("验证缓存项查询失败: %w", err)
	}

	if loader == nil {
		return nil, fmt.Errorf("加载器函数不能为空")
	}

	// 使用读透缓存获取数据
	value, err := s.readThroughRepo.GetWithLoader(ctx, query.Key, loader, expiration)
	if err != nil {
		return nil, fmt.Errorf("读透缓存获取失败: %w", err)
	}

	return &CacheItemResult{
		Key:       query.Key,
		Value:     value,
		Found:     true,
		CreatedAt: time.Now(),
		IsDirty:   false,
	}, nil
}

// WriteThroughApplicationService 写透缓存应用服务
// 专门处理写透缓存的业务用例
type WriteThroughApplicationService struct {
	*ApplicationService
	writeThroughRepo cache.WriteThroughRepository
}

// NewWriteThroughApplicationService 创建写透缓存应用服务
func NewWriteThroughApplicationService(
	repository cache.Repository,
	cacheService *cache.CacheService,
	writeThroughRepo cache.WriteThroughRepository,
) *WriteThroughApplicationService {
	return &WriteThroughApplicationService{
		ApplicationService: NewApplicationService(repository, cacheService, nil),
		writeThroughRepo:   writeThroughRepo,
	}
}

// SetWithStore 使用存储器设置缓存项
// 用例：用户想要设置数据，同时写入缓存和持久化存储
func (s *WriteThroughApplicationService) SetWithStore(
	ctx context.Context,
	cmd CacheItemCommand,
	storer func(ctx context.Context, key string, val any) error,
) error {
	// 验证输入
	if err := s.validateCacheItemCommand(cmd); err != nil {
		return fmt.Errorf("验证缓存项命令失败: %w", err)
	}

	if storer == nil {
		return fmt.Errorf("存储器函数不能为空")
	}

	// 使用写透缓存设置数据
	err := s.writeThroughRepo.SetWithStore(ctx, cmd.Key, cmd.Value, storer, cmd.Expiration)
	if err != nil {
		return fmt.Errorf("写透缓存设置失败: %w", err)
	}

	return nil
}

// BloomFilterApplicationService 布隆过滤器应用服务
// 专门处理布隆过滤器相关的业务用例
type BloomFilterApplicationService struct {
	*ApplicationService
	bloomFilterCache interface {
		cache.Repository
		GetBloomFilterStats(ctx context.Context) (cache.BloomFilterStats, error)
		ClearBloomFilter(ctx context.Context) error
		AddKeyToBloomFilter(ctx context.Context, key string) error
		HasKeyInBloomFilter(ctx context.Context, key string) bool
		SetAutoAddToBloom(autoAdd bool)
		IsAutoAddToBloomEnabled() bool
	}
}

// NewBloomFilterApplicationService 创建布隆过滤器应用服务
func NewBloomFilterApplicationService(
	repository cache.Repository,
	cacheService *cache.CacheService,
	bloomFilterCache interface {
		cache.Repository
		GetBloomFilterStats(ctx context.Context) (cache.BloomFilterStats, error)
		ClearBloomFilter(ctx context.Context) error
		AddKeyToBloomFilter(ctx context.Context, key string) error
		HasKeyInBloomFilter(ctx context.Context, key string) bool
		SetAutoAddToBloom(autoAdd bool)
		IsAutoAddToBloomEnabled() bool
	},
) *BloomFilterApplicationService {
	return &BloomFilterApplicationService{
		ApplicationService: NewApplicationService(repository, cacheService, nil),
		bloomFilterCache:   bloomFilterCache,
	}
}

// BloomFilterStatsQuery 布隆过滤器统计查询
type BloomFilterStatsQuery struct{}

// BloomFilterStatsResult 布隆过滤器统计结果
type BloomFilterStatsResult struct {
	ExpectedElements      uint64  `json:"expected_elements"`
	FalsePositiveRate     float64 `json:"false_positive_rate"`
	BitArraySize          uint64  `json:"bit_array_size"`
	HashFunctions         uint64  `json:"hash_functions"`
	AddedElements         uint64  `json:"added_elements"`
	SetBits               uint64  `json:"set_bits"`
	EstimatedFPR          float64 `json:"estimated_fpr"`
	MemoryUsage           uint64  `json:"memory_usage"`
	LoadFactor            float64 `json:"load_factor"`
	IsOverloaded          bool    `json:"is_overloaded"`
	EfficiencyRatio       float64 `json:"efficiency_ratio"`
	AutoAddToBloomEnabled bool    `json:"auto_add_to_bloom_enabled"`
}

// BloomFilterKeyCommand 布隆过滤器键命令
type BloomFilterKeyCommand struct {
	Key string `json:"key"`
}

// BloomFilterKeyQuery 布隆过滤器键查询
type BloomFilterKeyQuery struct {
	Key string `json:"key"`
}

// BloomFilterKeyResult 布隆过滤器键结果
type BloomFilterKeyResult struct {
	Key           string `json:"key"`
	MightExist    bool   `json:"might_exist"`
	InBloomFilter bool   `json:"in_bloom_filter"`
}

// GetBloomFilterStats 获取布隆过滤器统计信息
// 用例：用户想要查看布隆过滤器的使用情况和性能指标
func (s *BloomFilterApplicationService) GetBloomFilterStats(ctx context.Context, query BloomFilterStatsQuery) (*BloomFilterStatsResult, error) {
	stats, err := s.bloomFilterCache.GetBloomFilterStats(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取布隆过滤器统计信息失败: %w", err)
	}

	return &BloomFilterStatsResult{
		ExpectedElements:      stats.Config().ExpectedElements(),
		FalsePositiveRate:     stats.Config().FalsePositiveRate(),
		BitArraySize:          stats.Config().BitArraySize(),
		HashFunctions:         stats.Config().HashFunctions(),
		AddedElements:         stats.AddedElements(),
		SetBits:               stats.SetBits(),
		EstimatedFPR:          stats.EstimatedFalsePositiveRate(),
		MemoryUsage:           stats.MemoryUsage(),
		LoadFactor:            stats.LoadFactor(),
		IsOverloaded:          stats.IsOverloaded(),
		EfficiencyRatio:       stats.EfficiencyRatio(),
		AutoAddToBloomEnabled: s.bloomFilterCache.IsAutoAddToBloomEnabled(),
	}, nil
}

// AddKeyToBloomFilter 添加键到布隆过滤器
// 用例：用户想要手动添加一个键到布隆过滤器
func (s *BloomFilterApplicationService) AddKeyToBloomFilter(ctx context.Context, cmd BloomFilterKeyCommand) error {
	// 验证输入
	if err := s.validateBloomFilterKeyCommand(cmd); err != nil {
		return fmt.Errorf("验证布隆过滤器键命令失败: %w", err)
	}

	// 添加键到布隆过滤器
	err := s.bloomFilterCache.AddKeyToBloomFilter(ctx, cmd.Key)
	if err != nil {
		return fmt.Errorf("添加键到布隆过滤器失败: %w", err)
	}

	return nil
}

// CheckKeyInBloomFilter 检查键是否在布隆过滤器中
// 用例：用户想要检查一个键是否可能存在于布隆过滤器中
func (s *BloomFilterApplicationService) CheckKeyInBloomFilter(ctx context.Context, query BloomFilterKeyQuery) (*BloomFilterKeyResult, error) {
	// 验证输入
	if err := s.validateBloomFilterKeyQuery(query); err != nil {
		return nil, fmt.Errorf("验证布隆过滤器键查询失败: %w", err)
	}

	// 检查键是否在布隆过滤器中
	inBloomFilter := s.bloomFilterCache.HasKeyInBloomFilter(ctx, query.Key)

	return &BloomFilterKeyResult{
		Key:           query.Key,
		MightExist:    inBloomFilter,
		InBloomFilter: inBloomFilter,
	}, nil
}

// ClearBloomFilter 清空布隆过滤器
// 用例：用户想要清空布隆过滤器，重新开始
func (s *BloomFilterApplicationService) ClearBloomFilter(ctx context.Context) error {
	err := s.bloomFilterCache.ClearBloomFilter(ctx)
	if err != nil {
		return fmt.Errorf("清空布隆过滤器失败: %w", err)
	}

	return nil
}

// SetAutoAddToBloom 设置是否自动添加键到布隆过滤器
// 用例：用户想要控制是否自动将成功加载的键添加到布隆过滤器
func (s *BloomFilterApplicationService) SetAutoAddToBloom(autoAdd bool) {
	s.bloomFilterCache.SetAutoAddToBloom(autoAdd)
}

// IsAutoAddToBloomEnabled 检查是否启用自动添加到布隆过滤器
// 用例：用户想要查看当前的自动添加设置
func (s *BloomFilterApplicationService) IsAutoAddToBloomEnabled() bool {
	return s.bloomFilterCache.IsAutoAddToBloomEnabled()
}

// GetWithBloomFilter 使用布隆过滤器优化的获取操作
// 用例：用户想要获取数据，利用布隆过滤器减少无效查询
func (s *BloomFilterApplicationService) GetWithBloomFilter(ctx context.Context, query CacheItemQuery) (*CacheItemResult, error) {
	// 验证输入
	if err := s.validateCacheItemQuery(query); err != nil {
		return nil, fmt.Errorf("验证缓存项查询失败: %w", err)
	}

	// 先检查布隆过滤器
	if !s.bloomFilterCache.HasKeyInBloomFilter(ctx, query.Key) {
		// 布隆过滤器返回false，键一定不存在
		return &CacheItemResult{
			Key:   query.Key,
			Found: false,
		}, nil
	}

	// 布隆过滤器返回true，继续正常的缓存查询
	return s.GetCacheItem(ctx, query)
}

// validateBloomFilterKeyCommand 验证布隆过滤器键命令
func (s *BloomFilterApplicationService) validateBloomFilterKeyCommand(cmd BloomFilterKeyCommand) error {
	if err := s.cacheService.ValidateKey(cmd.Key); err != nil {
		return fmt.Errorf("无效的布隆过滤器键: %w", err)
	}

	return nil
}

// validateBloomFilterKeyQuery 验证布隆过滤器键查询
func (s *BloomFilterApplicationService) validateBloomFilterKeyQuery(query BloomFilterKeyQuery) error {
	if err := s.cacheService.ValidateKey(query.Key); err != nil {
		return fmt.Errorf("无效的布隆过滤器键: %w", err)
	}

	return nil
}
