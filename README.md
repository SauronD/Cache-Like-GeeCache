# Cache-Like-GeeCache

> 一个类groupcache的分布式内存缓存中间件，基于对等的 P2P 架构，支持节点动态水平扩展。通过实现多级缓存隔离、一致性哈希路由寻址、并发访问控制以及注册中心服务发现，缓解高并发场景下的缓存击穿、缓存穿透及单节点热点倾斜等问题。

## 关键特性
- 节点注册与动态发现：自动处理节点的上/下线
- singleflight请求合并：同一Key的并发回源只执行一次，缓解缓存击穿。
- Bloom Filter防穿透：缓存未命中后先判定Key合法性，减少虚假Key回源概率。
- 双层缓存结构：新增hotcache对对热点Key进行概率性副本缓存，分散热点压力。
- 分段LRU锁优化：将缓存拆分为256个shard，降低全局锁竞争。
- Protobuf编解码：节点间通信使用protobuf序列化，减少传输体积。
- 内存复用：通过sync.Pool复用bytes.Buffer与[]byte，降低protobuf序列化/反序列化临时对象分配与GC压力。
- 多Group统一管理升级：将Group作为系统运行前的统一配置，在每个节点创建一致的Group集合与对应getter，避免按Group分别维护节点列表和哈希环。这样可以保证同一Key在所有节点上的路由语义一致，降低多业务场景下的管理复杂度。

## 运行

### 环境要求

- Go `1.23.5+`

### 启动一个本地集群（多节点 + 注册中心 + API）

```powershell
# 终端1：节点1（同时启动注册中心）
go run . -r -p 9001

# 终端2：节点2
go run . -p 9002

# 终端3：节点3
go run . -p 9003

# 终端4：节点4（提供API）
go run . -p 9004 -api
```

### 验证请求

API 入口：`GET http://localhost:9999/api?group=<group>&key=<key>`

示例：

```bash
curl "http://localhost:9999/api?group=scores&key=Tom"
curl "http://localhost:9999/api?group=age&key=Jack"
curl "http://localhost:9999/api?group=scores&key=Ghost"
```

说明：
- scores组内置Tom/Jack/Sam
- age组内置Tom/Jack/Sam
- Ghost预期触发布隆过滤器拦截（不存在 key）

### 运行测试

可选：
- Linux/macOS可直接运行项目自带集群脚本：`bash test.sh`
- Windows可按需参考并发压测脚本：`test.ps1`


## 致谢

本项目基于以下项目进行学习与扩展：

- geecache: https://github.com/geektutu/7days-golang/tree/master/gee-cache
- groupcache: https://github.com/golang/groupcache

---
Feat1:singleflight优化缓存击穿、缓解缓存雪崩、处理不了缓存穿透
singleflight实现了每个节点在向同一个节点请求一个相同的key时，这个节点并发的发送请求会被阻塞，等待当前的请求完成，避免一个key失效时，并发请求全部压到DB，造成数据库的压力过大的情况。也就是说被请求的节点在同一个时刻最多接收到当前所有节点数量个请求，启动相同数量个协程去内存中读取缓存的数据或从数据库中拉数据，这些请求同样会被singleflight阻塞，一瞬间最多只有一个协程在真正读取缓存数据或从数据库拉数据。
因此有效解决了缓存击穿问题，因为失效的key只会有一个DB连接获取数据。缓解了缓存雪崩问题，如前所述，每个失效key都只会请求一次，减少了并发请求。但对于缓存穿透，比如构造海量不存在的key，被映射到不同节点上，每个节点都并发向db请求不同的key，singleflight就不起作用了。
因此需要针对缓存穿透需要额外的防护——设置一个布隆过滤器处理，在缓存未命中后，向其他节点请求或从数据库里拉前先判断其key是否存在



New Feat1:peer Regist
实现一个节点的注册、退出
初步实现：创建一个Registry Server的协程监听一个端口，每个节点在启动上向其注册并在存活期间定期向registry server发送报文，再定期轮询registry server拿到当前存活的节点列表，并重构哈希环

如何重构哈希环？
A:全量更新


New Feat1:多Group的管理
多个Group如何启动、管理？
每个节点上的Group应该是相同的，避免为每个Group单独维护哈希环和存活列表，因此在运行前，就需要规定好所有的group和其对应的数据获取方法
如果要实现动态创建、删除一个Group，需要一个广播——同步机制，让所有存活节点都创建节点，否则还需要维护不同Group的哈希环以及相应物理节点的同步...


New Feat2:热点key的缓存问题——缓存雪崩优化
对于热点key的请求，在每个节点向其他节点请求时，用一个概率算法来决定是否将请求的节点在本地也保存一份，从而分散单个节点的请求处理压力()
将每个节点的cache分为两部分：maincache和hotcache，maincache保存该节点应该管理的数据(一致性哈希分配的数据)，hotcache处理从节点请求的hot key:每次向远程节点访问时，有1/x概率存入hotcache，对于经常访问的节点就有更大可能性被缓存
这样做有两方面的好处：
1、节点网络io压力，对于热点key数据存储在多个节点上，提升cache命中率，减少向热点key对应节点的网络请求；
2、缓存雪崩场景，就算一部分节点的key过期或者节点下线，还有剩余节点上缓存了数据，需要向对应节点发起请求的节点减少，对应节点向mysql请求的也就减少。


New Feat3:通信协议换RPC
其实不太需要啊，因为已经有了proto来压缩载荷   


New Feat4:LRUCache的分段锁
访问一个节点中的LRUCache中的key-value时，不同key竞争同一把互斥锁，把LRUCache分段，对每个段分别上锁。
具体实现为：
1、引入新数据结构shard代替原来的cache，即持有一把互斥锁；
2、原cache则是一个[]*shard，根据一个哈希算法把key映射到一个具体的shard上；
这样并发的请求就不会一直竞争同一把锁，只有映射到同一个shard上的请求会竞争同一把互斥锁，也就实现了锁粒度细化；
PS:引入多锁，会导致状态不一致的问题：比如统计一个Group中每个cache里的所有key数量/空间占用，如果开一把锁累加、开一把锁累加，之前累加过的锁状态变化，就会导致最后结果的不一致。如果对全局所有所有shard加锁，则会阻塞所有的读写请求。若是引入一个新的全局变量，还是需要竞争这个变量的锁，等于没有优化。

New Feat5:sync.Pool复用对象
sync.Pool+*[]byte/bytes.Buffer实现临时对象的复用，避免GC的频繁触发影响性能
应用在客户端向其他节点请求数据时的反序列过程和服务器端向客户端发数据时的序列化过程
