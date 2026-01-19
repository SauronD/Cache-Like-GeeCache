package likecache

import (
	pb "Cache-Like-GeeCache/likecachepb"
	"Cache-Like-GeeCache/singleflight"
	"errors"
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
	maincache cache
	peers     PeerPicker //peers.PeerPick(key)返回其应该问询的真实节点,在本项目中为*HTTPPool
	loader    *singleflight.Group
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
		maincache: cache{maxBytes: maxBytes},
		loader:    &singleflight.Group{},
	}
	groups[name] = g
	return g, nil

}

func GetGroup(name string) *Group {
	mu.RLock()
	defer mu.RUnlock()
	return groups[name]
}

// 返回key
func (g *Group) Get(key string) (ByteView, error) {
	if g == nil {
		return ByteView{}, errors.New("nil Group")
	}
	if key == "" {
		return ByteView{}, errors.New("empty key")
	}
	// 先检查key是否在当前节点内存Cache中
	if value, ok := g.maincache.get(key); ok {
		log.Println("[Cache] hit")
		return value, nil
	}
	// key不存在，远程获取/回调函数getter获取数据
	return g.load(key)
}

func (g *Group) RegisterPeers(peer PeerPicker) {
	if g.peers != nil {
		panic("RegisterPeerPicker called more than once")
	}
	g.peers = peer
}

func (g *Group) load(key string) (ByteView, error) {
	view, err := g.loader.Do(key, func() (interface{}, error) {
		if g.peers != nil {
			if peer, ok := g.peers.PeerPick(key); ok {
				if value, err := g.getFromPeer(peer, key); err == nil {
					return value, nil
				} else {
					log.Println("LikeCache] Failed to get from peer", err)
				}
			}

		}
		// 单机缓存直接调用getLocally
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
