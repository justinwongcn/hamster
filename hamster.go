package hamster

import (
	"github.com/justinwongcn/hamster/cache"
	"github.com/justinwongcn/hamster/hash"
	"github.com/justinwongcn/hamster/lock"
)

// NewCache 创建缓存服务
// 这是创建缓存服务的便捷方法，使用默认配置
func NewCache(options ...cache.Option) (*cache.Service, error) {
	return cache.NewService(options...)
}

// NewCacheWithConfig 使用配置创建缓存服务
func NewCacheWithConfig(config *cache.Config) (*cache.Service, error) {
	return cache.NewServiceWithConfig(config)
}

// NewReadThroughCache 创建读透缓存服务
func NewReadThroughCache(options ...cache.Option) (*cache.ReadThroughService, error) {
	return cache.NewReadThroughService(options...)
}

// NewConsistentHash 创建一致性哈希服务
// 这是创建一致性哈希服务的便捷方法，使用默认配置
func NewConsistentHash(options ...hash.Option) (*hash.Service, error) {
	return hash.NewService(options...)
}

// NewConsistentHashWithConfig 使用配置创建一致性哈希服务
func NewConsistentHashWithConfig(config *hash.Config) (*hash.Service, error) {
	return hash.NewServiceWithConfig(config)
}

// NewDistributedLock 创建分布式锁服务
// 这是创建分布式锁服务的便捷方法，使用默认配置
func NewDistributedLock(options ...lock.Option) (*lock.Service, error) {
	return lock.NewService(options...)
}

// NewDistributedLockWithConfig 使用配置创建分布式锁服务
func NewDistributedLockWithConfig(config *lock.Config) (*lock.Service, error) {
	return lock.NewServiceWithConfig(config)
}

// Version 返回 Hamster 库的版本
const Version = "1.0.0"

// GetVersion 获取版本信息
func GetVersion() string {
	return Version
}
