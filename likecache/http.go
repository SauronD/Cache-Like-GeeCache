package likecache

// 基于HTTP实现与其他节点的通信
import (
	"Cache-Like-GeeCache/consistenthash"
	pb "Cache-Like-GeeCache/likecachepb"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"google.golang.org/protobuf/proto"
)

const (
	defaultBasePath = "/_likecache/"
	defaultReplicas = 50
)

// HTTP服务器端
type HTTPPool struct {
	self     string //域名(ip)+端口号，ListenAndServe启动的地址
	basePath string // /_likecache/
	peers    *consistenthash.Map
	// 注册模式，每个HTTPGetter接口
	httpGetters map[string]*HTTPGetter
	mu          sync.Mutex
	// registry地址，请求可用节点
	registryAddr string
}

func NewHTTPPool(self, registryPath string) *HTTPPool {
	return &HTTPPool{
		self:         self,
		basePath:     defaultBasePath,
		registryAddr: registryPath,
		peers:        consistenthash.New(defaultReplicas, nil),
	}
}
func (p *HTTPPool) Log(format string, v ...interface{}) {
	log.Printf("[Server %s] %s", p.self, fmt.Sprintf(format, v...))
}

// 通信服务器端，收到Get请求到http://xxx.xxx.xxx.xxx:port/_likecache/groupName/key
func (p *HTTPPool) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !strings.HasPrefix(r.URL.Path, p.basePath) {
		panic("HTTPPool serving unexpected path: " + r.URL.Path)
	}
	p.Log("%s %s", r.Method, r.URL.Path)
	// 收到的的URL:/_likecache/<groupname>/<key>,去掉前缀后：<groupname>/<key>
	parts := strings.SplitN(r.URL.Path[len(p.basePath):], "/", 2)
	if len(parts) < 2 {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	groupName := parts[0]
	key := parts[1]
	group := GetGroup(groupName)
	if group == nil {
		http.Error(w, fmt.Sprintf("no group[%s]", groupName), http.StatusNotFound)
		return
	}
	view, error := group.Get(key)
	// 获取数据后序列化为protobuf格式
	body, err := proto.Marshal(&pb.Response{Value: view.ByteSlice()})
	if err != nil {
		http.Error(w, error.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/octet-stream")
	// 返回的是缓存值的拷贝
	w.Write(body)
}

// 向baseURL请求的功能：每个真实节点一个对应的HTTPGetter
type HTTPGetter struct {
	baseURL string
}

// 向节点请求Group:key，将返回值反序列化放入out中
func (h *HTTPGetter) Get(in *pb.Request, out *pb.Response) error {
	requestURL := fmt.Sprintf(
		"%v%v/%v",
		h.baseURL,
		// 对groupName和key进行转义
		url.QueryEscape(in.GetGroup()),
		url.QueryEscape(in.GetKey()),
	)
	res, err := http.Get(requestURL)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned: %v", res.Status)
	}
	bytes, err := io.ReadAll(res.Body)
	if err = proto.Unmarshal(bytes, out); err != nil {
		return fmt.Errorf("decoding response body: %v", err)
	}
	return nil
}

func (p *HTTPPool) PeerPick(key string) (peer PeerGetter, ok bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	// p.peers.Get涉及slice的二分查找，p.httpGetters涉及map的读
	// 这些操作都不是线程安全的，因此需要上锁
	if peer := p.peers.Get(key); peer != "" && peer != p.self {
		p.Log("Pick peer %s", peer)
		return p.httpGetters[peer], true
	}
	return
}

// 更新HTTPPool的节点列表：一个Group中的每个Node能够访问的其他Node的地址，包括其自己
func (p *HTTPPool) Set(peers ...string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.peers.Set(peers...)
	// 重新注册存活节点
	p.httpGetters = make(map[string]*HTTPGetter, len(peers))
	for _, peer := range peers {
		// peer:http://localhost:9090
		// basePath:/_likecache/
		p.httpGetters[peer] = &HTTPGetter{peer + p.basePath}
	}
	log.Printf("[HTTPPool] Sync peers success: %v", peers)

}

// 查询registryPath,并检查是否有变化
func (p *HTTPPool) updatePeers() {
	res, err := http.Get(p.registryAddr)
	defer func() { _ = res.Body.Close() }()
	if err != nil {
		log.Println("[HTTPPool] Sync error:", err.Error())
		return
	}
	alive := res.Header.Get("X-LikeCache-Servers")
	if alive == "" {
		log.Println("[HTTPPool] empty peers")
	}
	peers := strings.Split(alive, ",")
	// 更新哈希环
	p.Set(peers...)
}

// 每10s轮询获取存活节点
func (p *HTTPPool) StartSyncLoop() {
	p.updatePeers()
	ticker := time.NewTicker(10.0 * time.Second)
	for range ticker.C {
		p.updatePeers()
	}
}
