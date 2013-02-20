package main

import (
	"fmt"
	"github.com/daviddengcn/go-villa"
	"net/http"
	"strings"
)

const s_SUFFIX = ".gep"

type HandlerInfo struct {
	w    http.ResponseWriter
	r    *http.Request
	done chan bool
}

func (h *HandlerInfo) Done() {
	h.done <- true
}

var gepMapQueue chan HandlerInfo

type ReigsterInfo struct {
	path        string
	handleQueue chan HandlerInfo
}

var gRegisterQueue chan ReigsterInfo

func handleGepMap() {
	fmt.Println("Start handling GEP map...")
	queueMap := make(map[string]chan HandlerInfo)
	for {
		select {
		case info := <-gepMapQueue:
			fmt.Println("Mapping", info.r.URL.Path)
			q, ok := queueMap[info.r.URL.Path]
			if !ok {
				http.NotFound(info.w, info.r)
				info.Done()
			} else {
				q <- info
			}
			
		case info := <-gRegisterQueue:
			queueMap[info.path] = info.handleQueue
			fmt.Println("Path", info.path, "registered!")
		}
	}
}

func init() {
	gRegisterQueue = make(chan ReigsterInfo)
	gepMapQueue = make(chan HandlerInfo)
	go handleGepMap()
}

func gepPageHandler(path string, q chan HandlerInfo) {
	for info := range q {
		//fmt.Fprintf(info.w, "Gep page at " + path)
		http.ServeFile(info.w, info.r, webRoot.Join(path).S())
		info.Done()
	}
}

func registerPath(path string) {
	q := make(chan HandlerInfo)
	gRegisterQueue <- ReigsterInfo{path: path, handleQueue: q}
	
	go gepPageHandler(path, q)
}

func handleGep(w http.ResponseWriter, r *http.Request) {
	info := HandlerInfo{w: w, r: r, done: make(chan bool)}
	gepMapQueue <- info
	<-info.done
}

var mediaSuffixes []string = []string{
	".jpg", ".jpeg", ".jpe",
	".png",
	".gif",
	".webp",
	".zip",
	".css"}

var webRoot villa.Path = `./web`

func isMediaFile(lowerPath string) bool {
	for _, suf := range mediaSuffixes {
		if strings.HasSuffix(lowerPath, suf) {
			return true
		}
	}

	return false
}

func handler(w http.ResponseWriter, r *http.Request) {
	lowerPath := strings.ToLower(r.URL.Path)
	if strings.HasSuffix(lowerPath, s_SUFFIX) {
		handleGep(w, r)
		return
	}

	if isMediaFile(lowerPath) {
		http.ServeFile(w, r, webRoot.Join(r.URL.Path).S())
	}
}

func main() {
	registerPath("/index.gep")
	
	http.HandleFunc("/", handler)
	fmt.Println("Listening at 8080")
	http.ListenAndServe(":8080", nil)
}
