package consistenthash

import (
	"hash/crc32"
	"sort"
	"strconv"
	"sync"
)

type Hash func([]byte) uint32

type Map struct {
	// 不同key之间的读竞争和重构的写竞争
	rw       sync.RWMutex
	hash     Hash
	replicas int            // 真实节点->虚拟节点的倍数
	keys     []int          // 模拟哈希环
	hashMAP  map[int]string //key:虚拟节点,value:真实节点
}

func New(replicas int, f Hash) *Map {
	m := &Map{
		hash:     f,
		replicas: replicas,
		keys:     []int{},
		hashMAP:  map[int]string{},
	}
	if f == nil {
		m.hash = crc32.ChecksumIEEE
	}
	return m
}

func (m *Map) Add(keys ...string) {
	for _, key := range keys {
		for i := 0; i < m.replicas; i++ {
			hash := int(m.hash([]byte(strconv.Itoa(i) + key)))
			m.keys = append(m.keys, hash)
			m.hashMAP[hash] = key
		}
	}
	// 将keys进行排序，才能实现O(logN)找到第一个大于等于val的节点
	sort.Ints(m.keys)
}

// 缓存数据的key，计算其在哈希环上对应的虚拟节点哈希，并返回真实节点
func (m *Map) Get(key string) string {
	m.rw.RLock()
	defer m.rw.RUnlock()
	hash := m.hash([]byte(key))
	idx := sort.Search(len(m.keys), func(i int) bool { return m.keys[i] >= int(hash) })
	return m.hashMAP[m.keys[idx%len(m.keys)]]
}

// 全量创建一个新哈希环，因为每次轮询都会重新创建，因此需要加锁，避免同时主协程在Get()
func (m *Map) Set(peers ...string) {
	m.rw.Lock()
	defer m.rw.Unlock()
	hashloop := []int{}
	newMap := make(map[int]string)
	for _, peer := range peers {
		for i := 1; i <= m.replicas; i++ {
			hash := int(m.hash([]byte(strconv.Itoa(i) + peer)))
			hashloop = append(hashloop, hash)
			newMap[hash] = peer
		}
	}
	sort.Ints(hashloop)
	m.keys = hashloop
	m.hashMAP = newMap
}
