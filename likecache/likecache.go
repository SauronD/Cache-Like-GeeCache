package likecache

import (
	pb "Cache-Like-GeeCache/likecachepb"
	"Cache-Like-GeeCache/singleflight"
	"errors"
	"fmt"
	"log"
	"sync"
)

type Getter interface {
	Get(key string) ([]byte, error)
}

// 定义一个函数类型，并为其实现Get方法，外部简单使用时可直接GetterFunc(f)作为Getter，避免创建一个结构体
// 对于复杂情况，也可以定义复杂结构体，并为其实现Get方法，实现Getter接口
type GetterFunc func(key string) ([]byte, error)

func (f GetterFunc) Get(key string) ([]byte, error) {

	return f(key)
}

// likecache的主结构，负责与用户交互，并控制缓存值的存储、获取
type Group struct {
	name      string
	getter    Getter //getter负责从数据源获取数据，比如从数据库获得数据
	maincache *cache
	peers     PeerPicker          //peers.PeerPick(key)返回其应该问询的真实节点,peers在本项目中为*HTTPPool
	loader    *singleflight.Group //控制请求的并发，即如果进行了一次请求，则期间所有后续相同的请求都等待这一请求返回结果
	bf        *BloomFilter
}

var groups = map[string]*Group{}

var mu sync.RWMutex

// getter是从数据源获取数据的方法，比如从数据库中获取原始数据的方法；
func NewGroup(name string, maxBytes int64, getter Getter) (*Group, error) {
	if getter == nil {
		return nil, errors.New("GetterFunc is nil")
	}
	mu.Lock()
	defer mu.Unlock()
	g := &Group{
		name:      name,
		getter:    getter,
		maincache: NewCache(maxBytes),
		loader:    &singleflight.Group{},
		bf:        NewBloomFilter(100000, 0.01),
	}
	groups[name] = g
	return g, nil

}

// 预热group的bloomfilter：
func WarmUp(g *Group, keys []string) {
	for _, key := range keys {
		g.bf.Add(key)
	}
	log.Printf("[Group %s] Bloom Filter warmed up with %d keys", g.name, len(keys))
}
func (g *Group) RegisterNewKey(key string) {
	if g.bf != nil {
		g.bf.Add(key)
	}
}

func GetGroup(name string) *Group {
	mu.RLock()
	defer mu.RUnlock()
	return groups[name]
}

// 返回key的value：整个查询流程中的第一步
func (g *Group) Get(key string) (ByteView, error) {
	if g == nil {
		return ByteView{}, errors.New("nil Group")
	}
	if key == "" {
		return ByteView{}, errors.New("empty key")
	}

	// 先检查key是否在当前节点内存Cache中：此处会被并发请求
	if value, ok := g.maincache.get(key); ok {
		log.Println("[Cache] hit")
		return value, nil
	}
	// 向外查询前先检查bloomfilter：
	// 对于缓存穿透，虽然先检查maincache会导致锁竞争，但是实现了分段锁，理论上会被平分到256个分段锁的竞争上
	if g.bf != nil && !g.bf.Contains(key) {
		return ByteView{}, fmt.Errorf("bloom filter: key [%s] does not exist", key)
	}

	// key不存在，远程获取/回调函数getter获取数据：DB
	return g.load(key)
}

func (g *Group) RegisterPeers(peer PeerPicker) {
	if g.peers != nil {
		panic("RegisterPeerPicker called more than once")
	}
	g.peers = peer
}

// load 处理请求key不在当前节点的缓存中，需要向其他节点请求或拉数据库中的数据
// 注意这里有两次合并，一个是向其他节点发送请求时，一个是被请求节点从数据库拉数据时
// 并且只能合并相同key，不同key的获取不会被阻塞，依然是并发处理。
func (g *Group) load(key string) (ByteView, error) {
	view, err := g.loader.Do(key, func() (interface{}, error) {
		if g.peers != nil {
			// 查找key对应的物理节点地址：
			if peer, ok := g.peers.PeerPick(key); ok {
				if value, err := g.getFromPeer(peer, key); err == nil {
					return value, nil
				} else {
					log.Println("[LikeCache] Failed to get from peer", err)
				}
			}

		}
		// key对应节点是当前节点或从远程节点获取失败时，比如目标节点下线，也在当前节点处拉取数据并缓存
		return g.getLocally(key)
	})

	if err != nil {
		return ByteView{}, err
	}
	return view.(ByteView), nil
}

func (g *Group) getLocally(key string) (ByteView, error) {
	bytes, error := g.getter.Get(key)
	if error != nil {
		return ByteView{}, error
	}
	value := ByteView{cloneBytes(bytes)}
	// 将获取的数据添加到内存数据结构Cache中
	g.populateCache(key, value)
	return value, nil
}
func (g *Group) populateCache(key string, value ByteView) {
	g.maincache.add(key, value)
}

func (g *Group) getFromPeer(peer PeerGetter, key string) (ByteView, error) {
	// 构造序列化查询请求
	req := &pb.Request{
		Group: g.name,
		Key:   key,
	}
	res := &pb.Response{}
	// Get中进行了反序列化
	err := peer.Get(req, res)
	if err != nil {
		return ByteView{}, err
	}
	return ByteView{b: res.Value}, nil
}
