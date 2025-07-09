# service.go - 缓存应用服务

## 文件概述

`service.go`
实现了缓存应用服务层，遵循DDD架构模式，协调领域服务和基础设施层，实现具体的业务用例。该文件提供了多种专门化的应用服务，包括基础缓存服务、读透缓存服务、写透缓存服务和布隆过滤器服务，为上层接口提供了丰富的缓存操作能力。

## 核心功能

### 1. ApplicationService 基础应用服务

```go
type ApplicationService struct {
    repository    cache.Repository
    cacheService  *cache.CacheService
    writeBackRepo cache.WriteBackRepository
}
```

**设计特点：**

- 协调领域服务和基础设施层
- 实现基本的CRUD操作
- 提供输入验证和错误处理
- 支持可选的写回缓存仓储

### 2. 数据传输对象 (DTOs)

#### CacheItemCommand - 缓存项命令

```go
type CacheItemCommand struct {
    Key        string
    Value      any
    Expiration time.Duration
}
```

#### CacheItemQuery - 缓存项查询

```go
type CacheItemQuery struct {
    Key string
}
```

#### CacheItemResult - 缓存项结果

```go
type CacheItemResult struct {
    Key       string
    Value     any
    Found     bool
    CreatedAt time.Time
    IsDirty   bool
}
```

#### CacheStatsResult - 缓存统计结果

```go
type CacheStatsResult struct {
    Hits      int64
    Misses    int64
    HitRate   float64
    Size      int64
    DirtyKeys []string
}
```

## 主要方法

### 1. 基础缓存操作

#### SetCacheItem - 设置缓存项

```go
func (s *ApplicationService) SetCacheItem(ctx context.Context, cmd CacheItemCommand) error
```

**用例**: 用户想要缓存一个数据项

**示例：**

```go
service := NewApplicationService(repository, cacheService, nil)

cmd := CacheItemCommand{
    Key:        "user:123",
    Value:      User{ID: "123", Name: "John"},
    Expiration: time.Hour,
}

err := service.SetCacheItem(ctx, cmd)
if err != nil {
    log.Printf("设置缓存失败: %v", err)
}
```

#### GetCacheItem - 获取缓存项

```go
func (s *ApplicationService) GetCacheItem(ctx context.Context, query CacheItemQuery) (*CacheItemResult, error)
```

**用例**: 用户想要获取一个缓存的数据项

**示例：**

```go
query := CacheItemQuery{Key: "user:123"}

result, err := service.GetCacheItem(ctx, query)
if err != nil {
    log.Printf("获取缓存失败: %v", err)
    return
}

if result.Found {
    fmt.Printf("找到缓存: %v\n", result.Value)
} else {
    fmt.Println("缓存未命中")
}
```

#### DeleteCacheItem - 删除缓存项

```go
func (s *ApplicationService) DeleteCacheItem(ctx context.Context, query CacheItemQuery) error
```

#### GetCacheStats - 获取缓存统计

```go
func (s *ApplicationService) GetCacheStats(ctx context.Context) (*CacheStatsResult, error)
```

### 2. ReadThroughApplicationService 读透缓存服务

```go
type ReadThroughApplicationService struct {
    *ApplicationService
    readThroughRepo cache.ReadThroughRepository
}
```

#### GetWithLoader - 使用加载器获取缓存项

```go
func (s *ReadThroughApplicationService) GetWithLoader(
    ctx context.Context,
    query CacheItemQuery,
    loader func(ctx context.Context, key string) (any, error),
    expiration time.Duration,
) (*CacheItemResult, error)
```

**用例**: 用户想要获取数据，如果缓存未命中则从数据源加载

**示例：**

```go
readThroughService := NewReadThroughApplicationService(repository, cacheService, readThroughRepo)

loader := func(ctx context.Context, key string) (any, error) {
    return database.GetUser(key)
}

result, err := readThroughService.GetWithLoader(ctx, query, loader, time.Hour)
if err != nil {
    log.Printf("读透获取失败: %v", err)
    return
}

fmt.Printf("获取到数据: %v\n", result.Value)
```

### 3. WriteThroughApplicationService 写透缓存服务

```go
type WriteThroughApplicationService struct {
    *ApplicationService
    writeThroughRepo cache.WriteThroughRepository
}
```

#### SetWithStore - 使用存储器设置缓存项

```go
func (s *WriteThroughApplicationService) SetWithStore(
    ctx context.Context,
    cmd CacheItemCommand,
    storer func(ctx context.Context, key string, val any) error,
) error
```

**用例**: 用户想要设置数据，同时写入缓存和持久化存储

**示例：**

```go
writeThroughService := NewWriteThroughApplicationService(repository, cacheService, writeThroughRepo)

storer := func(ctx context.Context, key string, val any) error {
    return database.SaveUser(key, val)
}

cmd := CacheItemCommand{
    Key:        "user:456",
    Value:      User{ID: "456", Name: "Jane"},
    Expiration: time.Hour,
}

err := writeThroughService.SetWithStore(ctx, cmd, storer)
if err != nil {
    log.Printf("写透设置失败: %v", err)
}
```

### 4. BloomFilterApplicationService 布隆过滤器服务

```go
type BloomFilterApplicationService struct {
    *ApplicationService
    bloomFilterCache interface {
        cache.Repository
        GetBloomFilterStats(ctx context.Context) (cache.BloomFilterStats, error)
        ClearBloomFilter(ctx context.Context) error
        AddKeyToBloomFilter(ctx context.Context, key string) error
        HasKeyInBloomFilter(ctx context.Context, key string) bool
        SetAutoAddToBloom(autoAdd bool)
        IsAutoAddToBloomEnabled() bool
    }
}
```

#### 布隆过滤器数据传输对象

```go
type BloomFilterStatsResult struct {
    ExpectedElements      uint64  `json:"expected_elements"`
    FalsePositiveRate     float64 `json:"false_positive_rate"`
    BitArraySize          uint64  `json:"bit_array_size"`
    HashFunctions         uint64  `json:"hash_functions"`
    AddedElements         uint64  `json:"added_elements"`
    SetBits               uint64  `json:"set_bits"`
    EstimatedFPR          float64 `json:"estimated_fpr"`
    MemoryUsage           uint64  `json:"memory_usage"`
    LoadFactor            float64 `json:"load_factor"`
    IsOverloaded          bool    `json:"is_overloaded"`
    EfficiencyRatio       float64 `json:"efficiency_ratio"`
    AutoAddToBloomEnabled bool    `json:"auto_add_to_bloom_enabled"`
}

type BloomFilterKeyCommand struct {
    Key string `json:"key"`
}

type BloomFilterKeyResult struct {
    Key           string `json:"key"`
    MightExist    bool   `json:"might_exist"`
    InBloomFilter bool   `json:"in_bloom_filter"`
}
```

#### 核心方法

##### GetBloomFilterStats - 获取布隆过滤器统计信息

```go
func (s *BloomFilterApplicationService) GetBloomFilterStats(ctx context.Context, query BloomFilterStatsQuery) (*BloomFilterStatsResult, error)
```

**用例**: 用户想要查看布隆过滤器的使用情况和性能指标

##### AddKeyToBloomFilter - 添加键到布隆过滤器

```go
func (s *BloomFilterApplicationService) AddKeyToBloomFilter(ctx context.Context, cmd BloomFilterKeyCommand) error
```

**用例**: 用户想要手动添加一个键到布隆过滤器

##### CheckKeyInBloomFilter - 检查键是否在布隆过滤器中

```go
func (s *BloomFilterApplicationService) CheckKeyInBloomFilter(ctx context.Context, query BloomFilterKeyQuery) (*BloomFilterKeyResult, error)
```

**用例**: 用户想要检查一个键是否可能存在于布隆过滤器中

##### GetWithBloomFilter - 使用布隆过滤器优化的获取操作

```go
func (s *BloomFilterApplicationService) GetWithBloomFilter(ctx context.Context, query CacheItemQuery) (*CacheItemResult, error)
```

**用例**: 用户想要获取数据，利用布隆过滤器减少无效查询

## 使用示例

### 1. 基础缓存服务使用

```go
package main

import (
    "context"
    "fmt"
    "log"
    "time"
    
    "github.com/justinwongcn/hamster/internal/application/cache"
    domainCache "github.com/justinwongcn/hamster/internal/domain/cache"
    infraCache "github.com/justinwongcn/hamster/internal/infrastructure/cache"
)

func main() {
    // 创建基础设施
    repository := infraCache.NewMaxMemoryCache(1024 * 1024) // 1MB
    cacheService := domainCache.NewCacheService(domainCache.NewLRUEvictionStrategy())
    
    // 创建应用服务
    appService := cache.NewApplicationService(repository, cacheService, nil)
    
    ctx := context.Background()
    
    // 设置缓存项
    cmd := cache.CacheItemCommand{
        Key:        "user:123",
        Value:      map[string]string{"name": "John", "email": "john@example.com"},
        Expiration: time.Hour,
    }
    
    err := appService.SetCacheItem(ctx, cmd)
    if err != nil {
        log.Printf("设置缓存失败: %v", err)
        return
    }
    
    fmt.Println("缓存设置成功")
    
    // 获取缓存项
    query := cache.CacheItemQuery{Key: "user:123"}
    result, err := appService.GetCacheItem(ctx, query)
    if err != nil {
        log.Printf("获取缓存失败: %v", err)
        return
    }
    
    if result.Found {
        fmt.Printf("找到缓存: %v\n", result.Value)
    } else {
        fmt.Println("缓存未命中")
    }
    
    // 获取统计信息
    stats, err := appService.GetCacheStats(ctx)
    if err != nil {
        log.Printf("获取统计失败: %v", err)
        return
    }
    
    fmt.Printf("缓存统计: 命中=%d, 未命中=%d, 命中率=%.2f%%\n",
        stats.Hits, stats.Misses, stats.HitRate*100)
}
```

### 2. 读透缓存服务使用

```go
func demonstrateReadThroughService() {
    // 创建读透缓存仓储
    readThroughRepo := &mockReadThroughRepository{}
    
    // 创建读透应用服务
    readThroughService := cache.NewReadThroughApplicationService(
        repository, cacheService, readThroughRepo)
    
    ctx := context.Background()
    
    // 定义数据加载器
    userLoader := func(ctx context.Context, key string) (any, error) {
        // 模拟从数据库加载用户
        userID := strings.TrimPrefix(key, "user:")
        return map[string]interface{}{
            "id":   userID,
            "name": fmt.Sprintf("User_%s", userID),
            "age":  25,
        }, nil
    }
    
    // 使用读透缓存获取数据
    query := cache.CacheItemQuery{Key: "user:456"}
    result, err := readThroughService.GetWithLoader(ctx, query, userLoader, time.Hour)
    if err != nil {
        log.Printf("读透获取失败: %v", err)
        return
    }
    
    fmt.Printf("读透获取成功: %v\n", result.Value)
    
    // 第二次获取应该从缓存获取
    result2, err := readThroughService.GetWithLoader(ctx, query, userLoader, time.Hour)
    if err == nil {
        fmt.Printf("第二次获取: %v\n", result2.Value)
    }
}
```

### 3. 写透缓存服务使用

```go
func demonstrateWriteThroughService() {
    // 创建写透缓存仓储
    writeThroughRepo := &mockWriteThroughRepository{}
    
    // 创建写透应用服务
    writeThroughService := cache.NewWriteThroughApplicationService(
        repository, cacheService, writeThroughRepo)
    
    ctx := context.Background()
    
    // 定义数据存储器
    userStorer := func(ctx context.Context, key string, val any) error {
        // 模拟保存到数据库
        fmt.Printf("保存到数据库: %s = %v\n", key, val)
        return nil
    }
    
    // 使用写透缓存设置数据
    cmd := cache.CacheItemCommand{
        Key:        "user:789",
        Value:      map[string]string{"name": "Alice", "email": "alice@example.com"},
        Expiration: time.Hour,
    }
    
    err := writeThroughService.SetWithStore(ctx, cmd, userStorer)
    if err != nil {
        log.Printf("写透设置失败: %v", err)
        return
    }
    
    fmt.Println("写透设置成功，数据已同步到缓存和数据库")
    
    // 验证数据已缓存
    query := cache.CacheItemQuery{Key: "user:789"}
    result, err := writeThroughService.GetCacheItem(ctx, query)
    if err == nil && result.Found {
        fmt.Printf("验证缓存: %v\n", result.Value)
    }
}
```

### 4. 布隆过滤器服务使用

```go
func demonstrateBloomFilterService() {
    // 创建布隆过滤器缓存
    bloomFilterCache := createBloomFilterCache()
    
    // 创建布隆过滤器应用服务
    bloomService := cache.NewBloomFilterApplicationService(
        repository, cacheService, bloomFilterCache)
    
    ctx := context.Background()
    
    // 添加键到布隆过滤器
    keys := []string{"user:1", "user:2", "user:3", "product:a", "product:b"}
    
    for _, key := range keys {
        cmd := cache.BloomFilterKeyCommand{Key: key}
        err := bloomService.AddKeyToBloomFilter(ctx, cmd)
        if err != nil {
            log.Printf("添加键失败: %v", err)
            continue
        }
        fmt.Printf("添加键到布隆过滤器: %s\n", key)
    }
    
    // 检查键是否存在
    testKeys := []string{"user:1", "user:999", "product:a", "product:z"}
    
    for _, key := range testKeys {
        query := cache.BloomFilterKeyQuery{Key: key}
        result, err := bloomService.CheckKeyInBloomFilter(ctx, query)
        if err != nil {
            log.Printf("检查键失败: %v", err)
            continue
        }
        
        fmt.Printf("键 %s 在布隆过滤器中: %v\n", key, result.InBloomFilter)
    }
    
    // 获取布隆过滤器统计信息
    statsQuery := cache.BloomFilterStatsQuery{}
    stats, err := bloomService.GetBloomFilterStats(ctx, statsQuery)
    if err != nil {
        log.Printf("获取统计失败: %v", err)
        return
    }
    
    fmt.Printf("布隆过滤器统计:\n")
    fmt.Printf("  预期元素数: %d\n", stats.ExpectedElements)
    fmt.Printf("  已添加元素: %d\n", stats.AddedElements)
    fmt.Printf("  假阳性率: %.4f\n", stats.FalsePositiveRate)
    fmt.Printf("  估算假阳性率: %.4f\n", stats.EstimatedFPR)
    fmt.Printf("  负载因子: %.4f\n", stats.LoadFactor)
    fmt.Printf("  是否过载: %v\n", stats.IsOverloaded)
    fmt.Printf("  自动添加启用: %v\n", stats.AutoAddToBloomEnabled)
    
    // 使用布隆过滤器优化的获取操作
    for _, key := range testKeys {
        query := cache.CacheItemQuery{Key: key}
        result, err := bloomService.GetWithBloomFilter(ctx, query)
        if err != nil {
            log.Printf("布隆过滤器获取失败: %v", err)
            continue
        }
        
        if result.Found {
            fmt.Printf("布隆过滤器获取成功: %s = %v\n", key, result.Value)
        } else {
            fmt.Printf("布隆过滤器过滤: %s 不存在\n", key)
        }
    }
}

func createBloomFilterCache() interface {
    cache.Repository
    GetBloomFilterStats(ctx context.Context) (cache.BloomFilterStats, error)
    ClearBloomFilter(ctx context.Context) error
    AddKeyToBloomFilter(ctx context.Context, key string) error
    HasKeyInBloomFilter(ctx context.Context, key string) bool
    SetAutoAddToBloom(autoAdd bool)
    IsAutoAddToBloomEnabled() bool
} {
    // 实际实现中应该返回真正的布隆过滤器缓存实例
    // 这里只是示例
    return nil
}
```

### 5. 综合使用示例

```go
func demonstrateComprehensiveUsage() {
    // 创建所有必要的组件
    repository := infraCache.NewMaxMemoryCache(1024 * 1024)
    cacheService := domainCache.NewCacheService(domainCache.NewLRUEvictionStrategy())
    
    // 创建基础应用服务
    appService := cache.NewApplicationService(repository, cacheService, nil)
    
    ctx := context.Background()
    
    // 批量设置缓存项
    users := []struct {
        ID   string
        Name string
        Age  int
    }{
        {"1", "Alice", 25},
        {"2", "Bob", 30},
        {"3", "Charlie", 35},
    }
    
    fmt.Println("批量设置用户缓存...")
    for _, user := range users {
        cmd := cache.CacheItemCommand{
            Key:        fmt.Sprintf("user:%s", user.ID),
            Value:      user,
            Expiration: time.Hour,
        }
        
        err := appService.SetCacheItem(ctx, cmd)
        if err != nil {
            log.Printf("设置用户%s失败: %v", user.ID, err)
        } else {
            fmt.Printf("设置用户%s成功\n", user.ID)
        }
    }
    
    // 批量获取缓存项
    fmt.Println("批量获取用户缓存...")
    for i := 1; i <= 5; i++ { // 包括不存在的用户
        query := cache.CacheItemQuery{Key: fmt.Sprintf("user:%d", i)}
        result, err := appService.GetCacheItem(ctx, query)
        if err != nil {
            log.Printf("获取用户%d失败: %v", i, err)
            continue
        }
        
        if result.Found {
            fmt.Printf("用户%d: %v\n", i, result.Value)
        } else {
            fmt.Printf("用户%d: 未找到\n", i)
        }
    }
    
    // 获取最终统计
    stats, err := appService.GetCacheStats(ctx)
    if err != nil {
        log.Printf("获取统计失败: %v", err)
        return
    }
    
    fmt.Printf("\n最终统计:\n")
    fmt.Printf("  总命中: %d\n", stats.Hits)
    fmt.Printf("  总未命中: %d\n", stats.Misses)
    fmt.Printf("  命中率: %.2f%%\n", stats.HitRate*100)
    fmt.Printf("  缓存大小: %d\n", stats.Size)
    
    // 清理部分缓存
    fmt.Println("\n清理用户2的缓存...")
    deleteQuery := cache.CacheItemQuery{Key: "user:2"}
    err = appService.DeleteCacheItem(ctx, deleteQuery)
    if err != nil {
        log.Printf("删除失败: %v", err)
    } else {
        fmt.Println("删除成功")
    }
    
    // 验证删除结果
    result, err := appService.GetCacheItem(ctx, deleteQuery)
    if err == nil {
        fmt.Printf("删除后查询结果: Found=%v\n", result.Found)
    }
}
```

## 设计原则

### 1. 应用服务职责

- 协调领域服务和基础设施
- 实现具体的业务用例
- 提供输入验证和错误处理
- 转换数据传输对象

### 2. 依赖注入

- 通过构造函数注入依赖
- 支持可选的依赖组件
- 便于测试和扩展

### 3. 错误处理

- 统一的错误包装和传播
- 有意义的错误消息
- 区分业务错误和技术错误

### 4. 数据传输对象

- 清晰的输入输出结构
- JSON标签支持序列化
- 业务语义的字段命名

## 注意事项

### 1. 输入验证

```go
// ✅ 推荐：在应用服务层进行输入验证
func (s *ApplicationService) SetCacheItem(ctx context.Context, cmd CacheItemCommand) error {
    if err := s.validateCacheItemCommand(cmd); err != nil {
        return fmt.Errorf("验证缓存项命令失败: %w", err)
    }
    // ... 业务逻辑
}

// ❌ 避免：跳过输入验证
func (s *ApplicationService) SetCacheItem(ctx context.Context, cmd CacheItemCommand) error {
    // 直接调用底层服务，没有验证
    return s.repository.Set(ctx, cmd.Key, cmd.Value, cmd.Expiration)
}
```

### 2. 错误处理

```go
// ✅ 推荐：包装错误并提供上下文
if err != nil {
    return fmt.Errorf("设置缓存项失败: %w", err)
}

// ❌ 避免：直接返回底层错误
if err != nil {
    return err // 丢失了上下文信息
}
```

### 3. 依赖管理

```go
// ✅ 推荐：通过构造函数注入依赖
func NewApplicationService(
    repository cache.Repository,
    cacheService *cache.CacheService,
    writeBackRepo cache.WriteBackRepository,
) *ApplicationService {
    return &ApplicationService{
        repository:    repository,
        cacheService:  cacheService,
        writeBackRepo: writeBackRepo,
    }
}

// ❌ 避免：在方法中创建依赖
func (s *ApplicationService) SomeMethod() {
    service := cache.NewCacheService(...) // 违反依赖注入原则
}
```

### 4. 上下文传递

```go
// ✅ 推荐：始终传递上下文
func (s *ApplicationService) GetCacheItem(ctx context.Context, query CacheItemQuery) (*CacheItemResult, error) {
    return s.repository.Get(ctx, query.Key)
}

// ❌ 避免：忽略上下文
func (s *ApplicationService) GetCacheItem(query CacheItemQuery) (*CacheItemResult, error) {
    return s.repository.Get(context.Background(), query.Key) // 硬编码上下文
}
```

### 5. 数据转换

```go
// ✅ 推荐：在应用服务层进行数据转换
func (s *BloomFilterApplicationService) GetBloomFilterStats(ctx context.Context, query BloomFilterStatsQuery) (*BloomFilterStatsResult, error) {
    stats, err := s.bloomFilterCache.GetBloomFilterStats(ctx)
    if err != nil {
        return nil, err
    }
    
    // 转换为应用层的DTO
    return &BloomFilterStatsResult{
        ExpectedElements:  stats.Config().ExpectedElements(),
        FalsePositiveRate: stats.Config().FalsePositiveRate(),
        // ... 其他字段
    }, nil
}
```
