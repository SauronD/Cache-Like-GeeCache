package likecache

import pb "Cache-Like-GeeCache/likecachepb"

type PeerPicker interface {
	// 根据缓存键值对的key返回其对应的真实节点
	PeerPick(key string) (peer PeerGetter, ok bool)
}

type PeerGetter interface {
	// 当前真实节点根据/Group/key返回结果
	// Get(groupName string, key string) ([]byte, error)
	Get(in *pb.Request, out *pb.Response) error
}
