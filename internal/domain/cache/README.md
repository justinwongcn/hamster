# Cache 缓存领域模型

缓存领域模型定义了缓存系统的核心业务概念、规则和行为。包含缓存实体、值对象、领域服务和仓储接口的定义。

## 📁 文件结构

- `repository.go` - 缓存仓储接口定义
- `value_objects.go` - 缓存相关值对象
- `entities.go` - 缓存实体定义
- `services.go` - 缓存领域服务
- `bloom_filter.go` - 布隆过滤器领域接口

## 🎯 核心概念

### 仓储模式

定义了多种缓存访问模式：

- **Repository**：基础缓存仓储
- **ReadThroughRepository**：读透模式
- **WriteThroughRepository**：写透模式
- **WriteBackRepository**：写回模式

### 值对象

不可变的业务概念：

- **CacheKey**：缓存键，包含验证规则
- **CacheValue**：缓存值，支持多种数据类型
- **Expiration**：过期时间，封装过期逻辑
- **CacheStats**：缓存统计信息

### 实体

具有唯一标识的业务对象：

- **Entry**：缓存条目，包含键值和元数据
- **CacheInstance**：缓存实例，管理多个条目

### 领域服务

包含业务逻辑的服务：

- **CacheService**：核心缓存服务
- **EvictionService**：淘汰策略服务
- **WriteBackService**：写回服务

## 🔧 使用示例

### 创建缓存键值对象

```go
// 创建缓存键
key, err := cache.NewCacheKey("user:123")
if err != nil {
    // 处理验证错误
}

// 创建缓存值
value, err := cache.NewCacheValue("John Doe")
if err != nil {
    // 处理验证错误
}

// 创建过期时间
expiration := cache.NewExpiration(time.Hour)
```

### 使用缓存实体

```go
// 创建缓存条目
entry := cache.NewEntry(key, value, expiration)

// 检查是否过期
if entry.IsExpired(time.Now()) {
    // 处理过期逻辑
}

// 更新访问时间
entry = entry.UpdateAccessTime(time.Now())
```

### 实现自定义仓储

```go
type MyCustomRepository struct {
    // 内部存储
}

func (r *MyCustomRepository) Get(ctx context.Context, key string) (any, error) {
    // 实现获取逻辑
    return nil, nil
}

func (r *MyCustomRepository) Set(ctx context.Context, key string, val any, expiration time.Duration) error {
    // 实现设置逻辑
    return nil
}

// 实现其他必需方法...
```

## ⚠️ 注意事项

### 1. 值对象不可变性

```go
// ❌ 错误：直接修改值对象
key.value = "new_value"

// ✅ 正确：通过方法创建新实例
newKey, err := cache.NewCacheKey("new_value")
```

### 2. 键的验证规则

- 键不能为空字符串
- 键长度不能超过250个字符
- 键不能包含控制字符

### 3. 过期时间处理

- 零值表示永不过期
- 负值会被拒绝
- 过期检查基于创建时间和当前时间

### 4. 并发安全

- 值对象是不可变的，天然线程安全
- 实体的状态变更需要在基础设施层保证线程安全
- 仓储实现必须考虑并发访问

## 🧪 测试指南

### 值对象测试

```go
func TestCacheKey(t *testing.T) {
    tests := []struct {
        name    string
        key     string
        wantErr bool
    }{
        {"正常键", "user:123", false},
        {"空键", "", true},
        {"长键", strings.Repeat("a", 251), true},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            _, err := cache.NewCacheKey(tt.key)
            if (err != nil) != tt.wantErr {
                t.Errorf("NewCacheKey() error = %v, wantErr %v", err, tt.wantErr)
            }
        })
    }
}
```

### 实体测试

```go
func TestEntry(t *testing.T) {
    key, _ := cache.NewCacheKey("test")
    value, _ := cache.NewCacheValue("data")
    expiration := cache.NewExpiration(time.Hour)
    
    entry := cache.NewEntry(key, value, expiration)
    
    // 测试过期检查
    assert.False(t, entry.IsExpired(time.Now()))
    assert.True(t, entry.IsExpired(time.Now().Add(2*time.Hour)))
}
```

## 🔄 扩展指南

### 添加新的缓存策略

1. **定义新接口**：

```go
type CustomCacheRepository interface {
    Repository
    CustomOperation(ctx context.Context, key string) error
}
```

2. **在基础设施层实现**：

```go
type CustomCacheImpl struct {
    // 实现细节
}

func (c *CustomCacheImpl) CustomOperation(ctx context.Context, key string) error {
    // 具体实现
    return nil
}
```

### 添加新的值对象

1. **定义值对象**：

```go
type CustomValue struct {
    data string
}

func NewCustomValue(data string) (CustomValue, error) {
    if data == "" {
        return CustomValue{}, errors.New("数据不能为空")
    }
    return CustomValue{data: data}, nil
}
```

2. **添加验证方法**：

```go
func (v CustomValue) IsValid() bool {
    return v.data != ""
}

func (v CustomValue) String() string {
    return v.data
}
```

## 📊 性能考虑

### 1. 内存使用

- 值对象设计要考虑内存占用
- 避免在值对象中存储大量数据
- 使用指针时要注意内存泄漏

### 2. 计算复杂度

- 键的哈希计算要高效
- 过期检查要避免频繁的时间计算
- 统计信息的计算要考虑性能影响

### 3. 垃圾回收

- 避免频繁创建临时对象
- 合理使用对象池
- 注意循环引用问题
