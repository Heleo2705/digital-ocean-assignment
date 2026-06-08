package main

import (
    "encoding/json"
    "io"
    "log"
    "net/http"
)

func main() {
    http.HandleFunc("/echo", func(w http.ResponseWriter, r *http.Request) {
        body, _ := io.ReadAll(r.Body)
        log.Printf("received webhook: %s\n", string(body))
        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
    })

    log.Println("webhook receiver listening on :9000")
    log.Fatal(http.ListenAndServe(":9000", nil))
}
