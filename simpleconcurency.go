package main

import (
	"fmt"
	"time"
	"sync"
)

var counter = 0;
var lock sync.Mutex

func main() {
	
	for i := 0; i < 100; i++ {
		go func() {
			lock.Lock()
			defer lock.Unlock()
			counter++
			fmt.Println(counter)
		} ()
	}

	time.Sleep(time.Millisecond * 10)
}
