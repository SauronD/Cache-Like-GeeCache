package likecache

import(
	"sync"
	"Cache-Like-GeeCache/lru"
)
// 添加互斥锁的LRUCache
type cache struct {
	mu         	sync.Mutex
	lru        	*lru.LRUCache
	maxBytes 	int64
}

// sync.Mutex是一个结构体，复制的锁状态和原来不同步，因此只能传引用来控制同一个互斥锁
func(c *cache)add(key string,value ByteView){
	c.mu.Lock()
	defer c.mu.Unlock() 	
	if c.lru == nil {
		c.lru=lru.New(c.maxBytes,nil)
	}
	c.lru.Add(key,value)
}
// ByteView的默认零值不是nil，是一个空结构体
func(c *cache)get(key string)(value ByteView,ok bool){
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.lru!=nil {
		if value,ok:=c.lru.Get(key);ok{
			// Get()返回的是lru.Value类型，要作断言
			return value.(ByteView),ok
		}
	}
	return 
	
}

