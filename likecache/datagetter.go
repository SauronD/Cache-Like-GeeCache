package likecache

import (
	"fmt"
	"sync"
)

// 预注册的所有db连接
type getDB struct {
	rw        sync.RWMutex
	dbs       map[string]map[string]string
	dbgetters map[string]*GetterFunc
}

var DB = getDB{
	dbs:       map[string]map[string]string{},
	dbgetters: map[string]*GetterFunc{},
}

func InitDB(dbName string, db map[string]string) error {
	if db == nil {
		return fmt.Errorf("db is nil")
	}
	DB.rw.RLock()

	if _, ok := DB.dbs[dbName]; ok {
		// 已经注册此db
		DB.rw.RUnlock()
		return fmt.Errorf("%s already exists", dbName)
	}
	// 不存在，则需要写入新db(连接)：
	DB.rw.Lock()
	defer DB.rw.Unlock()
	DB.dbs[dbName] = db
	return nil
}

// 创建对应db的GetterFunc
func createDBGetter(dbName string) *GetterFunc {

}
