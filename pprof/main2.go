package main

import (
	"bytes"
	"fmt"
	"math/rand"
	"net/http"
	_ "net/http/pprof"
)

func main() {
	http.HandleFunc("/", SimulationAlloc)
	http.ListenAndServe(":6060", nil)
}

// SimulationAlloc 模拟分配内存
func SimulationAlloc(w http.ResponseWriter, r *http.Request) {
	var buf bytes.Buffer
	for i := 0; i < 100; i++ {
		for j := 0; j < 10; j++ {
			buf.WriteString(randomString(1000))
		}
	}
	fmt.Println(buf.String())
}

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

func randomString(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
}
