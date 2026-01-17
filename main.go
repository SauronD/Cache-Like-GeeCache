package main

import (
	"Cache-Like-GeeCache/likecache"
	"flag"
	"fmt"
	"log"
	"net/http"
)

var db = map[string]string{
	"Tom":  "630",
	"Jack": "589",
	"Sam":  "567",
}

func main() {
	TestClient()
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
	// 所有的节点
	node := map[int]string{
		9090: "http://localhost:9090",
		9091: "http://localhost:9091",
		9092: "http://localhost:9092",
		9093: "http://localhost:9093",
	}

	var CreatCacheServer = func(addr string, addrs ...string) {
		// addr like :http://localhost:8080，是这个节点的地址
		httppool := likecache.NewHTTPPool(addr) //httppool的baseURL:http://localhost:8080/_likecache/
		// addrs是一个Group下的所有节点
		group.RegisterPeers(httppool)
		httppool.Set(addrs...)
		log.Println("likecache is running at", addr)
		log.Fatal(http.ListenAndServe(addr[7:], httppool))
	}
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
	var port int
	var api bool

	flag.IntVar(&port, "port", 8090, "Geecache server port")
	flag.BoolVar(&api, "api", false, "Start a api server?")
	flag.Parse()

	if api {
		go CreateAPIServer("http://localhost:9001")
	}
	addrs := []string{}
	for _, add := range node {
		addrs = append(addrs, add)
	}
	CreatCacheServer(node[port], addrs...)

}
