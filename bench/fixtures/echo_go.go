// Go echo server — benchmark baseline
// Run: go run echo_go.go
package main

import (
	"fmt"
	"net/http"
	"time"
)

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"status":"ok","server":"go","ts":%d}`, time.Now().Unix())
	})
	fmt.Println("Go echo server on :8091")
	http.ListenAndServe(":8091", mux)
}
