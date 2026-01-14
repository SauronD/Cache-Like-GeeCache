package likecache

type PeerPicker interface {
	// 根据缓存键值对的key返回其对应的真实节点
	PeerPick(key string) (peer PeerGetter, ok bool)
}

type PeerGetter interface {
	// 根据真实节点和key返回缓存值(拷贝)
	Get(group *Group, key string) ([]byte, error)
}

