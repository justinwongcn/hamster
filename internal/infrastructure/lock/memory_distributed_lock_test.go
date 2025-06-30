package lock

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	domainLock "github.com/justinwongcn/hamster/internal/domain/lock"
)

// TestLockValueObjects 测试锁相关的值对象
func TestLockValueObjects(t *testing.T) {
	t.Run("LockKey", func(t *testing.T) {
		tests := []struct {
			name    string
			keyStr  string
			wantErr bool
		}{
			{
				name:    "正常键",
				keyStr:  "test_lock_key",
				wantErr: false,
			},
			{
				name:    "空键",
				keyStr:  "",
				wantErr: true,
			},
			{
				name:    "长键",
				keyStr:  string(make([]byte, 201)), // 超过200字符
				wantErr: true,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				key, err := domainLock.NewLockKey(tt.keyStr)
				if tt.wantErr {
					assert.Error(t, err)
					return
				}
				require.NoError(t, err)
				assert.Equal(t, tt.keyStr, key.String())
			})
		}
	})

	t.Run("LockExpiration", func(t *testing.T) {
		tests := []struct {
			name     string
			duration time.Duration
			wantErr  bool
		}{
			{
				name:     "正常过期时间",
				duration: time.Minute,
				wantErr:  false,
			},
			{
				name:     "零过期时间",
				duration: 0,
				wantErr:  true,
			},
			{
				name:     "负过期时间",
				duration: -time.Second,
				wantErr:  true,
			},
			{
				name:     "过长过期时间",
				duration: 25 * time.Hour,
				wantErr:  true,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				exp, err := domainLock.NewLockExpiration(tt.duration)
				if tt.wantErr {
					assert.Error(t, err)
					return
				}
				require.NoError(t, err)
				assert.Equal(t, tt.duration, exp.Duration())
			})
		}
	})
}

// TestMemoryDistributedLock_TryLock 测试尝试获取锁
func TestMemoryDistributedLock_TryLock(t *testing.T) {
	tests := []struct {
		name       string
		setupLocks func(*MemoryDistributedLock)
		key        string
		expiration time.Duration
		wantErr    error
		wantLock   bool
	}{
		{
			name:       "成功获取锁",
			setupLocks: func(mdl *MemoryDistributedLock) {},
			key:        "test_key",
			expiration: time.Minute,
			wantErr:    nil,
			wantLock:   true,
		},
		{
			name: "锁已被占用",
			setupLocks: func(mdl *MemoryDistributedLock) {
				// 预先占用锁
				_, _ = mdl.TryLock(context.Background(), "test_key", time.Minute)
			},
			key:        "test_key",
			expiration: time.Minute,
			wantErr:    domainLock.ErrFailedToPreemptLock,
			wantLock:   false,
		},
		{
			name:       "无效的键",
			setupLocks: func(mdl *MemoryDistributedLock) {},
			key:        "",
			expiration: time.Minute,
			wantErr:    domainLock.ErrInvalidLockKey,
			wantLock:   false,
		},
		{
			name:       "无效的过期时间",
			setupLocks: func(mdl *MemoryDistributedLock) {},
			key:        "test_key",
			expiration: 0,
			wantErr:    domainLock.ErrInvalidExpiration,
			wantLock:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mdl := NewMemoryDistributedLock()
			tt.setupLocks(mdl)

			lock, err := mdl.TryLock(context.Background(), tt.key, tt.expiration)

			if tt.wantErr != nil {
				assert.Error(t, err)
				assert.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, lock)
			} else {
				assert.NoError(t, err)
				if tt.wantLock {
					assert.NotNil(t, lock)
					assert.Equal(t, tt.key, lock.Key())
					assert.Equal(t, tt.expiration, lock.Expiration())
				}
			}
		})
	}
}

// TestMemoryDistributedLock_LockExpiration 测试锁过期
func TestMemoryDistributedLock_LockExpiration(t *testing.T) {
	mdl := NewMemoryDistributedLock()

	// 获取一个短期锁
	lock, err := mdl.TryLock(context.Background(), "test_key", 100*time.Millisecond)
	require.NoError(t, err)
	require.NotNil(t, lock)

	// 立即检查锁是否有效
	assert.False(t, lock.IsExpired(time.Now()))

	// 等待锁过期
	time.Sleep(150 * time.Millisecond)

	// 检查锁是否已过期
	assert.True(t, lock.IsExpired(time.Now()))

	// 现在应该能够获取同一个键的锁
	newLock, err := mdl.TryLock(context.Background(), "test_key", time.Minute)
	assert.NoError(t, err)
	assert.NotNil(t, newLock)
}

// TestMemoryDistributedLock_Refresh 测试锁续约
func TestMemoryDistributedLock_Refresh(t *testing.T) {
	mdl := NewMemoryDistributedLock()

	lock, err := mdl.TryLock(context.Background(), "test_key", time.Minute)
	require.NoError(t, err)
	require.NotNil(t, lock)

	// 续约锁
	err = lock.Refresh(context.Background())
	assert.NoError(t, err)

	// 解锁后尝试续约应该失败
	err = lock.Unlock(context.Background())
	require.NoError(t, err)

	err = lock.Refresh(context.Background())
	assert.ErrorIs(t, err, domainLock.ErrLockNotHold)
}

// TestMemoryDistributedLock_Unlock 测试解锁
func TestMemoryDistributedLock_Unlock(t *testing.T) {
	mdl := NewMemoryDistributedLock()

	lock, err := mdl.TryLock(context.Background(), "test_key", time.Minute)
	require.NoError(t, err)
	require.NotNil(t, lock)

	// 解锁
	err = lock.Unlock(context.Background())
	assert.NoError(t, err)

	// 重复解锁应该失败
	err = lock.Unlock(context.Background())
	assert.ErrorIs(t, err, domainLock.ErrLockNotHold)

	// 解锁后应该能够重新获取锁
	newLock, err := mdl.TryLock(context.Background(), "test_key", time.Minute)
	assert.NoError(t, err)
	assert.NotNil(t, newLock)
}

// TestMemoryDistributedLock_ConcurrentAccess 测试并发访问
func TestMemoryDistributedLock_ConcurrentAccess(t *testing.T) {
	mdl := NewMemoryDistributedLock()

	const numGoroutines = 10
	const lockKey = "concurrent_test_key"

	var successCount int64
	var wg sync.WaitGroup
	var mu sync.Mutex
	successLocks := make([]domainLock.Lock, 0)

	// 启动多个goroutine尝试获取同一个锁
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			lock, err := mdl.TryLock(context.Background(), lockKey, time.Second)
			if err == nil {
				mu.Lock()
				successCount++
				successLocks = append(successLocks, lock)
				mu.Unlock()

				// 持有锁一段时间
				time.Sleep(10 * time.Millisecond)

				// 释放锁
				_ = lock.Unlock(context.Background())
			}
		}(i)
	}

	wg.Wait()

	// 只应该有一个goroutine成功获取锁
	assert.Equal(t, int64(1), successCount)
	assert.Len(t, successLocks, 1)
}

// TestMemoryDistributedLock_AutoRefresh 测试自动续约
func TestMemoryDistributedLock_AutoRefresh(t *testing.T) {
	mdl := NewMemoryDistributedLock()

	lock, err := mdl.TryLock(context.Background(), "test_key", 200*time.Millisecond)
	require.NoError(t, err)
	require.NotNil(t, lock)

	// 启动自动续约
	refreshDone := make(chan error, 1)
	go func() {
		err := lock.AutoRefresh(50*time.Millisecond, 100*time.Millisecond)
		refreshDone <- err
	}()

	// 等待一段时间，确保锁被续约
	time.Sleep(300 * time.Millisecond)

	// 锁应该仍然有效（因为自动续约）
	valid, err := lock.IsValid(context.Background())
	assert.NoError(t, err)
	assert.True(t, valid)

	// 解锁应该停止自动续约
	err = lock.Unlock(context.Background())
	assert.NoError(t, err)

	// 等待自动续约结束
	select {
	case err := <-refreshDone:
		assert.NoError(t, err)
	case <-time.After(time.Second):
		t.Fatal("自动续约没有及时结束")
	}
}

// TestRetryStrategy 测试重试策略
func TestRetryStrategy(t *testing.T) {
	t.Run("FixedIntervalRetryStrategy", func(t *testing.T) {
		strategy := NewFixedIntervalRetryStrategy(100*time.Millisecond, 3)

		intervals := make([]time.Duration, 0)
		for interval := range strategy.Iterator() {
			intervals = append(intervals, interval)
		}

		assert.Len(t, intervals, 3)
		for _, interval := range intervals {
			assert.Equal(t, 100*time.Millisecond, interval)
		}
	})

	t.Run("ExponentialBackoffRetryStrategy", func(t *testing.T) {
		strategy := NewExponentialBackoffRetryStrategy(100*time.Millisecond, 2.0, 3)

		intervals := make([]time.Duration, 0)
		for interval := range strategy.Iterator() {
			intervals = append(intervals, interval)
		}

		assert.Len(t, intervals, 3)
		assert.Equal(t, 100*time.Millisecond, intervals[0])
		assert.Equal(t, 200*time.Millisecond, intervals[1])
		assert.Equal(t, 400*time.Millisecond, intervals[2])
	})
}

// TestMemoryDistributedLock_SingleflightLock 测试singleflight优化
func TestMemoryDistributedLock_SingleflightLock(t *testing.T) {
	mdl := NewMemoryDistributedLock()

	const numGoroutines = 5
	const lockKey = "singleflight_test_key"

	var wg sync.WaitGroup
	var mu sync.Mutex
	lockValues := make(map[string]int)
	errors := make([]error, 0)

	retryStrategy := NewFixedIntervalRetryStrategy(10*time.Millisecond, 2)

	// 启动多个goroutine尝试获取同一个锁
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			lock, err := mdl.SingleflightLock(context.Background(), lockKey, time.Second, time.Second, retryStrategy)

			mu.Lock()
			if err != nil {
				errors = append(errors, err)
			} else {
				lockValues[lock.Value()]++
			}
			mu.Unlock()
		}(i)
	}

	wg.Wait()

	// 验证singleflight效果：所有成功的goroutine应该获得相同的锁值
	if len(errors) > 0 {
		t.Logf("有 %d 个错误: %v", len(errors), errors)
	}

	// 至少应该有一些成功的锁获取
	assert.Greater(t, len(lockValues), 0, "应该有成功的锁获取")
	assert.LessOrEqual(t, len(lockValues), 1, "所有成功的goroutine应该获得相同的锁值")

	for lockValue, count := range lockValues {
		assert.Greater(t, count, 0, "应该有goroutine获得锁")
		assert.LessOrEqual(t, count, numGoroutines, "获得锁的goroutine数量不应超过总数")

		// 验证锁确实存在于内存中
		mdl.mu.RLock()
		memLock, exists := mdl.locks[lockKey]
		mdl.mu.RUnlock()

		assert.True(t, exists, "锁应该存在于内存中")
		assert.Equal(t, lockValue, memLock.value, "锁值应该匹配")

		// 清理：解锁
		err := memLock.Unlock(context.Background())
		assert.NoError(t, err)
	}
}

// TestMemoryDistributedLock_LockWithRetry 测试带重试的锁获取
func TestMemoryDistributedLock_LockWithRetry(t *testing.T) {
	mdl := NewMemoryDistributedLock()

	// 先获取一个短期锁
	firstLock, err := mdl.TryLock(context.Background(), "retry_test_key", 100*time.Millisecond)
	require.NoError(t, err)
	require.NotNil(t, firstLock)

	// 使用重试策略尝试获取同一个锁
	retryStrategy := NewFixedIntervalRetryStrategy(50*time.Millisecond, 3)

	start := time.Now()
	secondLock, err := mdl.Lock(context.Background(), "retry_test_key", time.Minute, time.Second, retryStrategy)
	elapsed := time.Since(start)

	// 应该在第一个锁过期后成功获取
	assert.NoError(t, err)
	assert.NotNil(t, secondLock)
	assert.Greater(t, elapsed, 100*time.Millisecond) // 至少等待了第一个锁过期
	assert.Less(t, elapsed, 500*time.Millisecond)    // 但不应该等待太久

	// 清理
	_ = secondLock.Unlock(context.Background())
}
