package likecache

import (
	"Cache-Like-GeeCache/lru"
	"sync"
)

// 添加互斥锁的LRUCache:一个节点的不同key的访问/添加是并发的
// 将单锁升级成分段锁：
type cache struct {
	shards []*shard
}

// cache中的分段数量：
const defaultShardCount = 256

func NewCache(maxBytes int64) *cache {
	c := &cache{
		shards: make([]*shard, defaultShardCount),
	}
	// 平均每个shard的大小是总容量/shard总数量
	shardMaxBytes := maxBytes / defaultShardCount
	for i := range c.shards {
		c.shards[i] = &shard{maxBytes: shardMaxBytes}
	}
	return c
}

// 计算不同key应该存放的shard:
func (c *cache) getShard(key string) *shard {
	// FNV-1a 32-bit初始偏移量
	var hash uint32 = 2166136261
	for i := 0; i < len(key); i++ {
		hash ^= uint32(key[i])
		hash *= 16777619
	}
	return c.shards[hash%uint32(defaultShardCount)]
}

// 对外暴露的add和get方法
// 它们不再自己加锁，而是将任务代理给算出来的目标shard
func (c *cache) add(key string, value ByteView) {
	shard := c.getShard(key)
	shard.add(key, value)
}

func (c *cache) get(key string) (value ByteView, ok bool) {
	shard := c.getShard(key)
	return shard.get(key)
}

// 增加分段锁机制：
type shard struct {
	mu       sync.Mutex
	lru      *lru.LRUCache
	maxBytes int64
}

// sync.Mutex是一个结构体，复制的锁状态和原来不同步，因此只能传引用来控制同一个互斥锁
func (s *shard) add(key string, value ByteView) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.lru == nil {
		s.lru = lru.New(s.maxBytes, nil)
	}
	s.lru.Add(key, value)
}

// ByteView的默认零值不是nil，是一个空结构体
func (s *shard) get(key string) (value ByteView, ok bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.lru != nil {
		if value, ok := s.lru.Get(key); ok {
			// Get()返回的是lru.Value类型，要作断言
			return value.(ByteView), ok
		}
	}
	return

}
