package likecache

type PeerPicker interface {
	// 根据key返回其对应的虚拟节点
	Pick(key string) (peer PeerGetter, ok bool)
}

type PeerGetter interface {
	// 根据真实节点和key返回缓存值(拷贝)
	Get(group *Group, key string) ([]byte, error)
}
