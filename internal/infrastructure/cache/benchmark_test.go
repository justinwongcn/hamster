package cache

import (
	"context"
	"fmt"
	"testing"
	"time"
)

// BenchmarkLRUPolicy_KeyAccessed 测试LRU策略的KeyAccessed性能
func BenchmarkLRUPolicy_KeyAccessed(b *testing.B) {
	ctx := context.Background()
	policy := NewLRUPolicy()

	for i := 0; b.Loop(); i++ {
		key := fmt.Sprintf("key%d", i%1000) // 循环使用1000个key
		_ = policy.KeyAccessed(ctx, key)
	}
}

// BenchmarkLRUPolicy_Evict 测试LRU策略的Evict性能
func BenchmarkLRUPolicy_Evict(b *testing.B) {
	ctx := context.Background()
	policy := NewLRUPolicy()

	// 预先添加一些key
	for i := range 1000 {
		_ = policy.KeyAccessed(ctx, fmt.Sprintf("key%d", i))
	}

	for i := 0; b.Loop(); i++ {
		_, _ = policy.Evict(ctx)
		// 重新添加一个key以保持策略中有数据
		_ = policy.KeyAccessed(ctx, fmt.Sprintf("key%d", i+1000))
	}
}

// BenchmarkFIFOPolicy_KeyAccessed 测试FIFO策略的KeyAccessed性能
func BenchmarkFIFOPolicy_KeyAccessed(b *testing.B) {
	ctx := context.Background()
	policy := NewFIFOPolicy()

	for i := 0; b.Loop(); i++ {
		key := fmt.Sprintf("key%d", i%1000) // 循环使用1000个key
		_ = policy.KeyAccessed(ctx, key)
	}
}

// BenchmarkFIFOPolicy_Evict 测试FIFO策略的Evict性能
func BenchmarkFIFOPolicy_Evict(b *testing.B) {
	ctx := context.Background()
	policy := NewFIFOPolicy()

	// 预先添加一些key
	for i := range 1000 {
		_ = policy.KeyAccessed(ctx, fmt.Sprintf("key%d", i))
	}

	
	for i := 0; b.Loop(); i++ {
		_, _ = policy.Evict(ctx)
		// 重新添加一个key以保持策略中有数据
		_ = policy.KeyAccessed(ctx, fmt.Sprintf("key%d", i+1000))
	}
}

// BenchmarkRandomPolicy_KeyAccessed 测试随机策略的KeyAccessed性能
func BenchmarkRandomPolicy_KeyAccessed(b *testing.B) {
	ctx := context.Background()
	policy := NewRandomPolicy()

	
	for i := 0; b.Loop(); i++ {
		key := fmt.Sprintf("key%d", i%1000) // 循环使用1000个key
		_ = policy.KeyAccessed(ctx, key)
	}
}

// BenchmarkRandomPolicy_Evict 测试随机策略的Evict性能
func BenchmarkRandomPolicy_Evict(b *testing.B) {
	ctx := context.Background()
	policy := NewRandomPolicy()

	// 预先添加一些key
	for i := 0; i < 1000; i++ {
		_ = policy.KeyAccessed(ctx, fmt.Sprintf("key%d", i))
	}

	
	for i := 0; b.Loop(); i++ {
		_, _ = policy.Evict(ctx)
		// 重新添加一个key以保持策略中有数据
		_ = policy.KeyAccessed(ctx, fmt.Sprintf("key%d", i+1000))
	}
}

// BenchmarkMaxMemoryCache_Set 测试MaxMemoryCache的Set性能
func BenchmarkMaxMemoryCache_Set(b *testing.B) {
	ctx := context.Background()

	benchmarks := []struct {
		name   string
		policy EvictionPolicy
	}{
		{"LRU", NewLRUPolicy()},
		{"FIFO", NewFIFOPolicy()},
		{"Random", NewRandomPolicy()},
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			cache := NewMaxMemoryCache(1024*1024, // 1MB
				NewBuildInMapCache(time.Hour), // 使用长间隔避免清理干扰
				bm.policy)

			b.ResetTimer()
			for i := 0; b.Loop(); i++ {
				key := fmt.Sprintf("key%d", i%1000)
				value := fmt.Appendf(nil, "value%d", i)
				_ = cache.Set(ctx, key, value, 0)
			}
		})
	}
}

// BenchmarkMaxMemoryCache_Get 测试MaxMemoryCache的Get性能
func BenchmarkMaxMemoryCache_Get(b *testing.B) {
	ctx := context.Background()

	benchmarks := []struct {
		name   string
		policy EvictionPolicy
	}{
		{"LRU", NewLRUPolicy()},
		{"FIFO", NewFIFOPolicy()},
		{"Random", NewRandomPolicy()},
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			cache := NewMaxMemoryCache(1024*1024, // 1MB
				NewBuildInMapCache(0), // 禁用清理以避免干扰
				bm.policy)

			// 预先填充缓存
			for i := range 1000 {
				key := fmt.Sprintf("key%d", i)
				value := fmt.Appendf(nil, "value%d", i)
				_ = cache.Set(ctx, key, value, 0)
			}

			b.ResetTimer()
			for i := 0; b.Loop(); i++ {
				key := fmt.Sprintf("key%d", i%1000)
				_, _ = cache.Get(ctx, key)
			}
		})
	}
}

// BenchmarkPolicyComparison 比较不同策略在混合操作下的性能
func BenchmarkPolicyComparison(b *testing.B) {
	ctx := context.Background()

	benchmarks := []struct {
		name   string
		policy EvictionPolicy
	}{
		{"LRU", NewLRUPolicy()},
		{"FIFO", NewFIFOPolicy()},
		{"Random", NewRandomPolicy()},
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			policy := bm.policy

			// 预先添加一些key
			for i := range 100 {
				_ = policy.KeyAccessed(ctx, fmt.Sprintf("key%d", i))
			}

			b.ResetTimer()
			for i := 0; b.Loop(); i++ {
				switch i % 4 {
				case 0: // 25% KeyAccessed
					_ = policy.KeyAccessed(ctx, fmt.Sprintf("key%d", i%200))
				case 1: // 25% Has
					_, _ = policy.Has(ctx, fmt.Sprintf("key%d", i%200))
				case 2: // 25% Evict
					_, _ = policy.Evict(ctx)
					_ = policy.KeyAccessed(ctx, fmt.Sprintf("key%d", i+200))
				case 3: // 25% Remove
					_ = policy.Remove(ctx, fmt.Sprintf("key%d", i%200))
				}
			}
		})
	}
}

// BenchmarkConcurrentAccess 测试并发访问性能
func BenchmarkConcurrentAccess(b *testing.B) {
	ctx := context.Background()

	benchmarks := []struct {
		name   string
		policy EvictionPolicy
	}{
		{"LRU", NewLRUPolicy()},
		{"FIFO", NewFIFOPolicy()},
		{"Random", NewRandomPolicy()},
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			policy := bm.policy

			// 预先添加一些key
			for i := range 100 {
				_ = policy.KeyAccessed(ctx, fmt.Sprintf("key%d", i))
			}

			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				i := 0
				for pb.Next() {
					key := fmt.Sprintf("key%d", i%200)
					switch i % 3 {
					case 0:
						_ = policy.KeyAccessed(ctx, key)
					case 1:
						_, _ = policy.Has(ctx, key)
					case 2:
						_, _ = policy.Size(ctx)
					}
					i++
				}
			})
		})
	}
}
