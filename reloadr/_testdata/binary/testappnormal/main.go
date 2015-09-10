package main

import (
	"fmt"
	"time"
)

func main() {
	for {
		fmt.Println("Testing...")
		time.Sleep(250 * time.Millisecond)
	}
}
