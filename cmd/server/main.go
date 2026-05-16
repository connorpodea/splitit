package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/you/p2p-bnpl/internal/handlers"
	"github.com/you/p2p-bnpl/internal/store"
)

func main() {
	s := store.New()
	h := handlers.New(s)

	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	mux.Handle("/", http.FileServer(http.Dir("web/static")))

	addr := ":8080"
	fmt.Printf("server running → http://localhost%s\n", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}
