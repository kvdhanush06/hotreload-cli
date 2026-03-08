package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		message := "Hello recovered!"
		slog.Info("Handling request", "path", r.URL.Path)
		fmt.Fprintln(w, message)
	})

	slog.Info("Test server starting", "port", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		slog.Error("Server failed", "error", err)
	}
}
