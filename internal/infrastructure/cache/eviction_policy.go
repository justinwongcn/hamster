package cache

// EvictionPolicy 定义缓存淘汰策略接口
type EvictionPolicy interface {
    KeyAccessed(key string) // 记录key被访问
    Evict() string          // 执行淘汰并返回被淘汰的key
    Remove(key string)      // 移除指定key
    Has(key string) bool    // 判断key是否存在
}