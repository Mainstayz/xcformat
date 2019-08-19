package main

import (
	"io/ioutil"
	"log"
	"os"
	"strings"
)

func ext(path string) string {
	for i := len(path) - 1; i >= 0 && path[i] != '/'; i-- {
		if path[i] == '.' {
			return path[i:]
		}
	}
	return ""
}

func isFileExist(path string) bool {
	_, err := os.Stat(path)
	if err != nil {
		return false
	}
	return true
}

func isSwiftFile(name string) bool {
	if strings.HasSuffix(name, ".swift") {
		return true
	}
	return false
}

func WriteWithIOutil(name string, content []byte) {
	err := ioutil.WriteFile(name, content, 0644)
	if err != nil {
		log.Println(err)
	}
}
