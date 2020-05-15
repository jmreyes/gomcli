package gomcli

import (
	"fmt"
	"sync"
)

var lock sync.Mutex

func Print(a ...interface{}) (n int, err error) {
	lock.Lock()
	defer lock.Unlock()
	return fmt.Print(a...)
}

func Printf(format string, a ...interface{}) (n int, err error) {
	lock.Lock()
	defer lock.Unlock()
	return fmt.Printf(format, a...)
}

func Println(a ...interface{}) (n int, err error) {
	lock.Lock()
	defer lock.Unlock()
	return fmt.Println(a...)
}
