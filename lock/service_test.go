package lock

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()
	
	assert.Equal(t, 30*time.Second, config.DefaultExpiration)
	assert.Equal(t, 5*time.Second, config.DefaultTimeout)
	assert.Equal(t, RetryTypeExponential, config.DefaultRetryType)
	assert.Equal(t, 3, config.DefaultRetryCount)
	assert.Equal(t, 100*time.Millisecond, config.DefaultRetryBase)
	assert.False(t, config.EnableAutoRefresh)
	assert.Equal(t, 10*time.Second, config.AutoRefreshInterval)
}

func TestWithDefaultExpiration(t *testing.T) {
	config := DefaultConfig()
	option := WithDefaultExpiration(60 * time.Second)
	option(config)
	
	assert.Equal(t, 60*time.Second, config.DefaultExpiration)
}

func TestWithDefaultTimeout(t *testing.T) {
	config := DefaultConfig()
	option := WithDefaultTimeout(10 * time.Second)
	option(config)
	
	assert.Equal(t, 10*time.Second, config.DefaultTimeout)
}

func TestWithDefaultRetry(t *testing.T) {
	config := DefaultConfig()
	option := WithDefaultRetry(RetryTypeLinear, 5, 200*time.Millisecond)
	option(config)
	
	assert.Equal(t, RetryTypeLinear, config.DefaultRetryType)
	assert.Equal(t, 5, config.DefaultRetryCount)
	assert.Equal(t, 200*time.Millisecond, config.DefaultRetryBase)
}

func TestWithAutoRefresh(t *testing.T) {
	config := DefaultConfig()
	option := WithAutoRefresh(true, 5*time.Second)
	option(config)
	
	assert.True(t, config.EnableAutoRefresh)
	assert.Equal(t, 5*time.Second, config.AutoRefreshInterval)
}

func TestRetryTypes(t *testing.T) {
	assert.Equal(t, RetryType("fixed"), RetryTypeFixed)
	assert.Equal(t, RetryType("exponential"), RetryTypeExponential)
	assert.Equal(t, RetryType("linear"), RetryTypeLinear)
}

func TestNewService(t *testing.T) {
	tests := []struct {
		name    string
		options []Option
		wantErr bool
	}{
		{
			name:    "default config",
			options: nil,
			wantErr: false,
		},
		{
			name: "with custom options",
			options: []Option{
				WithDefaultExpiration(60 * time.Second),
				WithDefaultTimeout(10 * time.Second),
			},
			wantErr: false,
		},
		{
			name: "with retry options",
			options: []Option{
				WithDefaultRetry(RetryTypeLinear, 5, 200*time.Millisecond),
			},
			wantErr: false,
		},
		{
			name: "with auto refresh",
			options: []Option{
				WithAutoRefresh(true, 5*time.Second),
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service, err := NewService(tt.options...)
			
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, service)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, service)
				assert.NotNil(t, service.appService)
			}
		})
	}
}

func TestNewServiceWithConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name:    "nil config",
			config:  nil,
			wantErr: true,
		},
		{
			name:    "valid config",
			config:  DefaultConfig(),
			wantErr: false,
		},
		{
			name: "custom config",
			config: &Config{
				DefaultExpiration:   60 * time.Second,
				DefaultTimeout:      10 * time.Second,
				DefaultRetryType:    RetryTypeLinear,
				DefaultRetryCount:   5,
				DefaultRetryBase:    200 * time.Millisecond,
				EnableAutoRefresh:   true,
				AutoRefreshInterval: 5 * time.Second,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service, err := NewServiceWithConfig(tt.config)
			
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, service)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, service)
				assert.NotNil(t, service.appService)
			}
		})
	}
}

func TestService_TryLock(t *testing.T) {
	service, err := NewService()
	require.NoError(t, err)
	
	ctx := context.Background()
	key := "test_lock"

	// Test TryLock with default options
	lock, err := service.TryLock(ctx, key)
	assert.NoError(t, err)
	assert.NotNil(t, lock)
	assert.Equal(t, key, lock.Key)
	assert.NotEmpty(t, lock.Value)
	assert.True(t, lock.IsValid)
	assert.False(t, lock.CreatedAt.IsZero())
	assert.False(t, lock.ExpiresAt.IsZero())
}

func TestService_TryLockWithOptions(t *testing.T) {
	service, err := NewService()
	require.NoError(t, err)
	
	ctx := context.Background()
	key := "test_lock_with_options"

	options := LockOptions{
		Expiration: 60 * time.Second,
		Timeout:    10 * time.Second,
		RetryType:  RetryTypeLinear,
		RetryCount: 5,
		RetryBase:  200 * time.Millisecond,
	}

	lock, err := service.TryLock(ctx, key, options)
	assert.NoError(t, err)
	assert.NotNil(t, lock)
	assert.Equal(t, key, lock.Key)
	assert.NotEmpty(t, lock.Value)
	assert.True(t, lock.IsValid)
}

func TestService_Lock(t *testing.T) {
	service, err := NewService()
	require.NoError(t, err)
	
	ctx := context.Background()
	key := "test_lock_with_retry"

	// Test Lock with default options
	lock, err := service.Lock(ctx, key)
	assert.NoError(t, err)
	assert.NotNil(t, lock)
	assert.Equal(t, key, lock.Key)
	assert.NotEmpty(t, lock.Value)
	assert.True(t, lock.IsValid)
}

func TestService_LockWithOptions(t *testing.T) {
	service, err := NewService()
	require.NoError(t, err)
	
	ctx := context.Background()
	key := "test_lock_retry_with_options"

	options := LockOptions{
		Expiration: 45 * time.Second,
		Timeout:    8 * time.Second,
		RetryType:  RetryTypeExponential,
		RetryCount: 2,
		RetryBase:  150 * time.Millisecond,
	}

	lock, err := service.Lock(ctx, key, options)
	assert.NoError(t, err)
	assert.NotNil(t, lock)
	assert.Equal(t, key, lock.Key)
	assert.NotEmpty(t, lock.Value)
	assert.True(t, lock.IsValid)
}

func TestService_Unlock(t *testing.T) {
	service, err := NewService()
	require.NoError(t, err)
	
	ctx := context.Background()
	key := "test_unlock"

	// Get a lock first
	lock, err := service.TryLock(ctx, key)
	require.NoError(t, err)
	require.NotNil(t, lock)

	// Try to unlock (should return error as not implemented)
	err = service.Unlock(ctx, key)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "暂未实现")
}

func TestService_Refresh(t *testing.T) {
	service, err := NewService()
	require.NoError(t, err)
	
	ctx := context.Background()
	key := "test_refresh"

	// Get a lock first
	lock, err := service.TryLock(ctx, key)
	require.NoError(t, err)
	require.NotNil(t, lock)

	// Try to refresh (should return error as not implemented)
	err = service.Refresh(ctx, key)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "暂未实现")
}

func TestService_IsLocked(t *testing.T) {
	service, err := NewService()
	require.NoError(t, err)
	
	ctx := context.Background()
	key := "test_is_locked"

	// Try to check lock status (should return error as not implemented)
	_, err = service.IsLocked(ctx, key)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "暂未实现")
}

func TestService_GetLockInfo(t *testing.T) {
	service, err := NewService()
	require.NoError(t, err)
	
	ctx := context.Background()
	key := "test_get_lock_info"

	// Try to get lock info (should return error as not implemented)
	_, err = service.GetLockInfo(ctx, key)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "暂未实现")
}

func TestService_StartAutoRefresh(t *testing.T) {
	service, err := NewService()
	require.NoError(t, err)
	
	ctx := context.Background()
	key := "test_auto_refresh"

	// Try to start auto refresh (should return error as not implemented)
	err = service.StartAutoRefresh(ctx, key, 5*time.Second)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "暂未实现")
}

func TestService_StopAutoRefresh(t *testing.T) {
	service, err := NewService()
	require.NoError(t, err)
	
	ctx := context.Background()
	key := "test_stop_auto_refresh"

	// Try to stop auto refresh (should return error as not implemented)
	err = service.StopAutoRefresh(ctx, key)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "暂未实现")
}

func TestLockStruct(t *testing.T) {
	now := time.Now()
	lock := Lock{
		Key:       "test_key",
		Value:     "test_value",
		CreatedAt: now,
		ExpiresAt: now.Add(time.Hour),
		IsValid:   true,
	}
	
	assert.Equal(t, "test_key", lock.Key)
	assert.Equal(t, "test_value", lock.Value)
	assert.Equal(t, now, lock.CreatedAt)
	assert.Equal(t, now.Add(time.Hour), lock.ExpiresAt)
	assert.True(t, lock.IsValid)
}

func TestLockOptionsStruct(t *testing.T) {
	options := LockOptions{
		Expiration: 30 * time.Second,
		Timeout:    5 * time.Second,
		RetryType:  RetryTypeExponential,
		RetryCount: 3,
		RetryBase:  100 * time.Millisecond,
	}
	
	assert.Equal(t, 30*time.Second, options.Expiration)
	assert.Equal(t, 5*time.Second, options.Timeout)
	assert.Equal(t, RetryTypeExponential, options.RetryType)
	assert.Equal(t, 3, options.RetryCount)
	assert.Equal(t, 100*time.Millisecond, options.RetryBase)
}

func TestService_ConcurrentLocks(t *testing.T) {
	service, err := NewService()
	require.NoError(t, err)
	
	ctx := context.Background()
	
	// Test that we can acquire multiple locks with different keys
	keys := []string{"lock1", "lock2", "lock3"}
	locks := make([]*Lock, len(keys))
	
	for i, key := range keys {
		lock, err := service.TryLock(ctx, key)
		assert.NoError(t, err)
		assert.NotNil(t, lock)
		assert.Equal(t, key, lock.Key)
		locks[i] = lock
	}
	
	// Verify all locks have different values
	values := make(map[string]bool)
	for _, lock := range locks {
		assert.False(t, values[lock.Value], "Lock values should be unique")
		values[lock.Value] = true
	}
}
