# Application 应用服务层

应用服务层协调领域对象和基础设施，实现具体的业务用例。该层负责事务管理、权限控制、数据转换等跨领域的关注点，为上层提供粗粒度的业务接口。

## 📁 包结构

```
application/
├── cache/                     # 缓存应用服务
│   └── service.go            # 缓存应用服务实现
├── consistent_hash/           # 一致性哈希应用服务
│   └── service.go            # 一致性哈希应用服务实现
└── lock/                      # 分布式锁应用服务
    └── service.go            # 分布式锁应用服务实现
```

## 🎯 设计原则

### 1. 用例驱动

- 每个方法对应一个具体的业务用例
- 方法名清晰表达业务意图
- 参数和返回值面向业务概念

### 2. 事务边界

- 定义清晰的事务边界
- 确保数据一致性
- 处理并发和异常情况

### 3. 数据转换

- 将领域对象转换为DTO
- 验证输入参数
- 格式化输出结果

### 4. 协调职责

- 协调多个领域服务
- 管理跨领域的业务流程
- 处理复杂的业务逻辑

## 🏗️ 核心服务

### 缓存应用服务 (CacheApplicationService)

提供缓存相关的业务用例：

```go
type ApplicationService struct {
    repository   cache.Repository
    cacheService *cache.CacheService
    validator    *cache.CacheValidator
}

// 主要用例
func (s *ApplicationService) GetCacheItem(ctx context.Context, query CacheItemQuery) (*CacheItemResult, error)
func (s *ApplicationService) SetCacheItem(ctx context.Context, cmd CacheItemCommand) error
func (s *ApplicationService) DeleteCacheItem(ctx context.Context, cmd DeleteCacheItemCommand) error
func (s *ApplicationService) GetCacheStats(ctx context.Context) (*CacheStatsResult, error)
func (s *ApplicationService) ClearCache(ctx context.Context) error
```

### 分布式锁应用服务 (DistributedLockApplicationService)

提供分布式锁相关的业务用例：

```go
type DistributedLockApplicationService struct {
    distributedLock domainLock.DistributedLock
}

// 主要用例
func (s *DistributedLockApplicationService) TryLock(ctx context.Context, cmd LockCommand) (*LockResult, error)
func (s *DistributedLockApplicationService) Lock(ctx context.Context, cmd LockCommand) (*LockResult, error)
func (s *DistributedLockApplicationService) SingleflightLock(ctx context.Context, cmd LockCommand) (*LockResult, error)
func (s *DistributedLockApplicationService) RefreshLock(ctx context.Context, cmd RefreshCommand, lock domainLock.Lock) error
func (s *DistributedLockApplicationService) UnlockLock(ctx context.Context, cmd UnlockCommand, lock domainLock.Lock) error
```

### 一致性哈希应用服务 (ConsistentHashApplicationService)

提供一致性哈希相关的业务用例：

```go
type ConsistentHashApplicationService struct {
    peerPicker domainHash.PeerPicker
}

// 主要用例
func (s *ConsistentHashApplicationService) SelectPeer(ctx context.Context, cmd PeerSelectionCommand) (*PeerSelectionResult, error)
func (s *ConsistentHashApplicationService) SelectMultiplePeers(ctx context.Context, cmd MultiplePeerSelectionCommand) (*MultiplePeerSelectionResult, error)
func (s *ConsistentHashApplicationService) AddPeers(ctx context.Context, cmd AddPeersCommand) error
func (s *ConsistentHashApplicationService) RemovePeers(ctx context.Context, cmd RemovePeersCommand) error
func (s *ConsistentHashApplicationService) GetHashStats(ctx context.Context) (*HashStatsResult, error)
```

## 🔧 使用示例

### 缓存应用服务使用

```go
// 创建应用服务
repository := cache.NewMaxMemoryCache(1024 * 1024)
cacheService := cache.NewCacheService()
appService := NewApplicationService(repository, cacheService, nil)

// 设置缓存项
cmd := CacheItemCommand{
    Key:        "user:123",
    Value:      "John Doe",
    Expiration: time.Hour,
}

err := appService.SetCacheItem(ctx, cmd)
if err != nil {
    log.Printf("设置缓存失败: %v", err)
    return
}

// 获取缓存项
query := CacheItemQuery{Key: "user:123"}
result, err := appService.GetCacheItem(ctx, query)
if err != nil {
    log.Printf("获取缓存失败: %v", err)
    return
}

fmt.Printf("用户信息: %v\n", result.Value)
```

### 分布式锁应用服务使用

```go
// 创建应用服务
lockManager := lock.NewMemoryDistributedLock()
appService := NewDistributedLockApplicationService(lockManager)

// 获取锁
cmd := LockCommand{
    Key:        "resource:123",
    Expiration: time.Minute,
    Timeout:    5 * time.Second,
    RetryType:  "exponential",
    RetryCount: 3,
    RetryBase:  100 * time.Millisecond,
}

lockResult, err := appService.Lock(ctx, cmd)
if err != nil {
    log.Printf("获取锁失败: %v", err)
    return
}

// 执行业务逻辑
fmt.Printf("获取锁成功: %s\n", lockResult.Value)

// 释放锁
unlockCmd := UnlockCommand{Key: "resource:123"}
err = appService.UnlockLock(ctx, unlockCmd, lockResult.lock)
if err != nil {
    log.Printf("释放锁失败: %v", err)
}
```

### 一致性哈希应用服务使用

```go
// 创建应用服务
hashMap := consistent_hash.NewConsistentHashMap(150, nil)
picker := consistent_hash.NewSingleflightPeerPicker(hashMap)
appService := NewConsistentHashApplicationService(picker)

// 添加节点
addCmd := AddPeersCommand{
    Peers: []PeerRequest{
        {ID: "server1", Address: "192.168.1.1:8080", Weight: 100},
        {ID: "server2", Address: "192.168.1.2:8080", Weight: 100},
    },
}

err := appService.AddPeers(ctx, addCmd)
if err != nil {
    log.Printf("添加节点失败: %v", err)
    return
}

// 选择节点
selectCmd := PeerSelectionCommand{Key: "user:123"}
result, err := appService.SelectPeer(ctx, selectCmd)
if err != nil {
    log.Printf("选择节点失败: %v", err)
    return
}

fmt.Printf("用户分配到服务器: %s\n", result.Peer.ID)
```

## 📊 命令和查询对象

### 命令对象 (Commands)

用于修改系统状态的操作：

```go
// 缓存命令
type CacheItemCommand struct {
    Key        string        `json:"key"`
    Value      any          `json:"value"`
    Expiration time.Duration `json:"expiration"`
}

// 锁命令
type LockCommand struct {
    Key        string        `json:"key"`
    Expiration time.Duration `json:"expiration"`
    Timeout    time.Duration `json:"timeout"`
    RetryType  string        `json:"retry_type"`
    RetryCount int           `json:"retry_count"`
    RetryBase  time.Duration `json:"retry_base"`
}

// 节点管理命令
type AddPeersCommand struct {
    Peers []PeerRequest `json:"peers"`
}
```

### 查询对象 (Queries)

用于查询系统状态的操作：

```go
// 缓存查询
type CacheItemQuery struct {
    Key string `json:"key"`
}

// 节点选择查询
type PeerSelectionCommand struct {
    Key string `json:"key"`
}

// 多节点选择查询
type MultiplePeerSelectionCommand struct {
    Key   string `json:"key"`
    Count int    `json:"count"`
}
```

### 结果对象 (Results)

返回给调用者的数据传输对象：

```go
// 缓存结果
type CacheItemResult struct {
    Key       string    `json:"key"`
    Value     any       `json:"value"`
    Found     bool      `json:"found"`
    ExpiresAt time.Time `json:"expires_at"`
}

// 锁结果
type LockResult struct {
    Key       string    `json:"key"`
    Value     string    `json:"value"`
    CreatedAt time.Time `json:"created_at"`
    ExpiresAt time.Time `json:"expires_at"`
    IsValid   bool      `json:"is_valid"`
}

// 节点选择结果
type PeerSelectionResult struct {
    Key  string     `json:"key"`
    Peer PeerResult `json:"peer"`
}
```

## ⚠️ 注意事项

### 1. 输入验证

```go
func (s *ApplicationService) validateCacheItemCommand(cmd CacheItemCommand) error {
    if cmd.Key == "" {
        return errors.New("缓存键不能为空")
    }
    
    if len(cmd.Key) > 250 {
        return errors.New("缓存键长度不能超过250个字符")
    }
    
    if cmd.Expiration < 0 {
        return errors.New("过期时间不能为负数")
    }
    
    return nil
}
```

### 2. 错误处理

```go
func (s *ApplicationService) GetCacheItem(ctx context.Context, query CacheItemQuery) (*CacheItemResult, error) {
    // 验证输入
    if err := s.validateCacheItemQuery(query); err != nil {
        return nil, fmt.Errorf("验证查询参数失败: %w", err)
    }
    
    // 调用领域服务
    value, err := s.repository.Get(ctx, query.Key)
    if err != nil {
        if errors.Is(err, cache.ErrKeyNotFound) {
            return &CacheItemResult{
                Key:   query.Key,
                Found: false,
            }, nil
        }
        return nil, fmt.Errorf("获取缓存项失败: %w", err)
    }
    
    return &CacheItemResult{
        Key:   query.Key,
        Value: value,
        Found: true,
    }, nil
}
```

### 3. 事务管理

```go
func (s *ApplicationService) TransferCacheItem(ctx context.Context, cmd TransferCommand) error {
    // 开始事务（如果支持）
    tx, err := s.repository.BeginTx(ctx)
    if err != nil {
        return err
    }
    defer tx.Rollback()
    
    // 执行多个操作
    value, err := tx.Get(ctx, cmd.SourceKey)
    if err != nil {
        return err
    }
    
    err = tx.Set(ctx, cmd.TargetKey, value, cmd.Expiration)
    if err != nil {
        return err
    }
    
    err = tx.Delete(ctx, cmd.SourceKey)
    if err != nil {
        return err
    }
    
    // 提交事务
    return tx.Commit()
}
```

### 4. 并发控制

```go
func (s *ApplicationService) IncrementCounter(ctx context.Context, cmd IncrementCommand) error {
    // 使用分布式锁确保原子性
    lockKey := fmt.Sprintf("lock:counter:%s", cmd.Key)
    lock, err := s.lockService.TryLock(ctx, lockKey, time.Second)
    if err != nil {
        return err
    }
    defer lock.Unlock(ctx)
    
    // 执行原子操作
    current, err := s.repository.Get(ctx, cmd.Key)
    if err != nil && !errors.Is(err, cache.ErrKeyNotFound) {
        return err
    }
    
    var newValue int64
    if current != nil {
        newValue = current.(int64) + cmd.Delta
    } else {
        newValue = cmd.Delta
    }
    
    return s.repository.Set(ctx, cmd.Key, newValue, cmd.Expiration)
}
```

## 🧪 测试策略

### 单元测试

```go
func TestApplicationService_GetCacheItem(t *testing.T) {
    tests := []struct {
        name    string
        query   CacheItemQuery
        setup   func(*MockRepository)
        want    *CacheItemResult
        wantErr bool
    }{
        {
            name:  "成功获取缓存项",
            query: CacheItemQuery{Key: "test_key"},
            setup: func(repo *MockRepository) {
                repo.On("Get", mock.Anything, "test_key").Return("test_value", nil)
            },
            want: &CacheItemResult{
                Key:   "test_key",
                Value: "test_value",
                Found: true,
            },
            wantErr: false,
        },
        {
            name:  "缓存项不存在",
            query: CacheItemQuery{Key: "missing_key"},
            setup: func(repo *MockRepository) {
                repo.On("Get", mock.Anything, "missing_key").Return(nil, cache.ErrKeyNotFound)
            },
            want: &CacheItemResult{
                Key:   "missing_key",
                Found: false,
            },
            wantErr: false,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            repo := &MockRepository{}
            tt.setup(repo)
            
            service := NewApplicationService(repo, nil, nil)
            result, err := service.GetCacheItem(context.Background(), tt.query)
            
            if tt.wantErr {
                assert.Error(t, err)
            } else {
                assert.NoError(t, err)
                assert.Equal(t, tt.want, result)
            }
            
            repo.AssertExpectations(t)
        })
    }
}
```

### 集成测试

```go
func TestApplicationService_Integration(t *testing.T) {
    // 使用真实的基础设施组件
    repository := cache.NewMaxMemoryCache(1024)
    cacheService := cache.NewCacheService()
    appService := NewApplicationService(repository, cacheService, nil)
    
    ctx := context.Background()
    
    // 测试完整的业务流程
    cmd := CacheItemCommand{
        Key:        "integration_test",
        Value:      "test_data",
        Expiration: time.Minute,
    }
    
    // 设置缓存项
    err := appService.SetCacheItem(ctx, cmd)
    require.NoError(t, err)
    
    // 获取缓存项
    query := CacheItemQuery{Key: "integration_test"}
    result, err := appService.GetCacheItem(ctx, query)
    require.NoError(t, err)
    assert.True(t, result.Found)
    assert.Equal(t, "test_data", result.Value)
    
    // 删除缓存项
    deleteCmd := DeleteCacheItemCommand{Key: "integration_test"}
    err = appService.DeleteCacheItem(ctx, deleteCmd)
    require.NoError(t, err)
    
    // 验证删除结果
    result, err = appService.GetCacheItem(ctx, query)
    require.NoError(t, err)
    assert.False(t, result.Found)
}
```
