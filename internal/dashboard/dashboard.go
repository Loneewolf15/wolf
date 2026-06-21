package dashboard

import (
	"bytes"
	_ "embed"
	"fmt"
	"net/http"
	"strconv"
)

//go:embed index.html
var indexHTML []byte

// Start launches the Observability dashboard on port 8081.
// targetPort is the port of the Wolf application being monitored.
func Start(targetPort int) {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")

		// Dynamically inject the target port into the HTML
		portStr := []byte(strconv.Itoa(targetPort))
		htmlContent := bytes.Replace(indexHTML, []byte("{{TARGET_PORT}}"), portStr, 1)

		w.Write(htmlContent)
	})

	go func() {
		fmt.Println("wolf dev: Observability Dashboard available at http://localhost:8081")
		if err := http.ListenAndServe(":8081", nil); err != nil {
			fmt.Printf("wolf dev: failed to start dashboard: %v\n", err)
		}
	}()
}
