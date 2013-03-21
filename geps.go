package main

import (
	"fmt"
	"github.com/daviddengcn/go-ljson-conf"
	"github.com/daviddengcn/go-villa"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"
)

// Storing the host of the current back server
var backHost villa.AtomicBox

var (
	gWaitBeforeKill  time.Duration = 10 * time.Second
	gWaitBeforeDel   time.Duration = 1 * time.Second
	gWaitBeforeStart time.Duration = 1 * time.Second
)

func startBackServer(exeFile villa.Path, host string) (cmd *exec.Cmd) {
	cmd = exeFile.Command(host)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Start()
	if err != nil {
		return nil
	}

	log.Printf("Waiting for new back server %s starting...\n", host)
	time.Sleep(gWaitBeforeStart)

	return cmd
}

func killBackServer(cmd *exec.Cmd, exeFile villa.Path, lock *sync.Mutex) {
	// release to lock to allow the entry reused.
	defer lock.Unlock()

	log.Println("Waiting for old host processing current requests", exeFile)
	time.Sleep(gWaitBeforeKill)
	err := cmd.Process.Kill()
	if err != nil {
		log.Println("Error killing old back server:", err, exeFile)
	}

	log.Println("Waiting for old host dying", exeFile)
	stat, err := cmd.Process.Wait()
	log.Println("Host killed:", stat, exeFile)

	time.Sleep(gWaitBeforeDel)
	log.Println("Deleting", exeFile)
	err = exeFile.Remove()
	if err != nil {
		log.Println("Deleting", exeFile, "error:", err)
	} else {
		log.Println(exeFile, "deleted!")
	}
}

func compilingLoop() {
	gWaitBeforeKill = time.Duration(gConf.Int("back.killwait", 10)) * time.Second
	gWaitBeforeDel = time.Duration(gConf.Int("back.delwait", 1)) * time.Second
	gWaitBeforeStart = time.Duration(gConf.Int("back.startwait", 1)) * time.Second

	codeRoot, _ := villa.Path(gConf.String("code.root", ".")).Abs()
	log.Println("Code root:", codeRoot)

	webRoot, _ := villa.Path(gConf.String("web.root", ".")).Abs()
	log.Println("Monitoring", webRoot, "...")

	backPorts := gConf.IntList("back.ports", []int{8081, 8082, 8083})
	log.Println("Back ports:", backPorts)
	rr_NUM := len(backPorts)

	type exeEntry struct {
		sync.Mutex
		exePath  villa.Path
		backHost string
	}

	// Initialize entries
	entries := make([]exeEntry, rr_NUM)

	for i := range entries {
		entries[i].exePath = webRoot.Join(fmt.Sprintf("gepsvr-%d.exe", (1 + i)))
		entries[i].backHost = fmt.Sprintf("localhost:%d", backPorts[i])
	}

	current, last := 0, 0
	m := newMonitor(webRoot.Join("web"), webRoot, codeRoot)
	var cmd *exec.Cmd = nil

	m.updateCheckExeFiles(entries[last].exePath, entries[current].exePath)
	for {
		func() {
			// lock the current entry, block if it is still waiting for killing
			entries[current].Lock()
			defer entries[current].Unlock()

			if m.run() || cmd == nil {
				// No server started yet, or a new back server ready
				// try start a back server
				newCmd := startBackServer(entries[current].exePath, entries[current].backHost)
				if newCmd != nil {
					// switch to new back server
					backHost.Set(entries[current].backHost)

					if cmd != nil {
						// kill the outdated back server
						entries[last].Lock()
						go killBackServer(cmd, entries[last].exePath, &entries[last].Mutex)
						cmd = nil
					}

					cmd = newCmd
					last, current = current, (current+1)%rr_NUM
					m.updateCheckExeFiles(entries[last].exePath, entries[current].exePath)
				}
				time.Sleep(1 * time.Second)
			} else {
				time.Sleep(1 * time.Second)
			}
		}()
	}
}

func startCompilingLoop() {
	backHost.Set("localhost:8081")
	go compilingLoop()
}

var client http.Client

func handleGep(w http.ResponseWriter, r *http.Request) {
	req := *r

	req.Host = backHost.Get().(string)
	req.URL.Scheme = "http"
	req.URL.Host = req.Host
	req.URL.Path = r.URL.Path
	req.RequestURI = ""

	resp, err := client.Do(&req)
	if err != nil {
		http.Error(w, fmt.Sprintf("Internal error accessing %s: %v", r.URL.Path, err), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	for k, v := range resp.Header {
		w.Header()[k] = v
	}
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

var mediaSuffixes []string = []string{
	".jpg", ".jpeg", ".jpe",
	".png",
	".gif",
	".webp",
	".zip",
	".js",
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
	if strings.HasSuffix(lowerPath, "/") {
		r.URL.Path = r.URL.Path + "index.gep"
		lowerPath = strings.ToLower(r.URL.Path)
	}
	if strings.HasSuffix(lowerPath, s_SUFFIX) {
		handleGep(w, r)
		return
	}

	if isMediaFile(lowerPath) {
		http.ServeFile(w, r, webRoot.Join(r.URL.Path).S())
	}
}

var confFile string

func init() {
	confFile = "geps.conf"
}

var gConf *ljconf.Conf

func loadConf() {
	var err error
	gConf, err = ljconf.Load(confFile)
	if err != nil {
		log.Println("Load configure files:", err)
	}
}

func main() {
	loadConf()
	startCompilingLoop()

	addr := gConf.String("listen.addr", ":8080")
	if len(os.Args) > 1 {
		addr = os.Args[1]
	}
	http.HandleFunc("/", handler)
	log.Println("Front server listening at", addr)
	http.ListenAndServe(addr, nil)
}
