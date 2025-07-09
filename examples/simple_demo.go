package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/justinwongcn/hamster"
	"github.com/justinwongcn/hamster/cache"
	"github.com/justinwongcn/hamster/hash"
	"github.com/justinwongcn/hamster/lock"
)

func main() {
	fmt.Printf("Hamster 库版本: %s\n", hamster.GetVersion())
	
	// 测试缓存功能
	testCache()
	
	// 测试一致性哈希功能
	testConsistentHash()
	
	// 测试分布式锁功能
	testDistributedLock()
}

func testCache() {
	fmt.Println("\n=== 测试缓存功能 ===")
	
	// 创建缓存服务
	cacheService, err := hamster.NewCache(
		cache.WithMaxMemory(1024*1024), // 1MB
		cache.WithEvictionPolicy("lru"),
	)
	if err != nil {
		log.Printf("创建缓存服务失败: %v", err)
		return
	}

	ctx := context.Background()

	// 设置缓存
	err = cacheService.Set(ctx, "test_key", "test_value", time.Minute)
	if err != nil {
		log.Printf("设置缓存失败: %v", err)
		return
	}
	fmt.Println("✓ 缓存设置成功")

	// 获取缓存
	value, err := cacheService.Get(ctx, "test_key")
	if err != nil {
		log.Printf("获取缓存失败: %v", err)
		return
	}
	fmt.Printf("✓ 获取缓存成功: %v\n", value)

	// 删除缓存
	err = cacheService.Delete(ctx, "test_key")
	if err != nil {
		log.Printf("删除缓存失败: %v", err)
		return
	}
	fmt.Println("✓ 缓存删除成功")
}

func testConsistentHash() {
	fmt.Println("\n=== 测试一致性哈希功能 ===")
	
	// 创建一致性哈希服务
	hashService, err := hamster.NewConsistentHash(
		hash.WithReplicas(150),
		hash.WithSingleflight(true),
	)
	if err != nil {
		log.Printf("创建一致性哈希服务失败: %v", err)
		return
	}

	ctx := context.Background()

	// 添加节点
	peers := []hash.Peer{
		{ID: "server1", Address: "192.168.1.1:8080", Weight: 100},
		{ID: "server2", Address: "192.168.1.2:8080", Weight: 100},
	}
	
	err = hashService.AddPeers(ctx, peers)
	if err != nil {
		log.Printf("添加节点失败: %v", err)
		return
	}
	fmt.Printf("✓ 成功添加 %d 个节点\n", len(peers))

	// 选择节点
	peer, err := hashService.SelectPeer(ctx, "test_key")
	if err != nil {
		log.Printf("选择节点失败: %v", err)
		return
	}
	fmt.Printf("✓ 键 test_key -> 节点 %s (%s)\n", peer.ID, peer.Address)
}

func testDistributedLock() {
	fmt.Println("\n=== 测试分布式锁功能 ===")
	
	// 创建分布式锁服务
	lockService, err := hamster.NewDistributedLock(
		lock.WithDefaultExpiration(30*time.Second),
		lock.WithDefaultTimeout(5*time.Second),
	)
	if err != nil {
		log.Printf("创建分布式锁服务失败: %v", err)
		return
	}

	ctx := context.Background()
	lockKey := "test_lock"

	// 尝试获取锁
	lockInfo, err := lockService.TryLock(ctx, lockKey)
	if err != nil {
		log.Printf("获取锁失败: %v", err)
		return
	}
	fmt.Printf("✓ 成功获取锁: %s (值: %s)\n", lockInfo.Key, lockInfo.Value)

	// 注意：由于一些方法暂未实现，这里只测试基本的获取锁功能
	fmt.Println("✓ 分布式锁基本功能测试完成")
}
