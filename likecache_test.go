package likecache

import(
	"testing"
	"reflect"
	"fmt"
	"log"
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
	test,_:=NewGroup("scores",2<<10,GetterFunc(
		func(key string)([]byte,error){
			log.Println("[SlowDB] search key", key)
			if val,ok:=db[key];ok {
				loadCounts[key]++
				return []byte(val),nil 
			}
			return nil,fmt.Errorf("key:%s not exists\n",key)
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