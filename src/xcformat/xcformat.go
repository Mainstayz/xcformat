package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
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

func usage() {
	_, _ = fmt.Fprintf(os.Stderr, `xcformat version: xcformat/1.0
Usage: xcformat [-d]
Options:
`)
	flag.PrintDefaults()
}

func init() {
	flag.BoolVar(&help, "h", false, "this help")
	flag.BoolVar(&debug, "d", true, "show debug info")
	// 改变默认的 Usage，flag包中的Usage 其实是一个函数类型。这里是覆盖默认函数实现，具体见后面Usage部分的分析
	flag.Usage = usage
}

func debugPrintln(v ...interface{}) {
	if debug {
		log.Println("debug[xcformat]: ", v)
	}
}

func ext(path string) string {
	for i := len(path) - 1; i >= 0 && path[i] != '/'; i-- {
		if path[i] == '.' {
			return path[i:]
		}
	}
	return ""
}

func isSwiftFile(name string) bool {
	if strings.HasSuffix(name, ".swift") {
		return true
	}
	return false
}

func isFileExist(path string) bool {
	_, err := os.Stat(path)
	if err != nil {
		return false
	}
	return true
}

func WriteWithIOutil(name string, content []byte) {
	err := ioutil.WriteFile(name, content, 0644)
	if err != nil {
		log.Println(err)
	}
}

func formatSwiftFile(path string) {
	debugPrintln(path)
	if size, ok := formatRecord[path]; ok {
		info, err := os.Stat(path)
		if err != nil {
			log.Println(err)
			return
		}
		if size == info.Size() {
			debugPrintln("both sides are the same, return!")
			return
		}
		debugPrintln("size:", info.Size())
	}

	debugPrintln("format start...")

	output, err := exec.Command("cat", path).Output()
	if err != nil {
		log.Println(err)
		return
	}

	formatCommand := exec.Command("swiftformat")
	stdin, err := formatCommand.StdinPipe()
	io.WriteString(stdin, string(output))
	stdin.Close()

	out, err := formatCommand.CombinedOutput()
	if err != nil {
		log.Println(err)
	}

	debugPrintln("write to file...")
	WriteWithIOutil(path, out)

	debugPrintln("format over !!!")

	info, err := os.Stat(path)
	if err != nil {
		log.Println(err)
		return
	}

	formatRecord[path] = info.Size()
	debugPrintln("format size :", info.Size())
}

func formatObjcFile(path string) {
	debugPrintln(path)
	if size, ok := formatRecord[path]; ok {
		info, err := os.Stat(path)
		if err != nil {
			log.Println(err)
			return
		}
		if size == info.Size() {
			debugPrintln("both sides are the same, return!")
			return
		}
		debugPrintln("size:", info.Size())
	}

	//_, err := exec.Command("clang-format", "-i", path).Output()
	debugPrintln("format start...")

	byte, err := exec.Command("clang-format", path).Output()
	if err != nil {
		log.Println(err)
		return
	}
	debugPrintln("write to file...")
	WriteWithIOutil(path, byte)

	debugPrintln("format over !!!")

	info, err := os.Stat(path)
	if err != nil {
		log.Println(err)
		return
	}

	formatRecord[path] = info.Size()
	debugPrintln("format size :", info.Size())
}

func hasCommand(name string) bool {
	_, err := exec.Command("type", name).Output()
	if err != nil {
		return false
	}
	return true
}

func execCommand(commandName string, params []string, handle func(string)) bool {
	//函数返回一个*Cmd，用于使用给出的参数执行name指定的程序
	cmd := exec.Command(commandName, params...)

	//显示运行的命令
	log.Println(cmd.Args)
	//StdoutPipe方法返回一个在命令Start后与命令标准输出关联的管道。Wait方法获知命令结束后会关闭这个管道，一般不需要显式的关闭该管道。
	stdout, err := cmd.StdoutPipe()

	if err != nil {
		log.Println(err)
		return false
	}

	cmd.Start()
	//创建一个流来读取管道内内容，这里逻辑是通过一行一行的读取的
	reader := bufio.NewReader(stdout)

	//实时循环读取输出流中的一行内容
	for {
		line, err2 := reader.ReadString('\n')
		if err2 != nil || io.EOF == err2 {
			break
		}
		if handle != nil {
			handle(line)
		}
	}

	//阻塞直到该命令执行完成，该命令必须是被Start方法开始执行的
	cmd.Wait()
	return true
}

func handleFile(line string) {

	// 如是目录或创建、删除操作
	if strings.Contains(line, "Created") ||
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

	// test
	//params := []string{"-xr", "/Users/apple/Documents/GitHub/PLTextView"}

	execCommand("fswatch", params, handleFile)

}
