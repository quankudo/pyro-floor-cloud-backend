package ws

// import (
// 	"context"
// 	"encoding/json"
// 	"log"
// 	"sync"
// 	"time"

// 	"github.com/quankudo/system-chat-api/internal/port"
// 	"github.com/quankudo/system-chat-api/internal/shared/event"
// )

// type Hub struct {
// 	mu         sync.RWMutex
// 	clients    map[int64]map[string]*Client
// 	register   chan *Client
// 	unregister chan *Client
// 	redisPub   port.Publisher
// 	ackMu      sync.Mutex
// 	pendingAck map[string]map[int64]int64
// 	pendingN   int
// }

// const (
// 	ackFlushInterval = 1 * time.Second
// 	ackBatchSize     = 1000
// )

// func NewHub(pub port.Publisher) *Hub {
// 	return &Hub{
// 		clients:    make(map[int64]map[string]*Client),
// 		register:   make(chan *Client),
// 		unregister: make(chan *Client),
// 		redisPub:   pub,
// 		pendingAck: make(map[string]map[int64]int64),
// 	}
// }

// func (h *Hub) Run() {
// 	go h.flushAcks()

// 	for {
// 		select {
// 		case c := <-h.register:
// 			h.registerClient(c)

// 		case c := <-h.unregister:
// 			h.unregisterClient(c)
// 		}
// 	}
// }

// func (h *Hub) registerClient(client *Client) {
// 	h.mu.Lock()
// 	defer h.mu.Unlock()

// 	if h.clients[client.UserID] == nil {
// 		h.clients[client.UserID] = make(map[string]*Client)
// 	}
// 	h.clients[client.UserID][client.DeviceID] = client
// }

// func (h *Hub) unregisterClient(client *Client) {
// 	h.mu.Lock()
// 	defer h.mu.Unlock()

// 	devices, ok := h.clients[client.UserID]
// 	if !ok || devices[client.DeviceID] != client {
// 		return
// 	}

// 	delete(devices, client.DeviceID)
// 	if len(devices) == 0 {
// 		delete(h.clients, client.UserID)
// 	}
// }

// func (h *Hub) QueueAck(conversationID string, userID, sequence int64) {
// 	h.ackMu.Lock()
// 	userSeq := h.pendingAck[conversationID]
// 	if userSeq == nil {
// 		userSeq = make(map[int64]int64)
// 		h.pendingAck[conversationID] = userSeq
// 	}

// 	if sequence > userSeq[userID] {
// 		if userSeq[userID] == 0 {
// 			h.pendingN++
// 		}
// 		userSeq[userID] = sequence
// 	}
// 	flush := h.pendingN >= ackBatchSize
// 	h.ackMu.Unlock()

// 	if flush {
// 		h.flushAcksOnce()
// 	}
// }

// func (h *Hub) flushAcks() {
// 	ticker := time.NewTicker(ackFlushInterval)
// 	defer ticker.Stop()

// 	for range ticker.C {
// 		h.flushAcksOnce()
// 	}
// }

// func (h *Hub) flushAcksOnce() {
// 	h.ackMu.Lock()
// 	if h.pendingN == 0 {
// 		h.ackMu.Unlock()
// 		return
// 	}

// 	pending := h.pendingAck
// 	h.pendingAck = make(map[string]map[int64]int64)
// 	h.pendingN = 0
// 	h.ackMu.Unlock()

// 	for conversationID, userSeq := range pending {
// 		acks := make([]event.UserAck, 0, len(userSeq))
// 		for userID, sequence := range userSeq {
// 			acks = append(acks, event.UserAck{
// 				UserID:   userID,
// 				Sequence: sequence,
// 			})
// 		}

// 		payload, err := json.Marshal(event.AckBatch{
// 			Type: "ack_batch",
// 			Data: event.AckBatchData{
// 				ConversationID: conversationID,
// 				Acks:           acks,
// 			},
// 		})
// 		if err != nil {
// 			log.Println("marshal ack batch error:", err)
// 			continue
// 		}

// 		if err := h.redisPub.Publish(context.Background(), event.ChatAckChannel, payload); err != nil {
// 			log.Printf("publish ack batch error conv=%s: %v", conversationID, err)
// 		}
// 	}
// }

// func (h *Hub) Send(userID int64, msg []byte) {
// 	h.mu.RLock()
// 	defer h.mu.RUnlock()

// 	if devices, ok := h.clients[userID]; ok {
// 		for _, client := range devices {
// 			select {
// 			case client.Send <- msg:
// 			default:
// 			}
// 		}
// 	}
// }

// func (h *Hub) IsOnline(userID int64) bool {
// 	h.mu.RLock()
// 	defer h.mu.RUnlock()

// 	devices, ok := h.clients[userID]
// 	return ok && len(devices) > 0
// }

// func (h *Hub) OnlineCount(userID int64) int {
// 	h.mu.RLock()
// 	defer h.mu.RUnlock()

// 	devices, ok := h.clients[userID]
// 	if ok {
// 		return len(devices)
// 	}
// 	return 0
// }
