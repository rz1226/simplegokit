package httpkit

//设立简单的http服务，一般用来做测试，例如测试http客户端，等
//服务端单个请求不能超过60秒
import (
	"errors"
	"net/http"
	"sync"
	"time"
)

//NewSimpleHttpServer().Add("/",f ).Start("8080")

type SimpleHttpServer struct {
	mux *memux
}

func NewSimpleHttpServer() *SimpleHttpServer {
	ms := &SimpleHttpServer{}
	ms.mux = newmemux()
	return ms
}
func (ms *SimpleHttpServer) Add(path string, f func(w http.ResponseWriter, r *http.Request)) *SimpleHttpServer {
	ms.mux.AddFunc(path, f)
	return ms
}
func (ms *SimpleHttpServer) Start(port string) {
	mux := ms.mux
	server := http.Server{
		Addr:         "0.0.0.0:" + port,
		Handler:      mux,
		ReadTimeout:  time.Second * 60,
		WriteTimeout: time.Second * 60,
	}
	server.SetKeepAlivesEnabled(true)
	server.ListenAndServe()
}

type memux struct {
	roads map[string]func(w http.ResponseWriter, r *http.Request)
	mu    *sync.RWMutex
}

func newmemux() *memux {
	m := &memux{}
	m.mu = &sync.RWMutex{}
	m.roads = make(map[string]func(w http.ResponseWriter, r *http.Request), 100)
	return m
}

func (m *memux) AddFunc(path string, f func(w http.ResponseWriter, r *http.Request)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.roads[path] = f
}

func (m *memux) getf(path string) (func(w http.ResponseWriter, r *http.Request), error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	p, ok := m.roads[path]
	if ok {
		return p, nil
	}
	return nil, errors.New("no router")
}

func (m *memux) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f, err := m.getf(r.URL.Path)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	f(w, r)
	return
}

/*
func sayhelloName(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hello myroute!")
}
*/
