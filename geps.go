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
	gWaitBeforeKill  time.Duration = 10
	gWaitBeforeDel   time.Duration = 1
	gWaitBeforeStart time.Duration = 1
)

func startBackServer(exeFile villa.Path, host string) (cmd *exec.Cmd) {
	cmd = exeFile.Command(host)
	cmd.Dir = gPaths.webRoot.S()
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Start()
	if err != nil {
		return nil
	}

	log.Printf("Waiting %ds for new back server %s starting...\n", gWaitBeforeStart, host)
	time.Sleep(gWaitBeforeStart * time.Second)

	return cmd
}

func killBackServer(cmd *exec.Cmd, exeFile villa.Path, lock *sync.Mutex) {
	// release to lock to allow the entry reused.
	defer lock.Unlock()

	log.Printf("Waiting %ds for old host processing current requests: %v", gWaitBeforeKill, exeFile)
	time.Sleep(gWaitBeforeKill * time.Second)
	err := cmd.Process.Kill()
	if err != nil {
		log.Println("Error killing old back server:", err, exeFile)
	}

	log.Printf("Waiting for old host dying", exeFile)
	stat, err := cmd.Process.Wait()
	
	log.Printf("Host killed(delete after %ds): %v, %s", gWaitBeforeDel, stat, exeFile)
	time.Sleep(gWaitBeforeDel * time.Second)
	
	log.Println("Deleting", exeFile)
	err = exeFile.Remove()
	if err != nil {
		log.Println("Deleting", exeFile, "failed:", err)
	} else {
		log.Println(exeFile, "deleted!")
	}
}

func compilingLoop() {
	gWaitBeforeKill = time.Duration(gConf.Int("back.killwait", int(gWaitBeforeKill)))
	gWaitBeforeDel = time.Duration(gConf.Int("back.delwait", int(gWaitBeforeDel)))
	gWaitBeforeStart = time.Duration(gConf.Int("back.startwait", int(gWaitBeforeStart)))

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

	gPaths.exe.MkdirAll(0777)
	gPaths.tmp.MkdirAll(0777)

	for i := range entries {
		entries[i].exePath = gPaths.exe.Join(fmt.Sprintf("gepsvr-%d.exe", (1 + i)))
		entries[i].backHost = fmt.Sprintf("localhost:%d", backPorts[i])
	}

	current, last := 0, 0
	m := newMonitor(gPaths.webRoot, gPaths.src, gPaths.inc, gPaths.tmp)
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
	backHost.Set("")
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

var gPaths struct {
	webRoot            villa.Path
	src, exe, tmp, inc villa.Path
}

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
		http.ServeFile(w, r, gPaths.webRoot.Join(r.URL.Path).S())
	}
}

var confFile string

func init() {
	confFile = "geps.conf"
}

var gConf *ljconf.Conf

const GEPS_PKG_PATH = "github.com/daviddengcn/geps"

func goPath() villa.Path {
	p := os.Getenv("GOPATH")
	return villa.Path(p).Join("src", GEPS_PKG_PATH, "gepsvr")
}

func loadConf() {
	var err error
	gConf, err = ljconf.Load(confFile)
	if err != nil {
		log.Println("Load configure files:", err)
	}

	gPaths.webRoot = villa.Path(gConf.String("web.root", "web")).AbsPath()
	gPaths.src = villa.Path(gConf.String("code.src", "src")).AbsPath()
	gPaths.exe = villa.Path(gConf.String("code.exe", "exe")).AbsPath()
	gPaths.tmp = villa.Path(gConf.String("code.tmp", ""))
	if gPaths.tmp != "" {
		gPaths.tmp = gPaths.tmp.AbsPath()
	}
	gPaths.inc = villa.Path(gConf.String("code.inc", goPath().S())).AbsPath()

	fmt.Printf("Path set: %+v\n", gPaths)
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
