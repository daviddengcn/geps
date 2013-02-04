package main

import(
    "fmt"
    "github.com/daviddengcn/go-villa"
    "strings"
    "os/exec"
    "time"
    "log"
)

const(
    fn_GSP_DIR = "gsp"
    fn_SOURCE_DIR = "src"
    fn_EXE_DIR = "exe"
    fn_TEMPLATE_DIR = "tmpl"
    
    fn_TEMPLATE_GO = "tmpl.go"
)

func needUpdate(src, dst villa.Path) bool {
    dstInfo, err := dst.Stat()
    if err != nil {
        // Destination does not exist
        return true
    } // if
    
    srcInfo, err := src.Stat()
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
    root villa.Path
    gspPath villa.Path
    srcPath villa.Path
    exePath villa.Path
    tmplFile villa.Path
}

func newMonitor(root villa.Path) *monitor {
    m := &monitor {
        root: root,
        gspPath: root.Join(fn_GSP_DIR),
        srcPath: root.Join(fn_SOURCE_DIR),
        exePath: root.Join(fn_EXE_DIR),
        tmplFile: root.Join(fn_TEMPLATE_DIR, fn_TEMPLATE_GO)}
		
	m.srcPath.MkdirAll(0777)
	m.exePath.MkdirAll(0777)
		
	return m
}

func isGsp(fn villa.Path) bool {
    return strings.ToLower(fn.Ext()) == ".gsp"
}

func (m *monitor) gspFile(gsp villa.Path) villa.Path {
    return m.gspPath.Join(gsp)
}

func (m *monitor) srcFile(gsp villa.Path) villa.Path {
    return m.srcPath.Join(gsp + ".go")
}

func (m *monitor) exeFile(gsp villa.Path) villa.Path {
    return m.exePath.Join(gsp + ".exe")
}

func (m *monitor) findChangedFiles() (changed []villa.Path) {
    files, _ := m.gspPath.ReadDir()
    for _, f := range files {
        fn := villa.Path(f.Name())
        if isGsp(fn) {
            if needUpdate(m.gspFile(fn), m.exeFile(fn)) {
                changed = append(changed, fn)
            } // if
        } // if
    } // for f
    
    return changed
}

func (m *monitor) generate(gsp villa.Path) error {
    gspFile := m.gspFile(gsp)
    srcFile := m.srcFile(gsp)
    fmt.Println("Generating", srcFile, "from", gspFile, "...")
    gspContents, err := gspFile.ReadFile()
    if err != nil {
        return err
    } // if
    
    parts := parse(string(gspContents))
    source := []byte(parts.goSource())
    return srcFile.WriteFile([]byte(source), 0666)
}

func (m *monitor) compile(gsp villa.Path) {
    tmpDir, err := villa.Path("").TempDir("gsp_")
    if err != nil {
        log.Println(err)
        return
    } // if
    
    tmpTmplGo := tmpDir.Join(fn_TEMPLATE_GO)
    m.tmplFile.Symlink(tmpTmplGo)
    tmpSrc := tmpDir.Join(gsp + ".go")
    m.srcFile(gsp).Symlink(tmpSrc)
    
    exeFile := m.exeFile(gsp)
    log.Println("Compiling", tmpSrc, tmpTmplGo, "to", exeFile)
    cmd := exec.Command("go", "build", "-o", exeFile.S(), tmpSrc.S(), tmpTmplGo.S())
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
    root := villa.Path(".")
    root, _ = root.Abs()
    
    log.Println("Monitoring", root, "...")

    m := newMonitor(root)   
    for {
        m.run()
        time.Sleep(1*time.Second)
    }
}
