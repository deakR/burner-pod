package main

import (
	"burner-pod/internal/handlers"
	"burner-pod/internal/utils"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"
)

func main() {
	hub := handlers.NewHub()
	go hub.Run()
	go hub.StartCleanup()

	fileServer := http.FileServer(http.Dir("./ui/static"))
	http.Handle("/static/", http.StripPrefix("/static/", fileServer))

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./ui/html/index.html")
	})

	http.HandleFunc("/chat", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./ui/html/chat.html")
	})

	http.HandleFunc("/ws/", hub.ServeWs)

	http.HandleFunc("/create", func(w http.ResponseWriter, r *http.Request) {
		roomID := r.URL.Query().Get("code")

		if roomID == "" {
			var err error
			roomID, err = utils.GenerateRoomID()
			if err != nil {
				http.Error(w, "Failed to generate room ID", http.StatusInternalServerError)
				return
			}
		}

		isPublic := r.URL.Query().Get("public") == "true"

		ttlStr := r.URL.Query().Get("ttl")
		seconds, err := strconv.Atoi(ttlStr)
		if err != nil || seconds < 1 {
			seconds = 60
		}

		username := r.URL.Query().Get("user")
		if username == "" {
			username = "Anonymous"
		}

		duration := time.Duration(seconds) * time.Second

		success := hub.CreateRoom(roomID, duration, isPublic)
		if !success {
			http.Error(w, "Error: Room name '"+roomID+"' is already taken.", http.StatusConflict)
			return
		}

		q := r.URL.Query()
		q.Set("room", roomID)
		q.Set("user", username)
		targetUrl := "/chat?" + q.Encode()
		http.Redirect(w, r, targetUrl, http.StatusFound)
	})

	http.HandleFunc("/api/rooms", func(w http.ResponseWriter, r *http.Request) {
		rooms := hub.GetPublicRooms()

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(rooms)
	})

	log.Println("Starting server on http://localhost:8080")
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
