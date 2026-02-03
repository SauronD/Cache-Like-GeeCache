package likecache

import (
	"fmt"
)

// 将map转换为GetterFunc(工厂方法)
func MapGetter(db map[string]string) GetterFunc {
	return GetterFunc(func(key string) ([]byte, error) {
		if val, ok := db[key]; ok {
			return []byte(val), nil
		}
		return nil, fmt.Errorf("[%s] not exists", key)
	})
}
