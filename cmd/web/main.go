package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"burner-pod/internal/handlers"
	"burner-pod/internal/hub"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	h := hub.NewHub()
	go h.Run()
	go h.StartCleanup()

	mux := http.NewServeMux()

	fileServer := http.FileServer(http.Dir("./web/static"))
	mux.Handle("/static/", http.StripPrefix("/static/", fileServer))

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./web/templates/index.html")
	})

	mux.HandleFunc("/chat", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./web/templates/chat.html")
	})

	mux.HandleFunc("/ws/", handlers.NewWebSocketHandler(h))

	roomCreateHandler := handlers.NewRoomCreateHandler(h)
	mux.HandleFunc("/rooms", roomCreateHandler)
	mux.HandleFunc("/create", roomCreateHandler)
	mux.HandleFunc("/api/rooms", handlers.NewRoomsListHandler(h))

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	go func() {
		log.Printf("Starting server on http://localhost:%s\n", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("ListenAndServe: %v", err)
		}
	}()

	quit := make(chan os.Signal, 2)
	signal.Notify(quit)
	<-quit
	log.Println("Shutting down server gracefully...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}
	log.Println("Server exiting")
}
