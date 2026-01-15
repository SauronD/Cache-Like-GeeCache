package likecache

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"reflect"
	"testing"
)

func TestGetter(t *testing.T) {
	var f Getter = GetterFunc(func(key string) ([]byte, error) {
		return []byte(key), nil
	})

	expect := []byte("key")
	if v, _ := f.Get("key"); !reflect.DeepEqual(v, expect) {
		t.Errorf("callback failed")
	}
}

var db = map[string]string{
	"Tom":  "630",
	"Jack": "589",
	"Sam":  "567",
}

func TestGet(t *testing.T) {

	loadCounts := make(map[string]int, len(db))
	test, _ := NewGroup("scores", 2<<10, GetterFunc(
		func(key string) ([]byte, error) {
			log.Println("[SlowDB] search key", key)
			if val, ok := db[key]; ok {
				loadCounts[key]++
				return []byte(val), nil
			}
			return nil, fmt.Errorf("key:%s not exists\n", key)
		}))
	for k, v := range db {
		if view, err := test.Get(k); err != nil || view.String() != v {
			t.Fatal("failed to get value of Tom")
		} // load from callback function
		if _, err := test.Get(k); err != nil || loadCounts[k] > 1 {
			t.Fatalf("cache %s miss", k)
		} // cache hit
	}

	if view, err := test.Get("unknown"); err == nil {
		t.Fatalf("the value of unknow should be empty, but %s got", view)
	}
}

func TestServer(t *testing.T) {
	NewGroup("scores", 2<<10, GetterFunc(
		func(key string) ([]byte, error) {
			log.Println("[SlowDB] search key", key)
			if v, ok := db[key]; ok {
				return []byte(v), nil
			}
			return nil, fmt.Errorf("%s not exist", key)
		}))

	addr := "localhost:9999"
	peers := NewHTTPPool(addr)
	log.Println("likecache is running at", addr)
	log.Fatal(http.ListenAndServe(addr, peers))
}

func TestClient(t *testing.T) {
	// group相当于一个key-value表，每个节点都由一个进程管理其存储的key-value表中的部分数据
	group, err := NewGroup("scores", 2<<10, GetterFunc(func(key string) ([]byte, error) {
		log.Println("[SlowDB] search key", key)
		if value, ok := db[key]; ok {
			return []byte(value), nil
		}
		return nil, errors.New(fmt.Sprintf("[%s] not exists", key))
	}))
	if err != nil {
		t.Fatal("group create failed")
	}
	// 所有的节点
	node := map[string]string{
		"8090": "http://localhost:8090",
		"8091": "http://localhost:8091",
		"8092": "http://localhost:8092",
		"8093": "http://localhost:8093",
	}

	group.RegisterPeers()
	var CreatCacheServer = func(addr string, addrs ...string) {
		// addr like :http://localhost:8080，是这个节点的地址
		httppool := NewHTTPPool(addr)
		// addrs是一个Group下的所有节点
		group.RegisterPeers(httppool)
		httppool.Set(addrs...)
		log.Println("likecache is running at", addr)
		log.Fatal(http.ListenAndServe(addr[7:], httppool))
	}
}
