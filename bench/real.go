package main

import (
	"encoding/json"
	"net/http"
)

type Payload struct {
	UserID    int    `json:"user_id"`
	Action    string `json:"action"`
	Timestamp int    `json:"timestamp"`
}

type Response struct {
	SQL    string `json:"sql"`
	Result int    `json:"math_result"`
}

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// 1. JSON parse
		data := `{"user_id": 999, "action": "login", "timestamp": 160000}`
		var p Payload
		json.Unmarshal([]byte(data), &p)

		// 2. Arithmetic (factorial)
		res := 1
		for i := 1; i <= 20; i++ {
			res += i * 2
		}

		// 3. Mock DB
		sql := "SELECT * FROM users WHERE id = 999 LIMIT 10"

		// 4. JSON Encode & Reply
		resp := Response{SQL: sql, Result: res}
		out, _ := json.Marshal(resp)

		w.Header().Set("Content-Type", "application/json")
		w.Write(out)
	})
	http.ListenAndServe(":8081", nil)
}
