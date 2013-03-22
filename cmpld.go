package main

import (
	"bytes"
	"fmt"
	"github.com/daviddengcn/geps/gep"
	"github.com/daviddengcn/go-villa"
	"log"
	"os"
	"strconv"
	"strings"
)

const (
	s_SUFFIX = ".gep"

	fn_WEB_DIR    = "web"
	fn_SOURCE_DIR = "src"
	fn_GEPSVR_GO  = "gepsvr.go"
)

type monitor struct {
	webDir     villa.Path
	srcDir     villa.Path
	checkFile  villa.Path
	exeFile    villa.Path
	gepsvrFile villa.Path
	tmpRoot    villa.Path
}

func newMonitor(web, src, inc, tmp villa.Path) *monitor {
	m := &monitor{
		webDir:     web,
		srcDir:     src,
		tmpRoot:    tmp,
		gepsvrFile: inc.Join(fn_GEPSVR_GO),
	}

	m.srcDir.MkdirAll(0777)

	return m
}

func (m *monitor) updateCheckExeFiles(check, exe villa.Path) {
	m.checkFile = check
	m.exeFile = exe
}

func (m *monitor) scanFiles() (files map[villa.Path]os.FileInfo) {
	files = make(map[villa.Path]os.FileInfo)
	m.webDir.Walk(func(path villa.Path, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() && strings.ToLower(path.Ext()) == s_SUFFIX {
			rel, err := m.webDir.Rel(path)
			if err == nil {
				files[rel] = info
			}
		} // if
		return nil
	})

	return
}

func (m *monitor) needUpdate(files map[villa.Path]os.FileInfo) bool {
	exeInfo, err := m.checkFile.Stat()

	if err != nil {
		return true
	}

	for _, info := range files {
		if exeInfo.ModTime().Before(info.ModTime()) {
			return true
		}
	}

	return false
}

func pathToSrcName(path villa.Path) string {
	return strings.Map(func(r rune) rune {
		if r == '\\' || r == '/' {
			return '_'
		}

		return r
	}, path.S())
}

func pathToUrl(path villa.Path) string {
	return strings.Map(func(r rune) rune {
		if r == '\\' {
			return '/'
		}

		return r
	}, path.S())
}

func greater(a, b villa.Path) bool {
	if len(a) > len(b) {
		return true
	}

	return a > b
}

func (m *monitor) genSourceNames(files map[villa.Path]os.FileInfo) (srcPath map[string]villa.Path) {
	srcPath = make(map[string]villa.Path)
	pathSrc := make(map[villa.Path]string)
	for path := range files {
		src := pathToSrcName(path)
		//log.Println("src:", src)
		for {
			p, ok := srcPath[src]
			if !ok {
				srcPath[src], pathSrc[path] = path, src
				break
			}

			if greater(p, path) {
				// replace (src, p) with (src, path)
				srcPath[src], pathSrc[path] = path, src
				// try find another src for p
				path = p
			}
			src += "_"
		}
	}

	return
}

type sourceGenerator struct {
	m *monitor
}

func (sg *sourceGenerator) Load(path villa.Path) (string, error) {
	//log.Println("Reading", path)
	src, err := sg.m.webDir.Join(path).ReadFile()
	if err != nil {
		return "", err
	} // if

	return string(src), nil
}

func (sg *sourceGenerator) GenRawPart(src string) interface{} {
	return fmt.Sprintf("__print__(__response__, %q)\n", src)
}

func (sg *sourceGenerator) GenCodePart(src string) interface{} {
	return string(src) + "\n"
}

func (sg *sourceGenerator) GenEvalPart(src string) interface{} {
	return fmt.Sprintf("__print__(__response__, %s)\n", src)
}

func (sg *sourceGenerator) Error(message string) {
	log.Println("GEP parse error:", message)
}

const sTemplate = `package main

import(
#_imports_#)

func init() {
	fmt.Print()
	strings.TrimSpace("")
	registerPath(#_url_path_#, __process_#_func_name_#)
}

func __process_#_func_name_#(response http.ResponseWriter, request *http.Request) {
	__response__ := response
	_ = __response__
#_body_#}
`

func genGoSource(parts *gep.GepParts, url, func_name string) string {
	parts.Imports.Put("fmt", "net/http", "strings")
	//log.Println("Imports:", parts.Imports)
	src := sTemplate
	var out bytes.Buffer
	for imp := range parts.Imports {
		out.WriteString("\t" + strconv.Quote(imp) + "\n")
	}
	src = strings.Replace(src, "#_imports_#", out.String(), -1)
	src = strings.Replace(src, "#_url_path_#", strconv.Quote("/"+url), -1)
	src = strings.Replace(src, "#_func_name_#", func_name, -1)

	out.Reset()
	for _, part := range parts.Parts {
		out.WriteString("\t" + fmt.Sprint(part))
	}
	src = strings.Replace(src, "#_body_#", out.String(), -1)
	return src
}

func (m *monitor) parse(srcFiles map[string]villa.Path) error {
	cnt := 0
	sg := sourceGenerator{m: m}
	for src, path := range srcFiles {
		url := pathToUrl(path)
		gepSrc, err := sg.Load(path)
		if err == nil {
			parts, err := gep.Parse(&sg, gepSrc)
			if err == nil {
				if parts.IncludeOnly {
					delete(srcFiles, src)
					log.Println(path, "IncludeOnly, ignored!")
					continue
				}
				goSrc := genGoSource(parts, url, fmt.Sprint(cnt))
				cnt++

				srcFile := m.srcDir.Join(src + ".go")
				//fmt.Println("Generating", srcFile, "...")
				err = srcFile.WriteFile([]byte(goSrc), 0666)
				if err != nil {
					sg.Error(fmt.Sprint(err))
				}
			} else {
				sg.Error(fmt.Sprint(err))
			}
		}
	}

	return nil
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
	}

	return copyFile(src, dst)
}

func (m *monitor) compile(srcFiles map[string]villa.Path) (err error) {
	// Create temporary directory
	tmpDir, err := villa.Path(m.tmpRoot).TempDir("gep_")
	if err != nil {
		log.Println("Creating tmpDir failed:", err)
		return err
	}

	// Link gepsrv.go
	gepsvrGo := tmpDir.Join(fn_GEPSVR_GO)
	err = safeLink(m.gepsvrFile, gepsvrGo)
	if err != nil {
		log.Println("Linking", m.gepsvrFile, "to temp folder:", err)
		return
	}

	// Link source go files
	for src := range srcFiles {
		safeLink(m.srcDir.Join(src+".go"), tmpDir.Join(src+".go"))
	}

	exeFile := m.exeFile
	cmplFile := villa.Path(m.exeFile + ".log")
	// Open log file
	cf, err := cmplFile.Create()
	if err != nil {
		log.Println("Create log file:", err)
		return
	}
	defer cf.Close()

	log.Println("Compiling", tmpDir, "to", exeFile)

	// Compile
	cmd := villa.Path("go").Command("build", "-o", exeFile.S())
	cmd.Stdout = cf
	cmd.Stderr = cf
	cmd.Dir = tmpDir.S()
	err = cmd.Run()

	if err != nil {
		log.Println("Compiling failed:", err)
		return err
	}

	return nil
}

func (m *monitor) run() (changed bool) {
	files := m.scanFiles()
	if !m.needUpdate(files) {
		return false
	}
	srcFiles := m.genSourceNames(files)
	log.Println("Compiling:", srcFiles)
	err := m.parse(srcFiles)
	if err != nil {
		return false
	}
	err = m.compile(srcFiles)
	if err != nil {
		return false
	}
	return true
}
