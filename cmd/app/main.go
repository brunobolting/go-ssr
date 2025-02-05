package main

import (
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"app/internal/db"
	"app/internal/handler"
	"app/internal/server"
	"app/internal/user"
	"app/pkg/session"

	"github.com/go-chi/chi/v5"
	"github.com/joho/godotenv"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	go run()
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	<-quit
}

func run() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	database, err := db.NewSqliteConnection(os.Getenv("DATABASE_PATH"))
	if err != nil {
		log.Fatal(err)
	}
	defer database.Close()

	sessionRepository := session.NewSqliteRepository(database)
	sm := session.New(&session.Options{
		Lifetime:   24 * time.Hour,
		Cookie:     &session.CookieConfig{
			Name:     "session_id",
			Path:     "/",
			MaxAge:   86400,
			Secure:   true,
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
		},
		Repository: sessionRepository,
		GCInterval: 1 * time.Hour,
		SecretKey: []byte(os.Getenv("SESSION_SECRET")),
	})

	UserRepository := user.NewUserRepositorySqlite(database)
	us := user.NewUserService(UserRepository)

	app := chi.NewRouter()
	httpHandler := handler.NewHttpHandler(app, us, sm, handler.Options{
		AllowedOrigins: []string{"*"},
	})
	s := server.NewServer(":8080", httpHandler)
	s.Run()
}
