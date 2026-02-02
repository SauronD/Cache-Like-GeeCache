Feat1:singleflight优化缓存击穿、缓解缓存雪崩、处理不了缓存雪崩
singleflight实现了每个节点在向同一个节点请求一个相同的key时，这个节点并发的发送请求会被阻塞，等待当前的请求完成，避免一个key失效时，并发请求全部压到DB，造成数据库的压力过大的情况。也就是说被请求的节点在同一个时刻最多接收到当前所有节点数量个请求，启动相同数量个协程去内存中读取缓存的数据或从数据库中拉数据，这些请求同样会被singleflight阻塞，一瞬间最多只有一个协程在真正读取缓存数据或从数据库拉数据。
因此有效解决了缓存击穿问题，因为失效的key只会有一个DB连接获取数据。缓解了缓存雪崩问题，如前所述，每个失效key都只会请求一次，减少了并发请求。
但对于缓存穿透，比如构造海量不存在的key，被映射到不同节点上，每个节点都并发向db请求不同的key，singleflight就不起作用了。


New Feat1:peer Regist
实现一个Group内节点的注册、退出
初步实现：创建一个Registry Server的协程监听一个端口，每个节点在启动上向其注册并在存活期间定期向registry server发送报文，再定期轮询registry server拿到当前存活的节点列表，并重构哈希环

如何重构哈希环？

多个Group如何启动、管理？



New Feat2:热点key的缓存问题——




New Feat3:通信协议换RPC



New Feat4:LRUCache的分段锁
访问一个节点中的LRUCache中的key-value时，不同key竞争同一把互斥锁，把LRUCache分段，对每个段分别上锁

New Feat5:sync.Pool复用对象
临时对象的复用，避免GC的频繁触发影响性能