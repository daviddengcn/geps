package main

import(
    "fmt"
    "strings"
    "time"
    "log"
	"os"
    "github.com/daviddengcn/go-villa"
    "github.com/daviddengcn/gsp/parser"
)

const(
    fn_GSP_DIR = "gsp"
    fn_SOURCE_DIR = "src"
    fn_EXE_DIR = "exe"
    fn_TEMPLATE_DIR = "tmpl"
    
    fn_TEMPLATE_GO = "tmpl.go"
)

func needUpdate(srcs, dsts []villa.Path) bool {
	var srcInfo, dstInfo os.FileInfo
	
	for _, dst := range dsts {
		info, err := dst.Stat()
		if err == nil {
			if dstInfo == nil || dstInfo.ModTime().Before(info.ModTime()) {
				dstInfo = info
			}
		}
	}
	if dstInfo == nil {
		// No destination file found!
		return true
	}
	
	for _, src := range srcs {
		info, err := src.Stat()
		if err == nil {
			if srcInfo == nil || srcInfo.ModTime().Before(info.ModTime()) {
				srcInfo = info
			}
		}
	}
    if srcInfo == nil {
		// No source file found!
        return false
    } // if
    
    return dstInfo.ModTime().Before(srcInfo.ModTime())
}

type monitor struct {
    root villa.Path
    gspPath villa.Path
    srcPath villa.Path
    exePath villa.Path
    tmplFile villa.Path
	parser *gp.Parser
}

func newMonitor(root villa.Path) *monitor {
    m := &monitor {
        root: root,
        gspPath: root.Join(fn_GSP_DIR),
        srcPath: root.Join(fn_SOURCE_DIR),
        exePath: root.Join(fn_EXE_DIR),
        tmplFile: root.Join(fn_TEMPLATE_DIR, fn_TEMPLATE_GO)}
		
	m.parser = gp.NewParser(func(fn villa.Path) (string, error) {
	    src, err := m.gspFile(fn).ReadFile()
	    if err != nil {
	        return "", err
	    } // if
		
		return string(src), nil
	});
		
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

func (m *monitor) cmplFile(gsp villa.Path) villa.Path {
    return m.srcPath.Join(gsp + ".go.log")
}

func (m *monitor) findChangedFiles() (changed []villa.Path) {
    files, _ := m.gspPath.ReadDir()
    for _, f := range files {
        fn := villa.Path(f.Name())
        if isGsp(fn) {
            if needUpdate([]villa.Path{m.gspFile(fn), m.tmplFile}, []villa.Path{m.exeFile(fn), m.cmplFile(fn)}) {
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
    gspContents , err := m.parser.Load(gsp)
    if err != nil {
        return err
    } // if
    
    parts, err := m.parser.Parse(string(gspContents))
	if err != nil {
		return err
	}
    source := []byte(parts.GoSource())
    return srcFile.WriteFile([]byte(source), 0666)
}

func copyFile(src, dst villa.Path) (err error) {
    bytes, err := src.ReadFile()
    if err != nil {
        return err
    } // if
    return dst.WriteFile(bytes, 0666)
}

func safeLink(src, dst villa.Path) (err error) {
    err = src.Symlink(dst)
    if err == nil {
        return nil
    } // if
    
    return copyFile(src, dst)
}

func (m *monitor) compile(gsp villa.Path) {
    tmpDir, err := villa.Path("").TempDir("gsp_")
    if err != nil {
        log.Println(err)
        return
    } // if
    
    tmpTmplGo := tmpDir.Join(fn_TEMPLATE_GO)
    err = safeLink(m.tmplFile, tmpTmplGo)
    if err != nil {
        log.Println(err)
		return
    } // if
    tmpSrc := tmpDir.Join(gsp + ".go")
    err = safeLink(m.srcFile(gsp), tmpSrc)
    if err != nil {
        log.Println(err)
		return
    } // if
    
    exeFile := m.exeFile(gsp)
	cmplFile := m.cmplFile(gsp)
	
	cf, err := cmplFile.Create()
	if err != nil {
		log.Println(err)
		return
	}
	defer cf.Close()
	
    log.Println("Compiling", tmpSrc, tmpTmplGo, "to", exeFile)
    cmd := villa.Path("go").Command("build", "-o", exeFile.S(), tmpSrc.S(), tmpTmplGo.S())
	cmd.Stdout = cf
	cmd.Stderr = cf
    err = cmd.Run()
    
    if err != nil {
        log.Println("Compiling failed:", err)
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
