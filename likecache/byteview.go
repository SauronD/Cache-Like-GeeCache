package likecache

// 实现一个只读数据结构ByteView用来表示缓存值，对于一些引用类型的变量，比如切片，返回引用作为值，如果修改了，那么缓存内的值也被修改了
// 因此实现一个只读数据结构来提供一个拷贝，选择[]byte是为了能够支持任意的数据类型的存储，例如字符串、图片等。
// b即从缓存中存放的真实值，外界存入/读取的都是ByteView
type ByteView struct {
	b []byte
}

// 拷贝缓存值，并返回拷贝
func cloneBytes(bytes []byte) []byte {
	b := make([]byte, len(bytes))
	copy(b, bytes)
	return b
}

// 只返回一个拷贝，避免b被修改
func (v ByteView) ByteSlice() []byte {
	return cloneBytes(v.b)
}

func (v ByteView) Len() int {
	return len(v.b)
}

func (v ByteView) String() string {
	return string(v.b)
}
