package webhook
package main

import (
    "fmt"
    "io"
    "log"
    "net/http"
)

func main() {
    http.HandleFunc("/echo", func(w http.ResponseWriter, r *http.Request) {
        body, _ := io.ReadAll(r.Body)
        log.Printf("received webhook: %s\n", string(body))
        w.Header().Set("Content-Type", "text/plain")
        fmt.Fprintln(w, "ok")
    })

    log.Println("webhook receiver listening on :9000")
    log.Fatal(http.ListenAndServe(":9000", nil))
}
