package registry

import (
	"log"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"
)

// 管理所有节点，
type Registry struct {
	//  注册节点的最后一次心跳时间
	servers map[string]time.Time
	mu      sync.Mutex
	timeout time.Duration
}

const (
	defaultPath = "/_likecache/registry"
	// 1min没心跳就下线节点
	defaultTimeOut = time.Minute * 1
)

func NewRegistry(timeout time.Duration) *Registry {
	return &Registry{
		servers: make(map[string]time.Time),
		timeout: timeout,
	}
}

// 更新节点时间戳
func (r *Registry) UpdateServer(addr string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.servers[addr] = time.Now()
}

// 获取所有存活节点
func (r *Registry) GetAliveServers() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	alive := []string{}
	for addr, t := range r.servers {
		// 永不超时或还未超时
		if r.timeout == 0 || t.Add(r.timeout).After(time.Now()) {
			alive = append(alive, addr)
		} else {
			// 删除超时节点
			delete(r.servers, addr)
		}
	}
	sort.Strings(alive)
	return alive
}

func (r *Registry) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case "GET":
		// 返回存活节点
		w.Header().Set("X-LikeCache-Servers", strings.Join(r.GetAliveServers(), ","))
	case "POST":
		// 接收心跳
		addr := req.Header.Get("X-LikeCache-Server")
		if addr == "" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		r.UpdateServer(addr)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

// 启动Registry服务
func HandleHTTP() {
	http.Handle(defaultPath, NewRegistry(defaultTimeOut))
	log.Println("Registry server started at", defaultPath)
}

// 节点上报心跳:每个节点存活期间，定时向Registry服务报告存活
func Heartbeat(registryURL, addr string, duration time.Duration) {
	if duration == 0 {
		duration = defaultTimeOut - time.Duration(10)*time.Second
	}
	// 定时器
	ticker := time.NewTicker(duration)
	go func() {
		sendHeartbeat(registryURL, addr)
		for range ticker.C {
			err := sendHeartbeat(registryURL, addr)
			if err != nil {
				log.Println("Heartbeat error:", err)
			}
		}
	}()
}

// 构造http包发送心跳包
func sendHeartbeat(registryURL, addr string) error {
	req, _ := http.NewRequest("POST", registryURL, nil)
	req.Header.Set("X-LikeCache-Server", addr)
	if _, err := http.DefaultClient.Do(req); err != nil {
		return err
	}
	return nil
}
