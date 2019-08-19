package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path"
	"strings"
	"syscall"
	"time"
)

var (
	help            = false
	debug           = false
	formatRecord    = map[string]int64{}
	whiteListExtMao = map[string]bool{
		".h":     true,
		".m":     true,
		".mm":    true,
		".swift": true,
	}
)

func init() {
	flag.BoolVar(&help, "h", false, "this help")
	flag.BoolVar(&debug, "d", true, "show debug info")
	// 改变默认的 Usage，flag包中的Usage 其实是一个函数类型。这里是覆盖默认函数实现，具体见后面Usage部分的分析
	flag.Usage = usage
}

func usage() {
	_, _ = fmt.Fprintf(os.Stderr, `xcformat version: xcformat/1.0
Usage: xcformat [-d]
Options:
`)
	flag.PrintDefaults()
}

func debugPrintln(v ...interface{}) {
	if debug {
		log.Println("debug[xcformat]: ", v)
	}
}

func handleFile(line string) {

	if strings.Contains(line, "Pods/") ||
		strings.Contains(line, "xcuserdata/") ||
		strings.Contains(line, "Carthage/Checkouts") ||
		strings.Contains(line, "Carthage/Build/") ||
		strings.Contains(line, "Created") ||
		strings.Contains(line, "Removed") ||
		strings.Contains(line, "IsDir") {
		return
	}

	if !strings.Contains(line, "AttributeModified") ||
		!strings.Contains(line, "IsFile") {
		return
	}

	stringArray := strings.Split(line, " ")

	if len(stringArray) > 0 {

		filePath := stringArray[0]

		// get ext
		ext := ext(filePath)

		if _, ok := whiteListExtMao[ext]; !ok {
			return
		}

		fileInfo, err := os.Stat(filePath)
		if err != nil {
			log.Println(err)
			return
		}

		fileName := fileInfo.Name()

		if strings.Contains(fileName, "~.") {
			debugPrintln(fileName, " is cache file.")
			return
		}

		modTime := fileInfo.ModTime().Unix()
		nowTime := time.Now().Unix()
		interval := nowTime - modTime

		// 2 秒内的改动才做 format
		if interval <= 2 {
			if isSwiftFile(fileName) {
				formatSwiftFile(filePath)
			} else {
				formatObjcFile(filePath)
			}
		}
	}

}

func main() {

	flag.Parse()
	if help {
		flag.Usage()
		return
	}

	sigs := make(chan os.Signal, 1)

	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		select {
		case sig := <-sigs:
			log.Println(sig)
			os.Exit(0)
		}
	}()

	if !hasCommand("fswatch") {
		log.Fatalf(" Error: Cannot find fswatch ")
		return
	}

	if !hasCommand("clang-format") {
		log.Fatalf(" Error: Cannot find clang-format ")
		return
	}

	if !hasCommand("swiftformat") {
		log.Fatalf(" Error: Cannot find swiftformat ")
		return
	}

	pwd, _ := exec.Command("pwd").Output()
	rootDir := strings.TrimSpace(string(pwd))
	styleFile := path.Join(rootDir, ".clang-format")
	if !isFileExist(styleFile) {
		log.Fatalf(" Error: Cannot find %s \n", styleFile)
		return
	}

	params := []string{"-xr", rootDir}

	execCommand("fswatch", params, handleFile)

}
