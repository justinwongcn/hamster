# Domain 领域层

领域层是 Hamster 项目的核心，包含了业务逻辑、实体、值对象和领域服务。遵循领域驱动设计（DDD）原则，确保业务逻辑的纯净性和可测试性。

## 📁 包结构

```
domain/
├── cache/                     # 缓存领域模型
│   ├── bloom_filter.go       # 布隆过滤器领域接口和值对象
│   ├── entities.go           # 缓存实体定义
│   ├── repository.go         # 缓存仓储接口
│   ├── services.go           # 缓存领域服务
│   └── value_objects.go      # 缓存值对象
├── consistent_hash/           # 一致性哈希领域模型
│   └── consistent_hash.go    # 一致性哈希接口和值对象
├── errs/                      # 领域错误定义
│   └── errors.go             # 通用错误类型
├── lock/                      # 分布式锁领域模型
│   └── distributed_lock.go   # 分布式锁接口和值对象
└── tools/                     # 工具类
    ├── lru.go                # LRU算法实现
    └── lru_test.go           # LRU测试
```

## 🎯 设计原则

### 1. 依赖倒置原则

- 领域层不依赖任何外部框架或基础设施
- 通过接口定义与外部的交互契约
- 基础设施层实现领域层定义的接口

### 2. 单一职责原则

- 每个包专注于一个特定的业务领域
- 实体、值对象、服务各司其职
- 避免跨领域的直接依赖

### 3. 开闭原则

- 通过接口扩展功能，而不修改现有代码
- 支持多种实现策略（如不同的缓存策略、哈希算法等）

## 🏗️ 核心概念

### 实体（Entities）

具有唯一标识的业务对象，生命周期内标识不变：

- `CacheInstance`：缓存实例实体
- `Entry`：缓存条目实体

### 值对象（Value Objects）

不可变的业务概念，通过属性值来标识：

- `CacheKey`、`CacheValue`：缓存键值对象
- `HashKey`、`PeerInfo`：哈希相关值对象
- `LockKey`、`LockValue`：锁相关值对象

### 领域服务（Domain Services）

包含不属于特定实体或值对象的业务逻辑：

- `CacheService`：缓存业务逻辑
- `EvictionService`：淘汰策略服务
- `WriteBackService`：写回服务

### 仓储接口（Repository Interfaces）

定义数据访问的抽象：

- `Repository`：基础缓存仓储
- `ReadThroughRepository`：读透仓储
- `WriteThroughRepository`：写透仓储
- `WriteBackRepository`：写回仓储

## 🔧 使用指南

### 1. 扩展新的缓存策略

```go
// 1. 在 repository.go 中定义新接口
type CustomCacheRepository interface {
    Repository
    CustomMethod(ctx context.Context, key string) error
}

// 2. 在基础设施层实现接口
type CustomCacheImpl struct {
    // 实现细节
}

func (c *CustomCacheImpl) CustomMethod(ctx context.Context, key string) error {
    // 具体实现
    return nil
}
```

### 2. 添加新的值对象

```go
// 在 value_objects.go 中添加
type NewValueObject struct {
    value string
}

func NewNewValueObject(value string) (NewValueObject, error) {
    // 验证逻辑
    if value == "" {
        return NewValueObject{}, errors.New("值不能为空")
    }
    return NewValueObject{value: value}, nil
}
```

## ⚠️ 注意事项

### 1. 依赖方向

- 领域层不能依赖应用层、基础设施层或接口层
- 只能依赖标准库和其他领域包
- 外部依赖通过接口抽象

### 2. 不可变性

- 值对象必须是不可变的
- 实体的标识不能改变
- 状态变更通过方法返回新实例

### 3. 验证规则

- 所有业务规则验证都在领域层进行
- 构造函数必须验证输入参数
- 不允许创建无效的领域对象

### 4. 错误处理

- 使用领域特定的错误类型
- 错误信息要清晰描述业务含义
- 避免暴露技术实现细节

## 🧪 测试策略

领域层的测试主要在基础设施层进行，因为：

1. 领域层主要是接口和值对象定义
2. 具体的业务逻辑在基础设施层实现
3. 通过实现测试来验证领域规则

测试重点：

- 值对象的验证逻辑
- 实体的业务规则
- 领域服务的业务逻辑
- 接口契约的正确性

## 🔄 演进策略

1. **向后兼容**：新增功能通过扩展接口实现
2. **渐进式重构**：逐步优化现有设计，避免大规模重写
3. **文档同步**：代码变更时同步更新文档
4. **版本管理**：重大变更通过版本号体现
