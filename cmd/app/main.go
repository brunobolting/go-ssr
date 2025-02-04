package main

import (
	"os"
	"os/signal"

	"app/internal/handler"
	"app/internal/server"

	"github.com/go-chi/chi/v5"
)

func main() {
	go run()
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	<-quit
}

func run() {
	app := chi.NewRouter()
	s := server.NewServer(":8080", handler.NewHttpHandler(app))
	s.Run()
}
