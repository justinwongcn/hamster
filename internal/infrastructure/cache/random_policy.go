package cache

import (
	"context"
	"math/rand"
	"sync"
	"time"
)

// RandomPolicy 实现随机淘汰策略
// 这是一个自定义策略示例，展示如何实现EvictionPolicy接口
// 当需要淘汰时，随机选择一个key进行淘汰
type RandomPolicy struct {
	capacity int            // 容量限制，0表示无限制
	keys     []string       // 存储所有key的切片
	keySet   map[string]int // key到索引的映射，用于快速查找
	mutex    sync.RWMutex   // 读写锁，保证并发安全
	rand     *rand.Rand     // 随机数生成器
}

// NewRandomPolicy 创建新的随机策略实例
// 参数:
//   - capacity: 容量限制，0表示无限制
//
// 返回值:
//   - *RandomPolicy: 新的随机策略实例
func NewRandomPolicy(capacity ...int) *RandomPolicy {
	capacityVal := 0
	if len(capacity) > 0 && capacity[0] > 0 {
		capacityVal = capacity[0]
	}

	return &RandomPolicy{
		capacity: capacityVal,
		keys:     make([]string, 0),
		keySet:   make(map[string]int),
		rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// KeyAccessed 记录key被访问
// 随机策略中，只需要记录key的存在，不需要维护访问顺序
func (r *RandomPolicy) KeyAccessed(_ context.Context, key string) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	// 如果key已存在，直接返回
	if _, exists := r.keySet[key]; exists {
		return nil
	}

	// 添加新key
	r.keys = append(r.keys, key)
	r.keySet[key] = len(r.keys) - 1

	// 检查容量限制
	if r.capacity > 0 && len(r.keys) > r.capacity {
		// 随机淘汰一个key
		_, err := r.evictInternal()
		return err
	}

	return nil
}

// Evict 执行淘汰并返回被淘汰的key
// 随机选择一个key进行淘汰
func (r *RandomPolicy) Evict(context.Context) (string, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	return r.evictInternal()
}

// evictInternal 内部淘汰方法（不加锁）
func (r *RandomPolicy) evictInternal() (string, error) {
	if len(r.keys) == 0 {
		return "", nil
	}

	// 随机选择一个索引
	randomIndex := r.rand.Intn(len(r.keys))
	key := r.keys[randomIndex]

	// 移除key
	r.removeByIndex(randomIndex)

	return key, nil
}

// Remove 移除指定key
func (r *RandomPolicy) Remove(_ context.Context, key string) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	index, exists := r.keySet[key]
	if !exists {
		return nil
	}

	r.removeByIndex(index)
	return nil
}

// removeByIndex 根据索引移除key（内部方法，不加锁）
func (r *RandomPolicy) removeByIndex(index int) {
	key := r.keys[index]

	// 将最后一个元素移动到被删除的位置
	lastIndex := len(r.keys) - 1
	if index != lastIndex {
		lastKey := r.keys[lastIndex]
		r.keys[index] = lastKey
		r.keySet[lastKey] = index
	}

	// 删除最后一个元素
	r.keys = r.keys[:lastIndex]
	delete(r.keySet, key)
}

// Has 判断key是否存在
func (r *RandomPolicy) Has(_ context.Context, key string) (bool, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	_, exists := r.keySet[key]
	return exists, nil
}

// Size 返回当前跟踪的key数量
func (r *RandomPolicy) Size(context.Context) (int, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	return len(r.keys), nil
}

// Clear 清空所有key
func (r *RandomPolicy) Clear(context.Context) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.keys = r.keys[:0]
	r.keySet = make(map[string]int)
	return nil
}
