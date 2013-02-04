package main

import(
    "fmt"
    "path/filepath"
    "io/ioutil"
    "github.com/daviddengcn/go-villa"
    "strings"
    "os"
    "os/exec"
    "time"
    "log"
    //"go/format"
)

const(
    fn_GSP_DIR = "gsp"
    fn_SOURCE_DIR = "src"
    fn_EXE_DIR = "exe"
    fn_TEMPLATE_DIR = "tmpl"
    
    fn_TEMPLATE_GO = "tmpl.go"
)

func needUpdate(src, dst string) bool {
    dstInfo, err := os.Stat(dst)
    if err != nil {
        // Destination does not exist
        return true
    } // if
    
    srcInfo, err := os.Stat(src)
    if err != nil {
        return false
    }
    return dstInfo.ModTime().Before(srcInfo.ModTime())
}

type GspPart interface {
    goSource() string
}

type HtmlGspPart string

func (p HtmlGspPart) goSource() string {
    return fmt.Sprintf("__print__(%q)\n", p)
}

type CodeGspPart string

func (p CodeGspPart) goSource() string {
    return string(p) + "\n"
}

type EvalGspPart string

func (p EvalGspPart) goSource() string {
    return fmt.Sprintf("__print__(%s)\n", p)
}

type GspParts struct {
    local []GspPart
    global []GspPart
}

func (ps *GspParts) addHtml(src string) {
    if len(src) == 0 {
        return
    } // if
    ps.local = append(ps.local, HtmlGspPart(src))
}

const (
    ct_LOCAL = iota
    ct_GLOBAL
    ct_EVAL
)

func (ps *GspParts) addCode(src string, codeType int) {
    switch codeType {
        case ct_LOCAL:
            ps.local = append(ps.local, CodeGspPart(src))
            
        case ct_GLOBAL:
            ps.global = append(ps.global, CodeGspPart(src))
            
        case ct_EVAL:
            ps.local = append(ps.local, EvalGspPart(strings.TrimSpace(src)))
    }
}

func (ps GspParts) goSource() (src string) {
    src = `package main

func __process__() {
`
    for _, p := range ps.local {
        src += "    " + p.goSource()
    } // for p
    
    src += "}\n"
    
    return src
}

func parse(src string) (parts GspParts){
    const (
        R = iota
        C0
        C1
        C2
        C3
    )
    status, tp := R, ct_GLOBAL
    source := ""
    for _, r := range src {
        switch status {
            case R:
                switch r {
                    case '<':
                        status = C0
                        
                    default:
                        source += string(r)
                }
            case C0:
                switch r {
                    case '%':
                        status, tp = C1, ct_LOCAL
                        parts.addHtml(source)
                        source = ""
                    
                    default:
                        status = R
                        source += "<"
                        source += string(r)
                }
                
            case C1:
                switch r {
                    case '=':
                        tp = ct_EVAL
                        
                    case '%':
                        status = C3
                        
                    default:
                        status = C2
                        source += string(r)
                }
                
            case C2:
                switch r {
                    case '%':
                        status = C3
                        
                    default:
                        source += string(r)
                }
                
            case C3:
                switch r {
                    case '>':
                        status = R
                        parts.addCode(source, tp)
                        source = ""
                        
                    default:
                        source += "%"
                        source += string(r)
                }
        }
    } // for r
    
    switch status {
        case R:
            if len(source) > 0 {
                parts.addHtml(source)
            } // if
    }
    return parts
}

type monitor struct {
    root string
    gspPath string
    srcPath string
    exePath string
    tmplFile string
}

func newMonitor(root string) (m *monitor) {
    m = &monitor {
        root: root,
        gspPath: filepath.Join(root, fn_GSP_DIR),
        srcPath: filepath.Join(root, fn_SOURCE_DIR),
        exePath: filepath.Join(root, fn_EXE_DIR),
        tmplFile: filepath.Join(filepath.Join(root, fn_TEMPLATE_DIR), fn_TEMPLATE_GO)}
        
    os.MkdirAll(m.gspPath, 0777)
    os.MkdirAll(m.srcPath, 0777)
    os.MkdirAll(m.exePath, 0777)
        
    return m
}

func isGsp(fn string) bool {
    return strings.ToLower(filepath.Ext(fn)) == ".gsp"
}

func (m *monitor) gspFile(gsp string) string {
    return filepath.Join(m.gspPath, gsp)
}

func (m *monitor) srcFile(gsp string) string {
    return filepath.Join(m.srcPath, gsp + ".go")
}

func (m *monitor) exeFile(gsp string) string {
    return filepath.Join(m.exePath, gsp + ".exe")
}

func (m *monitor) findChangedFiles() (changed villa.StringSlice) {
    files, _ := ioutil.ReadDir(m.gspPath)
    for _, f := range files {
        fn := f.Name()
        if isGsp(fn) {
            if needUpdate(m.gspFile(fn), m.exeFile(fn)) {
                changed.Add(fn)
            } // if
        } // if
    } // for f
    
    return changed
}

func (m *monitor) generate(gsp string) error {
    gspFile := m.gspFile(gsp)
    srcFile := m.srcFile(gsp)
    fmt.Println("Generating", srcFile, "from", gspFile, "...")
    gspContents, err := ioutil.ReadFile(gspFile)
    if err != nil {
        return err
    } // if
    
    parts := parse(string(gspContents))
    source := []byte(parts.goSource())
//    source, _ = format.Source(source)
    return ioutil.WriteFile(srcFile, []byte(source), 0666)
}

func copyFile(src, dst string, perm os.FileMode) (err error) {
    bytes, err := ioutil.ReadFile(src)
    if err != nil {
        return err
    } // if
    return ioutil.WriteFile(dst, bytes, perm)
}

func safeLink(src, dst string, perm os.FileMode) (err error) {
    err = os.Symlink(src, dst)
    if err == nil {
        return nil
    } // if
    
    return copyFile(src, dst, perm)
}

func (m *monitor) compile(gsp string) {
    tmpDir, err := ioutil.TempDir("", "gsp_")
    if err != nil {
        log.Println(err)
        return
    } // if
    
    tmpTmplGo := filepath.Join(tmpDir, fn_TEMPLATE_GO)
    safeLink(m.tmplFile, tmpTmplGo, 0666)
    if err != nil {
        log.Println(err)
    } // if
    tmpSrc := filepath.Join(tmpDir, gsp + ".go")
    err = safeLink(m.srcFile(gsp), tmpSrc, 0666)
    if err != nil {
        log.Println(err)
    } // if
    
    exeFile := m.exeFile(gsp)
    log.Println("Compiling", tmpSrc, tmpTmplGo, "to", exeFile)
    cmd := exec.Command("go", "build", "-o", exeFile, tmpSrc, tmpTmplGo)
    err = cmd.Run()
    
    if err != nil {
        log.Println(err)
    } // if
}

func (m *monitor) run() {
    files := m.findChangedFiles()
    if len(files) > 0 {
        log.Printf("%d changed files found: %v", len(files), files)
    } // if
    
    for _, gsp := range files {
        err := m.generate(gsp)
        if err != nil {
            fmt.Println(err)
            continue
        } // if
        m.compile(gsp)
    } // for file
}

func main() {
    root := "."
    root, _ = filepath.Abs(root)
    
    log.Println("Monitoring", root, "...")

    m := newMonitor(root)    
    for {
        m.run()
        time.Sleep(1*time.Second)
    }
}
