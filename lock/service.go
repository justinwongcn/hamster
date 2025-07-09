package lock

import (
	"context"
	"fmt"
	"time"

	appLock "github.com/justinwongcn/hamster/internal/application/lock"
	infraLock "github.com/justinwongcn/hamster/internal/infrastructure/lock"
)

// Service 分布式锁服务公共接口
type Service struct {
	appService *appLock.DistributedLockApplicationService
}

// Config 分布式锁配置
type Config struct {
	// DefaultExpiration 默认锁过期时间
	DefaultExpiration time.Duration

	// DefaultTimeout 默认获取锁超时时间
	DefaultTimeout time.Duration

	// DefaultRetryType 默认重试类型
	DefaultRetryType RetryType

	// DefaultRetryCount 默认重试次数
	DefaultRetryCount int

	// DefaultRetryBase 默认重试基础间隔
	DefaultRetryBase time.Duration

	// EnableAutoRefresh 是否启用自动续约
	EnableAutoRefresh bool

	// AutoRefreshInterval 自动续约间隔
	AutoRefreshInterval time.Duration
}

// RetryType 重试类型
type RetryType string

const (
	RetryTypeFixed       RetryType = "fixed"
	RetryTypeExponential RetryType = "exponential"
	RetryTypeLinear      RetryType = "linear"
)

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	return &Config{
		DefaultExpiration:   30 * time.Second,
		DefaultTimeout:      5 * time.Second,
		DefaultRetryType:    RetryTypeExponential,
		DefaultRetryCount:   3,
		DefaultRetryBase:    100 * time.Millisecond,
		EnableAutoRefresh:   false,
		AutoRefreshInterval: 10 * time.Second,
	}
}

// Option 配置选项函数
type Option func(*Config)

// WithDefaultExpiration 设置默认过期时间
func WithDefaultExpiration(expiration time.Duration) Option {
	return func(c *Config) {
		c.DefaultExpiration = expiration
	}
}

// WithDefaultTimeout 设置默认超时时间
func WithDefaultTimeout(timeout time.Duration) Option {
	return func(c *Config) {
		c.DefaultTimeout = timeout
	}
}

// WithDefaultRetry 设置默认重试策略
func WithDefaultRetry(retryType RetryType, count int, base time.Duration) Option {
	return func(c *Config) {
		c.DefaultRetryType = retryType
		c.DefaultRetryCount = count
		c.DefaultRetryBase = base
	}
}

// WithAutoRefresh 设置自动续约
func WithAutoRefresh(enable bool, interval time.Duration) Option {
	return func(c *Config) {
		c.EnableAutoRefresh = enable
		c.AutoRefreshInterval = interval
	}
}

// NewService 创建分布式锁服务
func NewService(options ...Option) (*Service, error) {
	config := DefaultConfig()
	for _, option := range options {
		option(config)
	}

	return NewServiceWithConfig(config)
}

// NewServiceWithConfig 使用配置创建分布式锁服务
func NewServiceWithConfig(config *Config) (*Service, error) {
	if config == nil {
		return nil, fmt.Errorf("配置不能为空")
	}

	// 创建基础设施层
	distributedLock := infraLock.NewMemoryDistributedLock()

	// 创建应用服务
	appService := appLock.NewDistributedLockApplicationService(distributedLock)

	return &Service{
		appService: appService,
	}, nil
}

// Lock 锁信息
type Lock struct {
	Key       string    `json:"key"`
	Value     string    `json:"value"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
	IsValid   bool      `json:"is_valid"`
}

// LockOptions 加锁选项
type LockOptions struct {
	Expiration time.Duration
	Timeout    time.Duration
	RetryType  RetryType
	RetryCount int
	RetryBase  time.Duration
}

// TryLock 尝试获取锁（不重试）
func (s *Service) TryLock(ctx context.Context, key string, options ...LockOptions) (*Lock, error) {
	var opts LockOptions
	if len(options) > 0 {
		opts = options[0]
	} else {
		// 使用默认配置
		config := DefaultConfig()
		opts = LockOptions{
			Expiration: config.DefaultExpiration,
			Timeout:    config.DefaultTimeout,
			RetryType:  config.DefaultRetryType,
			RetryCount: config.DefaultRetryCount,
			RetryBase:  config.DefaultRetryBase,
		}
	}

	cmd := appLock.LockCommand{
		Key:        key,
		Expiration: opts.Expiration,
		Timeout:    opts.Timeout,
		RetryType:  string(opts.RetryType),
		RetryCount: opts.RetryCount,
		RetryBase:  opts.RetryBase,
	}

	result, err := s.appService.TryLock(ctx, cmd)
	if err != nil {
		return nil, err
	}

	return &Lock{
		Key:       result.Key,
		Value:     result.Value,
		CreatedAt: result.CreatedAt,
		ExpiresAt: result.ExpiresAt,
		IsValid:   result.IsValid,
	}, nil
}

// Lock 获取锁（支持重试）
func (s *Service) Lock(ctx context.Context, key string, options ...LockOptions) (*Lock, error) {
	var opts LockOptions
	if len(options) > 0 {
		opts = options[0]
	} else {
		// 使用默认配置
		config := DefaultConfig()
		opts = LockOptions{
			Expiration: config.DefaultExpiration,
			Timeout:    config.DefaultTimeout,
			RetryType:  config.DefaultRetryType,
			RetryCount: config.DefaultRetryCount,
			RetryBase:  config.DefaultRetryBase,
		}
	}

	cmd := appLock.LockCommand{
		Key:        key,
		Expiration: opts.Expiration,
		Timeout:    opts.Timeout,
		RetryType:  string(opts.RetryType),
		RetryCount: opts.RetryCount,
		RetryBase:  opts.RetryBase,
	}

	result, err := s.appService.Lock(ctx, cmd)
	if err != nil {
		return nil, err
	}

	return &Lock{
		Key:       result.Key,
		Value:     result.Value,
		CreatedAt: result.CreatedAt,
		ExpiresAt: result.ExpiresAt,
		IsValid:   result.IsValid,
	}, nil
}

// Unlock 释放锁
func (s *Service) Unlock(ctx context.Context, key string) error {
	// 暂时不支持释放锁，需要扩展应用服务接口
	return fmt.Errorf("释放锁功能暂未实现")
}

// Refresh 续约锁
func (s *Service) Refresh(ctx context.Context, key string) error {
	// 暂时不支持续约锁，需要扩展应用服务接口
	return fmt.Errorf("续约锁功能暂未实现")
}

// IsLocked 检查锁是否存在
func (s *Service) IsLocked(ctx context.Context, key string) (bool, error) {
	// 暂时不支持检查锁状态，需要扩展应用服务接口
	return false, fmt.Errorf("检查锁状态功能暂未实现")
}

// GetLockInfo 获取锁信息
func (s *Service) GetLockInfo(ctx context.Context, key string) (*Lock, error) {
	// 暂时不支持获取锁信息，需要扩展应用服务接口
	return nil, fmt.Errorf("获取锁信息功能暂未实现")
}

// StartAutoRefresh 启动自动续约
func (s *Service) StartAutoRefresh(ctx context.Context, key string, interval time.Duration) error {
	// 暂时不支持自动续约，需要扩展应用服务接口
	return fmt.Errorf("自动续约功能暂未实现")
}

// StopAutoRefresh 停止自动续约
func (s *Service) StopAutoRefresh(ctx context.Context, key string) error {
	// 暂时不支持停止自动续约，需要扩展应用服务接口
	return fmt.Errorf("停止自动续约功能暂未实现")
}
