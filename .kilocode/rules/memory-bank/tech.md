# 技术文档

## 技术栈
- 编程语言：Go 1.24
- 数据存储：内置map
- 并发控制：细粒度读写锁
- 垃圾回收：FIFO + LRU策略
- 回调机制：支持自定义过期回调处理
- 批量写入：支持并发批量操作

## 开发环境
- Go版本：>=1.24
- 构建工具：go build
- 测试框架：testing
- 性能测试：benchmark

## 依赖管理
- 使用Go Modules进行依赖管理
- 主要依赖库：
  - github.com/sirupsen/logrus
  - github.com/stretchr/testify

## 编码规范
- 遵循Go官方代码规范
- 使用golangci-lint进行代码检查
- 单元测试覆盖率要求：>=80%