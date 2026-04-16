package main

import (
	"io"
	"net/http"
)

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		io.WriteString(w, "Hello World from Go")
	})
	http.ListenAndServe(":8081", nil)
}
