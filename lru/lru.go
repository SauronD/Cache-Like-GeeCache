package lru
// 哈希表+双向链表实现一个LRUCache

import(
	"container/list"
)
// 键值对缓存，为什么要在链表节点中继续保存key?
type entry struct {
	key string 
	value Value
}
// Value接口类型，any类型，但需要能够返回其占用内存大小
type Value interface {
	Len()int
}

type LRUCache struct {
	cache map[string]*list.Element
	ll *lisr.List
	maxBytes int64
	usedBytes int64
	// 删除键值对时的回调函数
	OnEvicted func(key string, value Value)
}

// Get
func(this *LRUCache)Get(key string)(Value,bool){
	if ele,ok:=this.cache[key];ok{
		
		// 将节点移动至头部：
		this.ll.MoveToFront(ele)
		kv:=ele.Value.(*entry)
		return kv.value,true
	}
	// 不存在这一键值对：
	return nil,false

}
// RemoveOldet
func(this *LRUCache)RemoveOldet(){
	ele:=this.ll.Back()
	if ele!=nil {
		kv := ele.Value.(*entry)
		this.usedBytes-=int64(len(kv.key))+int64(kv.value.Len())
		this.ll.Remove(ele)
		delete(this.cache,kv.key)
	}
}



// Add
func(this *LRUCache)Add(key string,value Value){
	if ele,ok:=this.cache[key];ok{
		// 移动至头部
		this.ll.MoveToFront(ele)
		kv:=ele.Value.(*entry)
		// 计算现在使用的内存大小，因为value可以是任意类型，所以可能会变化
		this.usedBytes+=int64(value.Len())-int64(kv.value.Len())
		kv.value=value
	}else{
		// 不存在则新建节点并移动至头部
		ele:=this.ll.PushFront(&entry{key,value})
		this.cache[key]=ele
		this.usedBytes+=int64(len(key))+int64(value.Len())
	}
	
	// 如果超过预设置内存空间，则需要删除链表末尾的节点
	for this.usedBytes>this.maxBytes {
		this.RemoveOldet()
	}
	

}
// Len
func (this *LRUCache) Len() int {
	return this.ll.Len()
}

func New(maxBytes int64,OnEvicted func(key string, value Value))*LRUCache{
	return &LRUCache{
		cache: map[string]*list.Element{},
		ll: list.New(),
		maxBytes:maxBytes,
		usedBytes:0,
		// 删除键值对时的回调函数
		OnEvicted:OnEvicted
	}
}
