package cache

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

var (
	// ErrInvalidCacheKey 无效的缓存键错误
	ErrInvalidCacheKey = errors.New("无效的缓存键")
	// ErrInvalidExpiration 无效的过期时间错误
	ErrInvalidExpiration = errors.New("无效的过期时间")
	// ErrKeyNotFound 键未找到错误
	ErrKeyNotFound = errors.New("键未找到")
	// ErrFailedToRefreshCache 刷新缓存失败错误
	ErrFailedToRefreshCache = errors.New("刷新缓存失败")
)

// CacheKey 缓存键值对象
// 封装缓存键的业务规则和验证逻辑
type CacheKey struct {
	value string
}

// NewCacheKey 创建新的缓存键
// key: 键值字符串
// 返回: CacheKey实例和错误信息
func NewCacheKey(key string) (CacheKey, error) {
	if err := validateKey(key); err != nil {
		return CacheKey{}, fmt.Errorf("%w: %s", ErrInvalidCacheKey, err.Error())
	}
	return CacheKey{value: key}, nil
}

// String 返回缓存键的字符串表示
func (k CacheKey) String() string {
	return k.value
}

// IsEmpty 检查缓存键是否为空
func (k CacheKey) IsEmpty() bool {
	return k.value == ""
}

// Equals 比较两个缓存键是否相等
func (k CacheKey) Equals(other CacheKey) bool {
	return k.value == other.value
}

// validateKey 验证缓存键的有效性
func validateKey(key string) error {
	if key == "" {
		return errors.New("缓存键不能为空")
	}
	if len(key) > 250 {
		return errors.New("缓存键长度不能超过250个字符")
	}
	if strings.Contains(key, "\n") || strings.Contains(key, "\r") {
		return errors.New("缓存键不能包含换行符")
	}
	return nil
}

// CacheValue 缓存值值对象
// 封装缓存值的业务规则和元数据
type CacheValue struct {
	data      any
	createdAt time.Time
	isDirty   bool
}

// NewCacheValue 创建新的缓存值
// data: 要缓存的数据
// 返回: CacheValue实例
func NewCacheValue(data any) CacheValue {
	return CacheValue{
		data:      data,
		createdAt: time.Now(),
		isDirty:   false,
	}
}

// NewDirtyCacheValue 创建新的脏缓存值
// data: 要缓存的数据
// 返回: 标记为脏的CacheValue实例
func NewDirtyCacheValue(data any) CacheValue {
	return CacheValue{
		data:      data,
		createdAt: time.Now(),
		isDirty:   true,
	}
}

// Data 获取缓存值的数据
func (v CacheValue) Data() any {
	return v.data
}

// CreatedAt 获取缓存值的创建时间
func (v CacheValue) CreatedAt() time.Time {
	return v.createdAt
}

// IsDirty 检查缓存值是否为脏数据
func (v CacheValue) IsDirty() bool {
	return v.isDirty
}

// MarkClean 标记缓存值为干净数据
func (v CacheValue) MarkClean() CacheValue {
	return CacheValue{
		data:      v.data,
		createdAt: v.createdAt,
		isDirty:   false,
	}
}

// MarkDirty 标记缓存值为脏数据
func (v CacheValue) MarkDirty() CacheValue {
	return CacheValue{
		data:      v.data,
		createdAt: v.createdAt,
		isDirty:   true,
	}
}

// Expiration 过期时间值对象
// 封装过期时间的业务规则和计算逻辑
type Expiration struct {
	duration time.Duration
}

// NewExpiration 创建新的过期时间
// duration: 过期时间间隔，0表示永不过期
// 返回: Expiration实例和错误信息
func NewExpiration(duration time.Duration) (Expiration, error) {
	if duration < 0 {
		return Expiration{}, fmt.Errorf("%w: 过期时间不能为负数", ErrInvalidExpiration)
	}
	return Expiration{duration: duration}, nil
}

// Duration 获取过期时间间隔
func (e Expiration) Duration() time.Duration {
	return e.duration
}

// IsNeverExpire 检查是否永不过期
func (e Expiration) IsNeverExpire() bool {
	return e.duration == 0
}

// ExpiresAt 计算过期时间点
// from: 起始时间
// 返回: 过期时间点
func (e Expiration) ExpiresAt(from time.Time) time.Time {
	if e.IsNeverExpire() {
		return time.Time{} // 零值表示永不过期
	}
	return from.Add(e.duration)
}

// IsExpired 检查是否已过期
// createdAt: 创建时间
// now: 当前时间
// 返回: 是否已过期
func (e Expiration) IsExpired(createdAt, now time.Time) bool {
	if e.IsNeverExpire() {
		return false
	}
	return now.After(e.ExpiresAt(createdAt))
}

// CacheStats 缓存统计值对象
// 封装缓存的统计信息
type CacheStats struct {
	hits        int64
	misses      int64
	sets        int64
	deletes     int64
	evictions   int64
	dirtyWrites int64
	flushes     int64
}

// NewCacheStats 创建新的缓存统计
func NewCacheStats() CacheStats {
	return CacheStats{}
}

// Hits 获取命中次数
func (s CacheStats) Hits() int64 {
	return s.hits
}

// Misses 获取未命中次数
func (s CacheStats) Misses() int64 {
	return s.misses
}

// Sets 获取设置次数
func (s CacheStats) Sets() int64 {
	return s.sets
}

// Deletes 获取删除次数
func (s CacheStats) Deletes() int64 {
	return s.deletes
}

// Evictions 获取淘汰次数
func (s CacheStats) Evictions() int64 {
	return s.evictions
}

// DirtyWrites 获取脏写次数
func (s CacheStats) DirtyWrites() int64 {
	return s.dirtyWrites
}

// Flushes 获取刷新次数
func (s CacheStats) Flushes() int64 {
	return s.flushes
}

// HitRate 计算命中率
func (s CacheStats) HitRate() float64 {
	total := s.hits + s.misses
	if total == 0 {
		return 0
	}
	return float64(s.hits) / float64(total)
}

// IncrementHits 增加命中次数
func (s CacheStats) IncrementHits() CacheStats {
	s.hits++
	return s
}

// IncrementMisses 增加未命中次数
func (s CacheStats) IncrementMisses() CacheStats {
	s.misses++
	return s
}

// IncrementSets 增加设置次数
func (s CacheStats) IncrementSets() CacheStats {
	s.sets++
	return s
}

// IncrementDeletes 增加删除次数
func (s CacheStats) IncrementDeletes() CacheStats {
	s.deletes++
	return s
}

// IncrementEvictions 增加淘汰次数
func (s CacheStats) IncrementEvictions() CacheStats {
	s.evictions++
	return s
}

// IncrementDirtyWrites 增加脏写次数
func (s CacheStats) IncrementDirtyWrites() CacheStats {
	s.dirtyWrites++
	return s
}

// IncrementFlushes 增加刷新次数
func (s CacheStats) IncrementFlushes() CacheStats {
	s.flushes++
	return s
}
