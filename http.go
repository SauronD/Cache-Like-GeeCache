package likecache

// 基于HTTP实现与其他节点的通信
import (
	"fmt"
	"log"
	"net/http"
	"strings"
)

const defaultBasePath = "/_likecache/"

type HTTPPool struct {
	self     string
	basePath string
}

func NewHTTPPool(self string) *HTTPPool {
	return &HTTPPool{
		self,
		defaultBasePath,
	}
}
func (p *HTTPPool) Log(format string, v ...interface{}) {
	log.Printf("[Server %s] %s", p.self, fmt.Sprintf(format, v...))
}

// 通信服务器端，http://xxx.xxx.xxx.xxx:port/basePath/groupName/key
func (p *HTTPPool) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !strings.HasPrefix(r.URL.Path, p.basePath) {
		panic("HTTPPool serving unexpected path: " + r.URL.Path)
	}
	p.Log("%s %s", r.Method, r.URL.Path)
	// 期望的URL:/<basepath>/<groupname>/<key>
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
	if error != nil {
		http.Error(w, error.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/octet-stream")
	// 返回的是缓存值的拷贝
	w.Write(view.ByteSlice())
}
