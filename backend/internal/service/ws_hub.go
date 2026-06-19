package service

import (
	"encoding/json"
	"log"
	"sync"
	"time"
	"bim-viewer/internal/model"

	"github.com/gorilla/websocket"
)

type WSClient struct {
	Hub     *WSHub
	Conn    *websocket.Conn
	Send    chan []byte
	ModelID string
}

type WSHub struct {
	Clients     map[*WSClient]bool
	ModelRooms  map[string]map[*WSClient]bool
	Register    chan *WSClient
	Unregister  chan *WSClient
	broadcast   chan *BroadcastMessage
	mu          sync.RWMutex
}

type BroadcastMessage struct {
	ModelID string
	Message model.WSMessage
}

func NewWSHub() *WSHub {
	return &WSHub{
		Clients:    make(map[*WSClient]bool),
		ModelRooms: make(map[string]map[*WSClient]bool),
		Register:   make(chan *WSClient),
		Unregister: make(chan *WSClient),
		broadcast:  make(chan *BroadcastMessage),
	}
}

func (h *WSHub) Run() {
	for {
		select {
		case client := <-h.Register:
			h.mu.Lock()
			h.Clients[client] = true
			if _, ok := h.ModelRooms[client.ModelID]; !ok {
				h.ModelRooms[client.ModelID] = make(map[*WSClient]bool)
			}
			h.ModelRooms[client.ModelID][client] = true
			h.mu.Unlock()
			log.Printf("WS client connected to model %s, total clients: %d", client.ModelID, len(h.Clients))

		case client := <-h.Unregister:
			h.mu.Lock()
			if _, ok := h.Clients[client]; ok {
				delete(h.Clients, client)
				if room, ok := h.ModelRooms[client.ModelID]; ok {
					delete(room, client)
					if len(room) == 0 {
						delete(h.ModelRooms, client.ModelID)
					}
				}
				close(client.Send)
			}
			h.mu.Unlock()
			log.Printf("WS client disconnected from model %s, total clients: %d", client.ModelID, len(h.Clients))

		case msg := <-h.broadcast:
			h.mu.RLock()
			if room, ok := h.ModelRooms[msg.ModelID]; ok {
				data, err := json.Marshal(msg.Message)
				if err != nil {
					h.mu.RUnlock()
					continue
				}
				for client := range room {
					select {
					case client.Send <- data:
					default:
						h.mu.RUnlock()
						h.mu.Lock()
						delete(h.Clients, client)
						if room2, ok := h.ModelRooms[client.ModelID]; ok {
							delete(room2, client)
						}
						close(client.Send)
						h.mu.Unlock()
						h.mu.RLock()
					}
				}
			}
			h.mu.RUnlock()
		}
	}
}

func (h *WSHub) BroadcastToModel(modelID string, msg model.WSMessage) {
	if h == nil {
		return
	}
	h.broadcast <- &BroadcastMessage{
		ModelID: modelID,
		Message: msg,
	}
}

func (c *WSClient) WritePump() {
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}

		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func (c *WSClient) ReadPump(repo interface{ GetAnnotationsByModelSince(modelID string, since time.Time) ([]*model.Annotation, error) }) {
	defer func() {
		c.Hub.Unregister <- c
		c.Conn.Close()
	}()

	c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			break
		}

		var msg map[string]interface{}
		if err := json.Unmarshal(message, &msg); err != nil {
			continue
		}

		if msgType, ok := msg["type"].(string); ok && msgType == "sync_request" {
			if sinceStr, ok := msg["since"].(string); ok {
				since, err := time.Parse(time.RFC3339, sinceStr)
				if err == nil {
					annotations, err := repo.GetAnnotationsByModelSince(c.ModelID, since)
					if err == nil {
						data, _ := json.Marshal(model.WSMessage{
							Type:      "sync_response",
							ModelID:   c.ModelID,
							Payload:   annotations,
							Timestamp: time.Now(),
						})
						c.Conn.WriteMessage(websocket.TextMessage, data)
					}
				}
			}
		}
	}
}
