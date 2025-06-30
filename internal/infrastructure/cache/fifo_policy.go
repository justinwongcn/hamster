package cache

import (
	"context"
	"sync"
)

// fifoNode 队列节点
type fifoNode struct {
	key  string
	next *fifoNode
}

// FIFOPolicy 实现FIFO淘汰策略
// 使用单向链表实现队列，先进先出
// 线程安全，支持并发访问
type FIFOPolicy struct {
	capacity int                  // 容量限制，0表示无限制
	size     int                  // 当前大小
	cache    map[string]*fifoNode // 哈希表，快速定位节点
	head     *fifoNode            // 队列头（最早添加）
	tail     *fifoNode            // 队列尾（最新添加）
	mutex    sync.RWMutex         // 读写锁，保证并发安全
}

// NewFIFOPolicy 创建新的FIFO策略实例
// 参数:
//   - capacity: 容量限制，0表示无限制
//
// 返回值:
//   - *FIFOPolicy: 新的FIFO策略实例
func NewFIFOPolicy(capacity ...int) *FIFOPolicy {
	capacityVal := 0
	if len(capacity) > 0 && capacity[0] > 0 {
		capacityVal = capacity[0]
	}

	return &FIFOPolicy{
		capacity: capacityVal,
		size:     0,
		cache:    make(map[string]*fifoNode),
		head:     nil,
		tail:     nil,
	}
}

// KeyAccessed 记录key被访问
// FIFO策略中，已存在的key不会改变位置，只有新key会被添加到队列尾部
func (f *FIFOPolicy) KeyAccessed(_ context.Context, key string) error {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	// 如果key已存在，不更新位置（FIFO特性）
	if _, exists := f.cache[key]; exists {
		return nil
	}

	// 新key添加到队列尾部
	newNode := &fifoNode{key: key}
	f.cache[key] = newNode

	if f.tail == nil {
		// 队列为空
		f.head = newNode
		f.tail = newNode
	} else {
		// 添加到队列尾部
		f.tail.next = newNode
		f.tail = newNode
	}
	f.size++

	// 检查容量限制
	if f.capacity > 0 && f.size > f.capacity {
		// 移除队列头部节点
		if f.head != nil {
			oldHead := f.head
			f.head = f.head.next
			if f.head == nil {
				f.tail = nil
			}
			delete(f.cache, oldHead.key)
			f.size--
		}
	}

	return nil
}

// Evict 执行淘汰并返回被淘汰的key
// 移除最早添加的key（队列头部）
func (f *FIFOPolicy) Evict(context.Context) (string, error) {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	if f.size == 0 || f.head == nil {
		return "", nil
	}

	// 移除队列头部节点
	oldHead := f.head
	f.head = f.head.next
	if f.head == nil {
		f.tail = nil
	}
	delete(f.cache, oldHead.key)
	f.size--

	return oldHead.key, nil
}

// Remove 移除指定key
func (f *FIFOPolicy) Remove(_ context.Context, key string) error {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	node, exists := f.cache[key]
	if !exists {
		return nil
	}

	// 如果是头节点
	if f.head == node {
		f.head = f.head.next
		if f.head == nil {
			f.tail = nil
		}
	} else {
		// 找到前一个节点
		prev := f.head
		for prev != nil && prev.next != node {
			prev = prev.next
		}
		if prev != nil {
			prev.next = node.next
			if node == f.tail {
				f.tail = prev
			}
		}
	}

	delete(f.cache, key)
	f.size--
	return nil
}

// Has 检查key是否存在
func (f *FIFOPolicy) Has(_ context.Context, key string) (bool, error) {
	f.mutex.RLock()
	defer f.mutex.RUnlock()

	_, exists := f.cache[key]
	return exists, nil
}

// Size 返回当前跟踪的key数量
func (f *FIFOPolicy) Size(context.Context) (int, error) {
	f.mutex.RLock()
	defer f.mutex.RUnlock()

	return f.size, nil
}

// Clear 清空所有key
func (f *FIFOPolicy) Clear(context.Context) error {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	f.cache = make(map[string]*fifoNode)
	f.head = nil
	f.tail = nil
	f.size = 0
	return nil
}
