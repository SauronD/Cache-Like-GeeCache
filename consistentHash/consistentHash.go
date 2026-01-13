package consistenthash

import(

	"hash/crc32"
)

type Hash func([]byte)uint32

type Map struct {
	hash  Hash
	replicas int // 真实节点->虚拟节点的倍数
	keys []int // 模拟哈希环
	hashMAP map[int]string //key:虚拟节点,value:真实节点
}

func New(replicas int,f Hash)*Map{
	m:=&Map{
		hash:f,
		replicas:replicas,
		keys:[]int{},
		hashMAP,map[int]string{},
	}
	if f==nil {
		m.hash=crc32.ChecksumIEEE
	}
	return m
}

func(m *Map)Add(keys ...string){
	for _,key := range keys {
		for i:=0;i<m.replicas;i++ {
			hash:=int(m.hash([]byte(strconv.Itoa(i)+key)))
			m.keys=append(m.keys,hash)
			m.hashMAP[hash]=key
		}
	}
	// 将keys进行排序，才能实现O(logN)找到第一个大于等于val的节点
	sort.Ints(m.keys)
}
