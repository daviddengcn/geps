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
    //"go/format"
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

func goFileOfSrc(src string) string {
    if filepath.Ext(src) == ".gsp" {
        src = src[:len(src)-len(".gsp")]
    } // if
    
    return src + ".go"
}

func exeFileOfSrc(src string) string {
    if filepath.Ext(src) == ".gsp" {
        src = src[:len(src)-len(".gsp")]
    } // if
    
    return src + ".exe"
}

func findChangedFiles(srcPath, exePath string) (changed villa.StringSlice) {
    files, _ := ioutil.ReadDir(srcPath)
    for _, f := range files {
        fn := f.Name()
        if filepath.Ext(fn) == ".gsp" {
            exe := exeFileOfSrc(fn)
            if needUpdate(filepath.Join(srcPath, fn), filepath.Join(exePath, exe)) {
                changed.Add(fn)
            } // if
        } // if
    } // for f
    
    return changed
}

type GspPart interface {
    goSource() string
}

type HtmlGspPart string

func (p HtmlGspPart) goSource() string {
    return fmt.Sprintf("fmt.Print(%q)\n", p)
}

type CodeGspPart string

func (p CodeGspPart) goSource() string {
    return string(p) + "\n"
}

type EvalGspPart string

func (p EvalGspPart) goSource() string {
    return fmt.Sprintf("fmt.Print(%s)\n", p)
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
import "fmt"

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

func generate(src, dst string) error {
    fmt.Println("Generating", dst, "from", src, "...")
    srcFile, err := ioutil.ReadFile(src)
    if err != nil {
        return err
    } // if
    
    parts := parse(string(srcFile))
    source := []byte(parts.goSource())
//    source, _ = format.Source(source)
    ioutil.WriteFile(dst, []byte(source), 0)
    return nil
}

func compile(src, tmpl, dst string) {
    cmd := exec.Command("go", "build", "-o", dst, src, tmpl)
    err := cmd.Run()
    
    if err != nil {
        fmt.Println(err)
    } // if
}

func run(srcPath, tmplPath, exePath string) {
    files := findChangedFiles(srcPath, exePath)
    if len(files) > 0 {
        fmt.Println("Changed files:", files)
    } // if
    
    for _, src := range files {
        goSrc := goFileOfSrc(src)
        
        err := generate(filepath.Join(srcPath, src), filepath.Join(exePath, goSrc))
        if err != nil {
            fmt.Println(err)
            continue
        } // if
        exe := exeFileOfSrc(src)
        compile(filepath.Join(exePath, goSrc), filepath.Join(tmplPath, "tmpl.go"), filepath.Join(exePath, exe))
    } // for file
}

func main() {
    path := "."
    path, _ = filepath.Abs(path)
    srcPath := filepath.Join(path, "src")
//    tmplPath := filepath.Join(path, "tmpl")
    tmplPath := filepath.Join(path, "exe")
    exePath := filepath.Join(path, "exe")
    
    fmt.Println("Monitoring", path, "...")
    fmt.Println("  Source folder:", srcPath, "...")
    fmt.Println("  Exec folder  :", exePath, "...")
    
    for {
        run(srcPath, tmplPath, exePath)
        time.Sleep(1*time.Second)
    }
}
