package gomcli

import (
	"fmt"
	"sync"
)

var lock sync.Mutex

// Print is a wrapper over fmt.Print for thread-safe usage from gomcli.
func Print(a ...interface{}) (n int, err error) {
	lock.Lock()
	defer lock.Unlock()
	return fmt.Print(a...)
}

// Printf is a wrapper over fmt.Printf for thread-safe usage from gomcli
func Printf(format string, a ...interface{}) (n int, err error) {
	lock.Lock()
	defer lock.Unlock()
	return fmt.Printf(format, a...)
}

// Println is a wrapper over fmt.Println for thread-safe usage from gomcli
func Println(a ...interface{}) (n int, err error) {
	lock.Lock()
	defer lock.Unlock()
	return fmt.Println(a...)
}
