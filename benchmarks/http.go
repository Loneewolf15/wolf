package main

import (
	"fmt"
	"net/http"
)

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		fmt.Fprintf(w, "Hello from Go Server!")
	})

	fmt.Println("Go server listening on port 8082")
	if err := http.ListenAndServe(":8082", nil); err != nil {
		panic(err)
	}
}
