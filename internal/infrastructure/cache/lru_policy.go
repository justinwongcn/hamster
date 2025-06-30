package cache

import (
	"context"
	"sync"
)

// lruNode 双向链表节点
type lruNode struct {
	key  string
	prev *lruNode
	next *lruNode
}

// LRUPolicy 实现LRU淘汰策略
// 使用双向链表+哈希表的经典LRU实现，避免索引管理问题
// 线程安全，支持并发访问
type LRUPolicy struct {
	capacity int                 // 容量限制，0表示无限制
	size     int                 // 当前大小
	cache    map[string]*lruNode // 哈希表，快速定位节点
	head     *lruNode            // 头节点（最近使用）
	tail     *lruNode            // 尾节点（最久未使用）
	mutex    sync.RWMutex        // 读写锁，保证并发安全
}

// NewLRUPolicy 创建新的LRU策略实例
// 参数:
//   - capacity: 容量限制，0表示无限制
//
// 返回值:
//   - *LRUPolicy: 新的LRU策略实例
func NewLRUPolicy(capacity ...int) *LRUPolicy {
	var capacityVal = 0
	if len(capacity) > 0 && capacity[0] > 0 {
		capacityVal = capacity[0]
	}

	// 创建头尾哨兵节点
	head := &lruNode{}
	tail := &lruNode{}
	head.next = tail
	tail.prev = head

	return &LRUPolicy{
		capacity: capacityVal,
		size:     0,
		cache:    make(map[string]*lruNode),
		head:     head,
		tail:     tail,
	}
}

// KeyAccessed 记录key被访问
// 将key移动到链表头部（最近使用位置）
func (l *LRUPolicy) KeyAccessed(_ context.Context, key string) error {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	if node, exists := l.cache[key]; exists {
		// key已存在，移动到头部
		l.moveToHead(node)
	} else {
		// 新key，添加到头部
		newNode := &lruNode{key: key}
		l.cache[key] = newNode
		l.addToHead(newNode)
		l.size++

		// 检查容量限制
		if l.capacity > 0 && l.size > l.capacity {
			// 移除尾部节点
			tail := l.removeTail()
			delete(l.cache, tail.key)
			l.size--
		}
	}
	return nil
}

// Evict 执行淘汰并返回被淘汰的key
// 移除最久未使用的key（链表尾部）
func (l *LRUPolicy) Evict(context.Context) (string, error) {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	if l.size == 0 {
		return "", nil
	}

	// 移除尾部节点
	tail := l.removeTail()
	delete(l.cache, tail.key)
	l.size--
	return tail.key, nil
}

// Remove 移除指定key
func (l *LRUPolicy) Remove(_ context.Context, key string) error {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	if node, exists := l.cache[key]; exists {
		l.removeNode(node)
		delete(l.cache, key)
		l.size--
	}
	return nil
}

// Has 判断key是否存在
func (l *LRUPolicy) Has(_ context.Context, key string) (bool, error) {
	l.mutex.RLock()
	defer l.mutex.RUnlock()

	_, exists := l.cache[key]
	return exists, nil
}

// Size 返回当前跟踪的key数量
func (l *LRUPolicy) Size(context.Context) (int, error) {
	l.mutex.RLock()
	defer l.mutex.RUnlock()

	return l.size, nil
}

// Clear 清空所有key
func (l *LRUPolicy) Clear(context.Context) error {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	l.cache = make(map[string]*lruNode)
	l.head.next = l.tail
	l.tail.prev = l.head
	l.size = 0
	return nil
}

// addToHead 将节点添加到头部
func (l *LRUPolicy) addToHead(node *lruNode) {
	node.prev = l.head
	node.next = l.head.next
	l.head.next.prev = node
	l.head.next = node
}

// removeNode 移除指定节点
func (l *LRUPolicy) removeNode(node *lruNode) {
	node.prev.next = node.next
	node.next.prev = node.prev
}

// moveToHead 将节点移动到头部
func (l *LRUPolicy) moveToHead(node *lruNode) {
	l.removeNode(node)
	l.addToHead(node)
}

// removeTail 移除尾部节点并返回
func (l *LRUPolicy) removeTail() *lruNode {
	lastNode := l.tail.prev
	l.removeNode(lastNode)
	return lastNode
}
