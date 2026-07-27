package main

import (
	"fmt"
	"log/slog"
	"net/http"
)

func handleMain(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "Hello world")
}

// test
func main() {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /", handleMain)

	slog.Info("starting http server on 0.0.0.0:8080")
	http.ListenAndServe("0.0.0.0:8080", mux)
}
