package singleflight

import (
	"sync"
)

// call代表一个正在进行的请求，每个正在进行的请求都会记录到对应Group的哈希表m中
type call struct {
	wg  sync.WaitGroup
	val interface{}
	err error
}

type Group struct {
	mu sync.Mutex
	m  map[string]*call
}

// f是一个func()(any,error)，是call对应的请求函数,f的结果记录在call结构体中
func (g *Group) Do(key string, f func() (interface{}, error)) (interface{}, error) {
	g.mu.Lock()
	if g.m == nil {
		g.m = make(map[string]*call)
	}

	// 如果对应key的请求正在进行，则等待请求完成
	if c, ok := g.m[key]; ok {
		g.mu.Unlock()
		c.wg.Wait()

		return c.val, c.err
	}
	c := new(call)
	c.wg.Add(1)
	g.m[key] = c
	g.mu.Unlock()
	// f()可能很耗时，因此先解锁
	c.val, c.err = f()
	c.wg.Done()

	g.mu.Lock()
	delete(g.m, key)
	g.mu.Unlock()
	return c.val, c.err
}
