package lock

import (
	"context"
	"iter"
	"sync"
	"time"

	"github.com/google/uuid"
	"golang.org/x/sync/singleflight"

	domainLock "github.com/justinwongcn/hamster/internal/domain/lock"
)

// MemoryDistributedLock 基于内存的分布式锁实现
// 使用内存存储锁信息，支持锁的获取、释放、续约等功能
// 线程安全，支持并发访问
type MemoryDistributedLock struct {
	locks map[string]*memoryLock // 锁存储
	mu    sync.RWMutex           // 读写锁保护
	g     singleflight.Group     // singleflight优化
	stats domainLock.LockStats   // 统计信息
}

// memoryLock 内存锁实例
type memoryLock struct {
	key        string
	value      string
	expiration time.Duration
	createdAt  time.Time
	unlockChan chan struct{}
	client     *MemoryDistributedLock
}

// NewMemoryDistributedLock 创建新的内存分布式锁
// 返回: MemoryDistributedLock实例
func NewMemoryDistributedLock() *MemoryDistributedLock {
	return &MemoryDistributedLock{
		locks: make(map[string]*memoryLock),
		stats: domainLock.NewLockStats(),
	}
}

// TryLock 尝试获取锁（不重试）
// ctx: 上下文
// key: 锁的键
// expiration: 锁的过期时间
// 返回: 锁实例和错误信息
func (mdl *MemoryDistributedLock) TryLock(ctx context.Context, key string, expiration time.Duration) (domainLock.Lock, error) {
	// 验证输入
	lockKey, err := domainLock.NewLockKey(key)
	if err != nil {
		mdl.mu.Lock()
		mdl.stats = mdl.stats.IncrementFailedLocks()
		mdl.mu.Unlock()
		return nil, err
	}

	lockExpiration, err := domainLock.NewLockExpiration(expiration)
	if err != nil {
		mdl.mu.Lock()
		mdl.stats = mdl.stats.IncrementFailedLocks()
		mdl.mu.Unlock()
		return nil, err
	}

	mdl.mu.Lock()
	defer mdl.mu.Unlock()

	// 检查是否已存在锁
	if existingLock, exists := mdl.locks[key]; exists {
		// 检查锁是否已过期
		existingExpiration, _ := domainLock.NewLockExpiration(existingLock.expiration)
		if !existingExpiration.IsExpired(existingLock.createdAt, time.Now()) {
			mdl.stats = mdl.stats.IncrementFailedLocks()
			return nil, domainLock.ErrFailedToPreemptLock
		}
		// 锁已过期，清理旧锁
		delete(mdl.locks, key)
		mdl.stats = mdl.stats.IncrementExpiredLocks().DecrementActiveLocks()
	}

	// 创建新锁
	value := uuid.New().String()
	lock := &memoryLock{
		key:        lockKey.String(),
		value:      value,
		expiration: lockExpiration.Duration(),
		createdAt:  time.Now(),
		unlockChan: make(chan struct{}, 1),
		client:     mdl,
	}

	mdl.locks[key] = lock
	mdl.stats = mdl.stats.IncrementTotalLocks().IncrementActiveLocks()

	return lock, nil
}

// Lock 获取锁（支持重试）
// ctx: 上下文，用于控制超时和取消
// key: 锁的键
// expiration: 锁的过期时间
// timeout: 获取锁的超时时间
// retryStrategy: 重试策略
// 返回: 锁实例和错误信息
func (mdl *MemoryDistributedLock) Lock(ctx context.Context, key string, expiration time.Duration, timeout time.Duration, retryStrategy domainLock.RetryStrategy) (domainLock.Lock, error) {
	// 创建带超时的上下文
	lockCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// 首次尝试
	lock, err := mdl.TryLock(lockCtx, key, expiration)
	if err == nil {
		return lock, nil
	}

	// 如果不是抢锁失败，直接返回错误
	if err != domainLock.ErrFailedToPreemptLock {
		return nil, err
	}

	// 使用重试策略重试
	for interval := range retryStrategy.Iterator() {
		select {
		case <-lockCtx.Done():
			return nil, lockCtx.Err()
		case <-time.After(interval):
			lock, err := mdl.TryLock(lockCtx, key, expiration)
			if err == nil {
				return lock, nil
			}
			if err != domainLock.ErrFailedToPreemptLock {
				return nil, err
			}
		}
	}

	return nil, domainLock.ErrFailedToPreemptLock
}

// SingleflightLock 使用singleflight优化的获取锁
// 本地goroutine先竞争，胜利者再去抢全局锁
// ctx: 上下文
// key: 锁的键
// expiration: 锁的过期时间
// timeout: 获取锁的超时时间
// retryStrategy: 重试策略
// 返回: 锁实例和错误信息
func (mdl *MemoryDistributedLock) SingleflightLock(ctx context.Context, key string, expiration time.Duration, timeout time.Duration, retryStrategy domainLock.RetryStrategy) (domainLock.Lock, error) {
	// 使用singleflight确保同一时间只有一个goroutine去获取锁
	result, err, _ := mdl.g.Do(key, func() (interface{}, error) {
		return mdl.Lock(ctx, key, expiration, timeout, retryStrategy)
	})

	if err != nil {
		return nil, err
	}

	return result.(domainLock.Lock), nil
}

// GetStats 获取锁统计信息
// 返回: 锁统计信息
func (mdl *MemoryDistributedLock) GetStats() domainLock.LockStats {
	mdl.mu.RLock()
	defer mdl.mu.RUnlock()
	return mdl.stats
}

// CleanExpiredLocks 清理过期锁
// 返回: 清理的锁数量
func (mdl *MemoryDistributedLock) CleanExpiredLocks() int {
	mdl.mu.Lock()
	defer mdl.mu.Unlock()

	now := time.Now()
	expiredKeys := make([]string, 0)

	for key, lock := range mdl.locks {
		lockExpiration, _ := domainLock.NewLockExpiration(lock.expiration)
		if lockExpiration.IsExpired(lock.createdAt, now) {
			expiredKeys = append(expiredKeys, key)
		}
	}

	for _, key := range expiredKeys {
		delete(mdl.locks, key)
		mdl.stats = mdl.stats.IncrementExpiredLocks().DecrementActiveLocks()
	}

	return len(expiredKeys)
}

// memoryLock 实现 domainLock.Lock 接口

// Key 获取锁的键
func (ml *memoryLock) Key() string {
	return ml.key
}

// Value 获取锁的值（UUID）
func (ml *memoryLock) Value() string {
	return ml.value
}

// Expiration 获取锁的过期时间
func (ml *memoryLock) Expiration() time.Duration {
	return ml.expiration
}

// CreatedAt 获取锁的创建时间
func (ml *memoryLock) CreatedAt() time.Time {
	return ml.createdAt
}

// IsExpired 检查锁是否已过期
// now: 当前时间
// 返回: 是否已过期
func (ml *memoryLock) IsExpired(now time.Time) bool {
	lockExpiration, _ := domainLock.NewLockExpiration(ml.expiration)
	return lockExpiration.IsExpired(ml.createdAt, now)
}

// Refresh 手动续约锁
// ctx: 上下文
// 返回: 操作错误
func (ml *memoryLock) Refresh(ctx context.Context) error {
	ml.client.mu.Lock()
	defer ml.client.mu.Unlock()

	// 检查锁是否仍然存在且属于当前实例
	existingLock, exists := ml.client.locks[ml.key]
	if !exists || existingLock.value != ml.value {
		return domainLock.ErrLockNotHold
	}

	// 更新创建时间以续约
	existingLock.createdAt = time.Now()
	ml.createdAt = existingLock.createdAt
	ml.client.stats = ml.client.stats.IncrementRefreshCount()

	return nil
}

// AutoRefresh 自动续约锁
// interval: 续约间隔
// timeout: 每次续约的超时时间
// 返回: 操作错误
func (ml *memoryLock) AutoRefresh(interval time.Duration, timeout time.Duration) error {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			ctx, cancel := context.WithTimeout(context.Background(), timeout)
			err := ml.Refresh(ctx)
			cancel()

			if err != nil {
				return err
			}
		case <-ml.unlockChan:
			return nil
		}
	}
}

// Unlock 释放锁
// ctx: 上下文
// 返回: 操作错误
func (ml *memoryLock) Unlock(ctx context.Context) error {
	ml.client.mu.Lock()
	defer ml.client.mu.Unlock()

	// 检查锁是否仍然存在且属于当前实例
	existingLock, exists := ml.client.locks[ml.key]
	if !exists || existingLock.value != ml.value {
		return domainLock.ErrLockNotHold
	}

	// 删除锁
	delete(ml.client.locks, ml.key)
	ml.client.stats = ml.client.stats.IncrementUnlockCount().DecrementActiveLocks()

	// 通知自动续约停止
	select {
	case ml.unlockChan <- struct{}{}:
	default:
		// 没有人在等待，忽略
	}

	return nil
}

// IsValid 检查锁是否仍然有效
// ctx: 上下文
// 返回: 是否有效和错误信息
func (ml *memoryLock) IsValid(ctx context.Context) (bool, error) {
	ml.client.mu.RLock()
	defer ml.client.mu.RUnlock()

	// 检查锁是否仍然存在且属于当前实例
	existingLock, exists := ml.client.locks[ml.key]
	if !exists || existingLock.value != ml.value {
		return false, nil
	}

	// 检查锁是否已过期
	if ml.IsExpired(time.Now()) {
		return false, nil
	}

	return true, nil
}

// FixedIntervalRetryStrategy 固定间隔重试策略
// 使用固定的时间间隔进行重试
type FixedIntervalRetryStrategy struct {
	interval time.Duration
	maxRetry int
}

// NewFixedIntervalRetryStrategy 创建固定间隔重试策略
// interval: 重试间隔
// maxRetry: 最大重试次数
// 返回: FixedIntervalRetryStrategy实例
func NewFixedIntervalRetryStrategy(interval time.Duration, maxRetry int) *FixedIntervalRetryStrategy {
	return &FixedIntervalRetryStrategy{
		interval: interval,
		maxRetry: maxRetry,
	}
}

// Iterator 返回重试间隔的迭代器
// 使用Go 1.23+的迭代器特性
// 返回: 重试间隔的迭代器
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
// 使用指数增长的时间间隔进行重试
type ExponentialBackoffRetryStrategy struct {
	initialInterval time.Duration
	multiplier      float64
	maxRetry        int
}

// NewExponentialBackoffRetryStrategy 创建指数退避重试策略
// initialInterval: 初始重试间隔
// multiplier: 倍数因子
// maxRetry: 最大重试次数
// 返回: ExponentialBackoffRetryStrategy实例
func NewExponentialBackoffRetryStrategy(initialInterval time.Duration, multiplier float64, maxRetry int) *ExponentialBackoffRetryStrategy {
	return &ExponentialBackoffRetryStrategy{
		initialInterval: initialInterval,
		multiplier:      multiplier,
		maxRetry:        maxRetry,
	}
}

// Iterator 返回重试间隔的迭代器
// 使用Go 1.23+的迭代器特性
// 返回: 重试间隔的迭代器
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
// 使用线性增长的时间间隔进行重试
type LinearBackoffRetryStrategy struct {
	initialInterval time.Duration
	increment       time.Duration
	maxRetry        int
}

// NewLinearBackoffRetryStrategy 创建线性退避重试策略
// initialInterval: 初始重试间隔
// increment: 每次增加的间隔
// maxRetry: 最大重试次数
// 返回: LinearBackoffRetryStrategy实例
func NewLinearBackoffRetryStrategy(initialInterval time.Duration, increment time.Duration, maxRetry int) *LinearBackoffRetryStrategy {
	return &LinearBackoffRetryStrategy{
		initialInterval: initialInterval,
		increment:       increment,
		maxRetry:        maxRetry,
	}
}

// Iterator 返回重试间隔的迭代器
// 使用Go 1.23+的迭代器特性
// 返回: 重试间隔的迭代器
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
