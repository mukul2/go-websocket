package main

import (
    "log"
    "net/http"
    "sync"

    "github.com/gorilla/websocket"
)

type Client struct {
    ID   string
    Conn *websocket.Conn
}

var clients = make(map[string]*Client)
var broadcast = make(chan Message)
var upgrader = websocket.Upgrader{}
var mutex = &sync.Mutex{}

type Message struct {
    SenderID    string `json:"senderId"`
    ReceiverID  string `json:"receiverId"`
    Content     string `json:"content"`
    Type        string `json:"type"`
}

func handleConnections(w http.ResponseWriter, r *http.Request) {
    conn, err := upgrader.Upgrade(w, r, nil)
    if err != nil {
        log.Fatalf("Failed to upgrade to WebSocket: %v", err)
    }
    defer conn.Close()

    clientID := r.URL.Query().Get("id")
    if clientID == "" {
        log.Println("Missing client ID")
        return
    }

    client := &Client{ID: clientID, Conn: conn}
    mutex.Lock()
    clients[clientID] = client
    mutex.Unlock()

    for {
        var msg Message
        err := conn.ReadJSON(&msg)
        if err != nil {
            log.Printf("Error reading JSON: %v", err)
            delete(clients, clientID)
            break
        }

        broadcast <- msg
    }
}

func handleMessages() {
    for {
        msg := <-broadcast

        mutex.Lock()
        receiver, ok := clients[msg.ReceiverID]
        mutex.Unlock()

        if ok {
            err := receiver.Conn.WriteJSON(msg)
            if err != nil {
                log.Printf("Error sending message to %s: %v", msg.ReceiverID, err)
                receiver.Conn.Close()
                delete(clients, msg.ReceiverID)
            }
        }
    }
}

func main() {
    http.HandleFunc("/ws", handleConnections)
    go handleMessages()

    log.Println("Starting server on :8080")
    err := http.ListenAndServe(":8080", nil)
    if err != nil {
        log.Fatalf("Server failed to start: %v", err)
    }
}
