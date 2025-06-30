package lock

import (
	"context"
	"errors"
	"fmt"
	"iter"
	"time"
)

var (
	// ErrFailedToPreemptLock 抢锁失败错误
	ErrFailedToPreemptLock = errors.New("抢锁失败")
	// ErrLockNotHold 未持有锁错误
	ErrLockNotHold = errors.New("你没有持有锁")
	// ErrLockExpired 锁已过期错误
	ErrLockExpired = errors.New("锁已过期")
	// ErrInvalidLockKey 无效的锁键错误
	ErrInvalidLockKey = errors.New("无效的锁键")
	// ErrInvalidExpiration 无效的过期时间错误
	ErrInvalidExpiration = errors.New("无效的过期时间")
)

// DistributedLock 分布式锁领域接口
// 定义分布式锁的核心操作，支持锁的获取、释放、续约等功能
type DistributedLock interface {
	// Lock 获取锁
	// ctx: 上下文，用于控制超时和取消
	// key: 锁的键
	// expiration: 锁的过期时间
	// timeout: 获取锁的超时时间
	// retryStrategy: 重试策略
	// 返回: 锁实例和错误信息
	Lock(ctx context.Context, key string, expiration time.Duration, timeout time.Duration, retryStrategy RetryStrategy) (Lock, error)
	
	// TryLock 尝试获取锁（不重试）
	// ctx: 上下文
	// key: 锁的键
	// expiration: 锁的过期时间
	// 返回: 锁实例和错误信息
	TryLock(ctx context.Context, key string, expiration time.Duration) (Lock, error)
	
	// SingleflightLock 使用singleflight优化的获取锁
	// 本地goroutine先竞争，胜利者再去抢全局锁
	// ctx: 上下文
	// key: 锁的键
	// expiration: 锁的过期时间
	// timeout: 获取锁的超时时间
	// retryStrategy: 重试策略
	// 返回: 锁实例和错误信息
	SingleflightLock(ctx context.Context, key string, expiration time.Duration, timeout time.Duration, retryStrategy RetryStrategy) (Lock, error)
}

// Lock 锁实例接口
// 代表一个已获取的锁，提供锁的管理操作
type Lock interface {
	// Key 获取锁的键
	Key() string
	
	// Value 获取锁的值（UUID）
	Value() string
	
	// Expiration 获取锁的过期时间
	Expiration() time.Duration
	
	// CreatedAt 获取锁的创建时间
	CreatedAt() time.Time
	
	// IsExpired 检查锁是否已过期
	// now: 当前时间
	// 返回: 是否已过期
	IsExpired(now time.Time) bool
	
	// Refresh 手动续约锁
	// ctx: 上下文
	// 返回: 操作错误
	Refresh(ctx context.Context) error
	
	// AutoRefresh 自动续约锁
	// interval: 续约间隔
	// timeout: 每次续约的超时时间
	// 返回: 操作错误
	AutoRefresh(interval time.Duration, timeout time.Duration) error
	
	// Unlock 释放锁
	// ctx: 上下文
	// 返回: 操作错误
	Unlock(ctx context.Context) error
	
	// IsValid 检查锁是否仍然有效
	// ctx: 上下文
	// 返回: 是否有效和错误信息
	IsValid(ctx context.Context) (bool, error)
}

// RetryStrategy 重试策略接口
// 定义锁获取失败时的重试行为
type RetryStrategy interface {
	// Iterator 返回重试间隔的迭代器
	// 使用Go 1.23+的迭代器特性
	// 返回: 重试间隔的迭代器
	Iterator() iter.Seq[time.Duration]
}

// LockKey 锁键值对象
// 封装锁键的业务规则和验证逻辑
type LockKey struct {
	value string
}

// NewLockKey 创建新的锁键
// key: 键值字符串
// 返回: LockKey实例和错误信息
func NewLockKey(key string) (LockKey, error) {
	if err := validateLockKey(key); err != nil {
		return LockKey{}, fmt.Errorf("%w: %s", ErrInvalidLockKey, err.Error())
	}
	return LockKey{value: key}, nil
}

// String 返回锁键的字符串表示
func (k LockKey) String() string {
	return k.value
}

// IsEmpty 检查锁键是否为空
func (k LockKey) IsEmpty() bool {
	return k.value == ""
}

// Equals 比较两个锁键是否相等
func (k LockKey) Equals(other LockKey) bool {
	return k.value == other.value
}

// validateLockKey 验证锁键的有效性
func validateLockKey(key string) error {
	if key == "" {
		return errors.New("锁键不能为空")
	}
	if len(key) > 200 {
		return errors.New("锁键长度不能超过200个字符")
	}
	return nil
}

// LockValue 锁值值对象
// 封装锁值（UUID）的业务规则
type LockValue struct {
	value string
}

// NewLockValue 创建新的锁值
// value: 值字符串（通常是UUID）
// 返回: LockValue实例和错误信息
func NewLockValue(value string) (LockValue, error) {
	if value == "" {
		return LockValue{}, errors.New("锁值不能为空")
	}
	return LockValue{value: value}, nil
}

// String 返回锁值的字符串表示
func (v LockValue) String() string {
	return v.value
}

// Equals 比较两个锁值是否相等
func (v LockValue) Equals(other LockValue) bool {
	return v.value == other.value
}

// LockExpiration 锁过期时间值对象
// 封装锁过期时间的业务规则和计算逻辑
type LockExpiration struct {
	duration time.Duration
}

// NewLockExpiration 创建新的锁过期时间
// duration: 过期时间间隔
// 返回: LockExpiration实例和错误信息
func NewLockExpiration(duration time.Duration) (LockExpiration, error) {
	if duration <= 0 {
		return LockExpiration{}, fmt.Errorf("%w: 过期时间必须大于0", ErrInvalidExpiration)
	}
	if duration > 24*time.Hour {
		return LockExpiration{}, fmt.Errorf("%w: 过期时间不能超过24小时", ErrInvalidExpiration)
	}
	return LockExpiration{duration: duration}, nil
}

// Duration 获取过期时间间隔
func (e LockExpiration) Duration() time.Duration {
	return e.duration
}

// ExpiresAt 计算过期时间点
// from: 起始时间
// 返回: 过期时间点
func (e LockExpiration) ExpiresAt(from time.Time) time.Time {
	return from.Add(e.duration)
}

// IsExpired 检查是否已过期
// createdAt: 创建时间
// now: 当前时间
// 返回: 是否已过期
func (e LockExpiration) IsExpired(createdAt, now time.Time) bool {
	return now.After(e.ExpiresAt(createdAt))
}

// RemainingTime 计算剩余时间
// createdAt: 创建时间
// now: 当前时间
// 返回: 剩余时间
func (e LockExpiration) RemainingTime(createdAt, now time.Time) time.Duration {
	expiresAt := e.ExpiresAt(createdAt)
	if now.After(expiresAt) {
		return 0
	}
	return expiresAt.Sub(now)
}

// LockStats 锁统计信息值对象
// 封装锁的统计数据
type LockStats struct {
	totalLocks    int64
	activeLocks   int64
	failedLocks   int64
	expiredLocks  int64
	refreshCount  int64
	unlockCount   int64
}

// NewLockStats 创建新的锁统计信息
func NewLockStats() LockStats {
	return LockStats{}
}

// TotalLocks 获取总锁数量
func (s LockStats) TotalLocks() int64 {
	return s.totalLocks
}

// ActiveLocks 获取活跃锁数量
func (s LockStats) ActiveLocks() int64 {
	return s.activeLocks
}

// FailedLocks 获取失败锁数量
func (s LockStats) FailedLocks() int64 {
	return s.failedLocks
}

// ExpiredLocks 获取过期锁数量
func (s LockStats) ExpiredLocks() int64 {
	return s.expiredLocks
}

// RefreshCount 获取续约次数
func (s LockStats) RefreshCount() int64 {
	return s.refreshCount
}

// UnlockCount 获取解锁次数
func (s LockStats) UnlockCount() int64 {
	return s.unlockCount
}

// SuccessRate 计算成功率
func (s LockStats) SuccessRate() float64 {
	if s.totalLocks == 0 {
		return 0
	}
	return float64(s.totalLocks-s.failedLocks) / float64(s.totalLocks)
}

// IncrementTotalLocks 增加总锁数量
func (s LockStats) IncrementTotalLocks() LockStats {
	s.totalLocks++
	return s
}

// IncrementActiveLocks 增加活跃锁数量
func (s LockStats) IncrementActiveLocks() LockStats {
	s.activeLocks++
	return s
}

// DecrementActiveLocks 减少活跃锁数量
func (s LockStats) DecrementActiveLocks() LockStats {
	s.activeLocks--
	return s
}

// IncrementFailedLocks 增加失败锁数量
func (s LockStats) IncrementFailedLocks() LockStats {
	s.failedLocks++
	return s
}

// IncrementExpiredLocks 增加过期锁数量
func (s LockStats) IncrementExpiredLocks() LockStats {
	s.expiredLocks++
	return s
}

// IncrementRefreshCount 增加续约次数
func (s LockStats) IncrementRefreshCount() LockStats {
	s.refreshCount++
	return s
}

// IncrementUnlockCount 增加解锁次数
func (s LockStats) IncrementUnlockCount() LockStats {
	s.unlockCount++
	return s
}
