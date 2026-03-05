package main

import (
	"Cache-Like-GeeCache/likecache"
	"Cache-Like-GeeCache/registry"
	"flag"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"
)

var db = map[string]string{
	"Tom":  "630",
	"Jack": "589",
	"Sam":  "567",
}

var db2 = map[string]string{
	"Tom":  "16",
	"Jack": "17",
	"Sam":  "18",
}

var registryAddr = "http://localhost:9000"

// 配置所有Group的连接并进行bloomfilter的预热：
func initDB(p *likecache.HTTPPool, dbs map[string]map[string]string) {
	// 注册配置Group和db连接
	for groupName, db := range dbs {
		g, err := likecache.NewGroup(groupName, 8<<20, likecache.MapGetter(db))
		if err != nil {
			panic(err)
		}
		g.RegisterPeers(p)

		// 预热操作：
		keys := []string{}
		for key := range db {
			keys = append(keys, key)
		}
		likecache.WarmUp(g, keys)
	}

}

func main() {
	TestMultipleGroup()
}
func TestSingleGroup() {
	var port int
	var api bool
	var registry bool
	flag.IntVar(&port, "p", -1, "Cache Server Port")
	flag.BoolVar(&api, "api", false, "If Create API Server")
	flag.BoolVar(&registry, "r", false, "If Create Registry Server")
	flag.Parse()

	if registry {
		log.Fatalf("[Regisrey Server] error : %s", CreateRegistryServer().Error())
	}
	if port == -1 {
		log.Fatal("flag -p is required")
	}
	// db只会被并发读，golang中的map并发读是安全的
	g, err := likecache.NewGroup("scores", 2<<10, likecache.GetterFunc(func(key string) ([]byte, error) {
		if value, ok := db[key]; ok {
			return []byte(value), nil
		}
		return nil, fmt.Errorf("[%s] not exists", key)
	}))
	if err != nil {
		log.Fatalln(err.Error())
	}
	if api {
		go CreateAPIServer()
	}
	CreateCacheServer(g, port)
}

func TestMultipleGroup() {
	var port int
	var api bool
	var regis bool
	flag.IntVar(&port, "p", -1, "Cache Server Port")
	flag.BoolVar(&api, "api", false, "If Create API Server")
	flag.BoolVar(&regis, "r", false, "If Create Registry Server")
	flag.Parse()

	if regis {
		go func() { log.Printf("[Regisrey Server] error : %s", CreateRegistryServer().Error()) }()
	}
	if port == -1 {
		log.Fatal("flag -p is required")
	}

	httpPool := likecache.NewHTTPPool("http://localhost:"+strconv.Itoa(port), registryAddr+"/_likecache/registry")
	// 创建每个节点的所有Group:
	initDB(httpPool, map[string]map[string]string{"scores": db, "age": db2})
	if api {
		go CreateAPIServer()
	}
	registry.Heartbeat(registryAddr+"/_likecache/registry", "http://localhost:"+strconv.Itoa(port), 10.0*time.Second)
	// 轮询当前存活节点
	go httpPool.StartSyncLoop()
	log.Fatalf("[Cache Server:%d] error: %s", port, http.ListenAndServe("localhost:"+strconv.Itoa(port), httpPool))
}

func CreateRegistryServer() error {
	registry.HandleHTTP()

	return http.ListenAndServe(registryAddr[7:], nil)

}
func CreateCacheServer(g *likecache.Group, port int) {

	httpPool := likecache.NewHTTPPool("http://localhost:"+strconv.Itoa(port), registryAddr+"/_likecache/registry")
	g.RegisterPeers(httpPool)
	registry.Heartbeat(registryAddr+"/_likecache/registry", "http://localhost:"+strconv.Itoa(port), 10.0*time.Second)
	// 轮询当前存活节点
	go httpPool.StartSyncLoop()
	log.Fatalf("[Cache Server:%d] error: %s", port, http.ListenAndServe("localhost:"+strconv.Itoa(port), httpPool))
}
func CreateAPIServer() {
	// API Server默认在9999接口
	port := 9999

	apiMux := http.NewServeMux()

	apiMux.Handle("/api", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not Allowed,use GET:?group=&key=", http.StatusBadRequest)
			return
		}
		groupName := r.URL.Query().Get("group")
		g := likecache.GetGroup(groupName)
		if g == nil {
			http.Error(w, "No such group", http.StatusBadRequest)
			return
		}
		key := r.URL.Query().Get("key")

		val, err := g.Get(key)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Write(val.ByteSlice())
	}))

	log.Printf("[API Server: http://localhost:%d] is running", port)
	log.Printf("[API Server] error:%s", http.ListenAndServe("localhost:"+strconv.Itoa(port), apiMux).Error())

}
func TestClient() {

	// group相当于一个key-value表，每个节点都由一个进程管理其存储的key-value表中的部分数据
	group, err := likecache.NewGroup("scores", 2<<10, likecache.GetterFunc(func(key string) ([]byte, error) {
		log.Println("[SlowDB] search key", key)
		if value, ok := db[key]; ok {
			return []byte(value), nil
		}
		return nil, fmt.Errorf("[%s] not exists", key)
	}))
	if err != nil {
		log.Fatal("group create failed")
		return
	}
	var port int
	var api bool

	flag.IntVar(&port, "port", 8090, "Geecache server port")
	flag.BoolVar(&api, "api", false, "Start a api server?")
	flag.Parse()

	var CreateAPIServer = func(apiAddr string) {
		http.Handle("/api", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := r.URL.Query().Get("key")
			view, err := group.Get(key)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/octet-stream")
			w.Write(view.ByteSlice())
		}))
		log.Println("fontend server is running at", apiAddr)
		log.Fatal(http.ListenAndServe(apiAddr[7:], nil))
	}

	if api {
		go CreateAPIServer("http://localhost:9001")
	}

}
