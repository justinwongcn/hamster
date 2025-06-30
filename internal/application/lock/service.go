package lock

import (
	"context"
	"fmt"
	"iter"
	"time"

	domainLock "github.com/justinwongcn/hamster/internal/domain/lock"
)

// DistributedLockApplicationService 分布式锁应用服务
// 协调领域服务和基础设施，实现具体的分布式锁业务用例
type DistributedLockApplicationService struct {
	distributedLock domainLock.DistributedLock
}

// NewDistributedLockApplicationService 创建分布式锁应用服务
// distributedLock: 分布式锁实现
func NewDistributedLockApplicationService(distributedLock domainLock.DistributedLock) *DistributedLockApplicationService {
	return &DistributedLockApplicationService{
		distributedLock: distributedLock,
	}
}

// LockCommand 加锁命令
type LockCommand struct {
	Key        string        `json:"key"`
	Expiration time.Duration `json:"expiration"`
	Timeout    time.Duration `json:"timeout"`
	RetryType  string        `json:"retry_type"` // "fixed", "exponential", "linear"
	RetryCount int           `json:"retry_count"`
	RetryBase  time.Duration `json:"retry_base"`
}

// LockQuery 锁查询
type LockQuery struct {
	Key string `json:"key"`
}

// LockResult 锁结果
type LockResult struct {
	Key       string    `json:"key"`
	Value     string    `json:"value"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
	IsValid   bool      `json:"is_valid"`
}

// RefreshCommand 续约命令
type RefreshCommand struct {
	Key string `json:"key"`
}

// AutoRefreshCommand 自动续约命令
type AutoRefreshCommand struct {
	Key      string        `json:"key"`
	Interval time.Duration `json:"interval"`
	Timeout  time.Duration `json:"timeout"`
}

// UnlockCommand 解锁命令
type UnlockCommand struct {
	Key string `json:"key"`
}

// TryLock 尝试获取锁（不重试）
// 用例：用户想要快速尝试获取锁，如果失败立即返回
func (s *DistributedLockApplicationService) TryLock(ctx context.Context, cmd LockCommand) (*LockResult, error) {
	// 验证输入
	if err := s.validateLockCommand(cmd); err != nil {
		return nil, fmt.Errorf("验证加锁命令失败: %w", err)
	}

	// 尝试获取锁
	lock, err := s.distributedLock.TryLock(ctx, cmd.Key, cmd.Expiration)
	if err != nil {
		return nil, fmt.Errorf("尝试获取锁失败: %w", err)
	}

	return s.buildLockResult(ctx, lock), nil
}

// Lock 获取锁（支持重试）
// 用例：用户想要获取锁，如果失败则按策略重试
func (s *DistributedLockApplicationService) Lock(ctx context.Context, cmd LockCommand) (*LockResult, error) {
	// 验证输入
	if err := s.validateLockCommand(cmd); err != nil {
		return nil, fmt.Errorf("验证加锁命令失败: %w", err)
	}

	// 创建重试策略
	retryStrategy, err := s.createRetryStrategy(cmd)
	if err != nil {
		return nil, fmt.Errorf("创建重试策略失败: %w", err)
	}

	// 获取锁
	lock, err := s.distributedLock.Lock(ctx, cmd.Key, cmd.Expiration, cmd.Timeout, retryStrategy)
	if err != nil {
		return nil, fmt.Errorf("获取锁失败: %w", err)
	}

	return s.buildLockResult(ctx, lock), nil
}

// SingleflightLock 使用singleflight优化的获取锁
// 用例：用户想要获取锁，本地goroutine先竞争，减少对分布式锁的压力
func (s *DistributedLockApplicationService) SingleflightLock(ctx context.Context, cmd LockCommand) (*LockResult, error) {
	// 验证输入
	if err := s.validateLockCommand(cmd); err != nil {
		return nil, fmt.Errorf("验证加锁命令失败: %w", err)
	}

	// 创建重试策略
	retryStrategy, err := s.createRetryStrategy(cmd)
	if err != nil {
		return nil, fmt.Errorf("创建重试策略失败: %w", err)
	}

	// 使用singleflight获取锁
	lock, err := s.distributedLock.SingleflightLock(ctx, cmd.Key, cmd.Expiration, cmd.Timeout, retryStrategy)
	if err != nil {
		return nil, fmt.Errorf("singleflight获取锁失败: %w", err)
	}

	return s.buildLockResult(ctx, lock), nil
}

// RefreshLock 手动续约锁
// 用例：用户想要延长锁的有效期
func (s *DistributedLockApplicationService) RefreshLock(ctx context.Context, cmd RefreshCommand, lock domainLock.Lock) error {
	// 验证输入
	if cmd.Key == "" {
		return fmt.Errorf("锁键不能为空")
	}

	if lock == nil {
		return fmt.Errorf("锁实例不能为空")
	}

	if lock.Key() != cmd.Key {
		return fmt.Errorf("锁键不匹配")
	}

	// 续约锁
	err := lock.Refresh(ctx)
	if err != nil {
		return fmt.Errorf("续约锁失败: %w", err)
	}

	return nil
}

// StartAutoRefresh 启动自动续约
// 用例：用户想要自动续约锁，避免锁过期
func (s *DistributedLockApplicationService) StartAutoRefresh(cmd AutoRefreshCommand, lock domainLock.Lock) error {
	// 验证输入
	if cmd.Key == "" {
		return fmt.Errorf("锁键不能为空")
	}

	if lock == nil {
		return fmt.Errorf("锁实例不能为空")
	}

	if lock.Key() != cmd.Key {
		return fmt.Errorf("锁键不匹配")
	}

	if cmd.Interval <= 0 {
		return fmt.Errorf("续约间隔必须大于0")
	}

	if cmd.Timeout <= 0 {
		return fmt.Errorf("续约超时时间必须大于0")
	}

	// 启动自动续约（异步）
	go func() {
		_ = lock.AutoRefresh(cmd.Interval, cmd.Timeout)
	}()

	return nil
}

// UnlockLock 释放锁
// 用例：用户想要释放持有的锁
func (s *DistributedLockApplicationService) UnlockLock(ctx context.Context, cmd UnlockCommand, lock domainLock.Lock) error {
	// 验证输入
	if cmd.Key == "" {
		return fmt.Errorf("锁键不能为空")
	}

	if lock == nil {
		return fmt.Errorf("锁实例不能为空")
	}

	if lock.Key() != cmd.Key {
		return fmt.Errorf("锁键不匹配")
	}

	// 释放锁
	err := lock.Unlock(ctx)
	if err != nil {
		return fmt.Errorf("释放锁失败: %w", err)
	}

	return nil
}

// CheckLockStatus 检查锁状态
// 用例：用户想要检查锁是否仍然有效
func (s *DistributedLockApplicationService) CheckLockStatus(ctx context.Context, query LockQuery, lock domainLock.Lock) (*LockResult, error) {
	// 验证输入
	if query.Key == "" {
		return nil, fmt.Errorf("锁键不能为空")
	}

	if lock == nil {
		return nil, fmt.Errorf("锁实例不能为空")
	}

	if lock.Key() != query.Key {
		return nil, fmt.Errorf("锁键不匹配")
	}

	return s.buildLockResult(ctx, lock), nil
}

// validateLockCommand 验证加锁命令
func (s *DistributedLockApplicationService) validateLockCommand(cmd LockCommand) error {
	if cmd.Key == "" {
		return fmt.Errorf("锁键不能为空")
	}

	if cmd.Expiration <= 0 {
		return fmt.Errorf("过期时间必须大于0")
	}

	if cmd.Timeout <= 0 {
		return fmt.Errorf("超时时间必须大于0")
	}

	if cmd.RetryCount < 0 {
		return fmt.Errorf("重试次数不能为负数")
	}

	if cmd.RetryBase <= 0 && cmd.RetryCount > 0 {
		return fmt.Errorf("重试基础时间必须大于0")
	}

	return nil
}

// createRetryStrategy 创建重试策略
func (s *DistributedLockApplicationService) createRetryStrategy(cmd LockCommand) (domainLock.RetryStrategy, error) {
	if cmd.RetryCount == 0 {
		// 不重试
		return &NoRetryStrategy{}, nil
	}

	switch cmd.RetryType {
	case "fixed", "":
		return &FixedIntervalRetryStrategy{
			interval: cmd.RetryBase,
			maxRetry: cmd.RetryCount,
		}, nil
	case "exponential":
		return &ExponentialBackoffRetryStrategy{
			initialInterval: cmd.RetryBase,
			multiplier:      2.0,
			maxRetry:        cmd.RetryCount,
		}, nil
	case "linear":
		return &LinearBackoffRetryStrategy{
			initialInterval: cmd.RetryBase,
			increment:       cmd.RetryBase,
			maxRetry:        cmd.RetryCount,
		}, nil
	default:
		return nil, fmt.Errorf("不支持的重试类型: %s", cmd.RetryType)
	}
}

// buildLockResult 构建锁结果
func (s *DistributedLockApplicationService) buildLockResult(ctx context.Context, lock domainLock.Lock) *LockResult {
	isValid, _ := lock.IsValid(ctx)

	return &LockResult{
		Key:       lock.Key(),
		Value:     lock.Value(),
		CreatedAt: lock.CreatedAt(),
		ExpiresAt: lock.CreatedAt().Add(lock.Expiration()),
		IsValid:   isValid,
	}
}

// NoRetryStrategy 不重试策略
type NoRetryStrategy struct{}

// Iterator 返回空的重试间隔迭代器
func (s *NoRetryStrategy) Iterator() iter.Seq[time.Duration] {
	return func(yield func(time.Duration) bool) {
		// 不产生任何重试间隔
	}
}

// FixedIntervalRetryStrategy 固定间隔重试策略
type FixedIntervalRetryStrategy struct {
	interval time.Duration
	maxRetry int
}

// Iterator 返回重试间隔的迭代器
func (s *FixedIntervalRetryStrategy) Iterator() iter.Seq[time.Duration] {
	return func(yield func(time.Duration) bool) {
		for i := 0; i < s.maxRetry; i++ {
			if !yield(s.interval) {
				return
			}
		}
	}
}

// ExponentialBackoffRetryStrategy 指数退避重试策略
type ExponentialBackoffRetryStrategy struct {
	initialInterval time.Duration
	multiplier      float64
	maxRetry        int
}

// Iterator 返回重试间隔的迭代器
func (s *ExponentialBackoffRetryStrategy) Iterator() iter.Seq[time.Duration] {
	return func(yield func(time.Duration) bool) {
		interval := s.initialInterval
		for i := 0; i < s.maxRetry; i++ {
			if !yield(interval) {
				return
			}
			interval = time.Duration(float64(interval) * s.multiplier)
		}
	}
}

// LinearBackoffRetryStrategy 线性退避重试策略
type LinearBackoffRetryStrategy struct {
	initialInterval time.Duration
	increment       time.Duration
	maxRetry        int
}

// Iterator 返回重试间隔的迭代器
func (s *LinearBackoffRetryStrategy) Iterator() iter.Seq[time.Duration] {
	return func(yield func(time.Duration) bool) {
		interval := s.initialInterval
		for i := 0; i < s.maxRetry; i++ {
			if !yield(interval) {
				return
			}
			interval += s.increment
		}
	}
}
