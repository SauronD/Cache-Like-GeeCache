package likecache

import (
	"bytes"
	"sync"
)

// 全局对象池来优化节点请求间的序列/反序列化的GC性能
var bufferPool = sync.Pool{
	New: func() interface{} {
		buf := new(bytes.Buffer)
		buf.Grow(1024) //1KB
		return buf
	},
}

// 处理append场景下的问题：append(buf,)扩容并返回新切片，buf并没有扩容，也没有起到复用减少GC的效果
var sliceBytesPool = sync.Pool{
	New: func() interface{} {
		buf := make([]byte, 0, 1024)
		return &buf
	},
}
