package likecache

import "math"

type BloomFilter struct {
	// 底层位图:[]uint64模拟连续bit数组，golang里没有单bit数据结构
	bitset []uint64
	// 位图的总bit数
	m uint
	// 哈希函数个数
	k uint
}

func NewBloomFilter(n uint, p float64) *BloomFilter {
	// 计算期望位图大小：
	m := uint(math.Ceil(-(float64(n) * math.Log(p)) / math.Pow(math.Log(2), 2)))
	// 计算期望哈希函数个数：
	k := uint(math.Ceil((float64(m) / float64(n)) * math.Log(2)))
	size := (m + 63) / 64

	return &BloomFilter{
		bitset: make([]uint64, size),
		m:      m,
		k:      k,
	}

}

// FNV-1a 64bit哈希算法：直接读string的值，避免[]byte拷贝和GC开销
func inlineFNV64a(key string) uint64 {
	// 64位 FNV offset basis
	var hash uint64 = 14695981039346656037
	for i := 0; i < len(key); i++ {
		hash ^= uint64(key[i])
		// 64位FNV prime
		hash *= 1099511628211
	}
	return hash
}

// 向bf中添加一个key，采取Kirsch-Mitzenmacher优化：用1次哈希计算+k次整数加法替代k次哈希计算
func (bf *BloomFilter) Add(key string) {
	hash64 := inlineFNV64a(key)
	// h1为hash64低32位，h2为高32位
	h1 := uint32(hash64 & 0xffffffff)
	h2 := uint32(hash64 >> 32)
	for i := uint(0); i < bf.k; i++ {
		// 计算第i个哈希函数对应的位图位置:Hi=(h1+h2*i)mod m
		bitPosition := (uint(h1) + i*uint(h2)) % bf.m

		// 对应bit：bf.bitset[sliceIndex]第bitOffset位
		sliceIndex := bitPosition / 64
		bitOffset := bitPosition % 64

		// 使用掩码+或运算将该位置为 1
		bf.bitset[sliceIndex] |= (1 << bitOffset)
	}
}

// Contains 检查一个key是否可能存在
func (bf *BloomFilter) Contains(key string) bool {
	hash64 := inlineFNV64a(key)
	// Kirsch-Mitzenmacher优化
	h1 := uint32(hash64 & 0xffffffff)
	h2 := uint32(hash64 >> 32)

	for i := uint(0); i < bf.k; i++ {
		bitPosition := (uint(h1) + i*uint(h2)) % bf.m
		sliceIndex := bitPosition / 64
		bitOffset := bitPosition % 64

		// AND运算检查该位是否为0：只要有任意一个哈希函数算出来的位置是0，说明不存在该key
		if (bf.bitset[sliceIndex] & (1 << bitOffset)) == 0 {
			return false
		}
	}
	// 所有k个位置全都是1，大概率存在
	return true
}
