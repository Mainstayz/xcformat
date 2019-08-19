package main

import (
	"io"
	"log"
	"os"
	"os/exec"
)

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

	data, err := exec.Command("clang-format", path).Output()
	if err != nil {
		log.Println(err)
		return
	}
	debugPrintln("write to file...")
	WriteWithIOutil(path, data)

	debugPrintln("format over !!!")

	info, err := os.Stat(path)
	if err != nil {
		log.Println(err)
		return
	}

	formatRecord[path] = info.Size()
	debugPrintln("format size :", info.Size())
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
