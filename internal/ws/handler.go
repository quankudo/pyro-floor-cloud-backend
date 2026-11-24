package ws

// import (
// 	"context"
// 	"encoding/json"
// 	"log"
// 	"net"
// 	"net/http"
// 	"time"

// 	"github.com/pyrofloor/cloud-backend/internal/shared/topic"
// 	"github.com/quankudo/system-chat-api/internal/modules/auth/service"
// 	"github.com/quankudo/system-chat-api/internal/port"
// 	sharedEvent "github.com/quankudo/system-chat-api/internal/shared/event"
// )

// const (
// 	workerCount     = 10
// 	eventBufferSize = 10000
// )

// type EventHandler func(userID int64, evt Event, raw []byte)

// type WSHandler struct {
// 	Hub   *Hub
// 	Redis port.Subscriber

// 	eventCh chan WSMessage

// 	handlers map[string]EventHandler
// }

// func NewHandler(
// 	hub *Hub,
// 	redis port.Subscriber,
// ) *WSHandler {
// 	return &WSHandler{
// 		Hub:      hub,
// 		Redis:    redis,
// 		eventCh:  make(chan WSMessage, eventBufferSize),
// 		handlers: make(map[string]EventHandler),
// 	}
// }

// func (h *WSHandler) ServeWS(w http.ResponseWriter, r *http.Request) {

// 	if err := r.ParseForm(); err != nil {
// 		http.Error(w, "invalid form", http.StatusBadRequest)
// 		return
// 	}

// 	tokenStr := r.FormValue("ctoken")

// 	claims, err := service.VerifyToken(tokenStr)

// 	if err != nil {
// 		http.Error(w, "invalid token", http.StatusBadRequest)
// 		return
// 	}

// 	userID := claims.UserID
// 	deviceID := claims.DeviceID

// 	conn, err := Upgrade(w, r)
// 	if err != nil {
// 		http.Error(w, "Upgrade failed", http.StatusBadRequest)
// 		return
// 	}

// 	client := &Client{
// 		UserID:   userID,
// 		DeviceID: deviceID,
// 		Conn:     conn,
// 		Send:     make(chan []byte, 256),
// 		Hub:      h.Hub,
// 	}

// 	h.Hub.register <- client

// 	defer func() {
// 		if r := recover(); r != nil {
// 			log.Println("panic recovered:", r)
// 		}
// 		h.Hub.unregister <- client
// 		log.Printf("WS disconnected user=%d device=%s", userID, deviceID)
// 	}()

// 	ctx, cancel := context.WithCancel(r.Context())
// 	defer cancel()

// 	go h.keepAlive(ctx, cancel, conn, userID)
// 	go client.WriteLoop(ctx)

// 	client.ReadLoop(ctx)
// }

// type WSMessage struct {
// 	UserID int64
// 	Data   []byte
// }

// func (h *WSHandler) pushEvent(userID int64, msg []byte) {
// 	event := WSMessage{
// 		UserID: userID,
// 		Data:   msg,
// 	}

// 	select {
// 	case h.eventCh <- event:
// 	case <-time.After(2 * time.Second):
// 		log.Println("event timeout drop:", userID)
// 	}
// }

// func (h *WSHandler) StartWorkers(ctx context.Context) {
// 	for i := 0; i < workerCount; i++ {
// 		go func(id int) {
// 			defer func() {
// 				if r := recover(); r != nil {
// 					log.Println("worker panic:", id, r)
// 				}
// 			}()

// 			log.Println("worker started:", id)

// 			for {
// 				select {
// 				case <-ctx.Done():
// 					return

// 				case data := <-h.eventCh:
// 					h.handleEvent(data.UserID, data.Data)
// 				}
// 			}
// 		}(i)
// 	}

// 	go h.listenEvents(ctx)
// }

// func (h *WSHandler) listenEvents(ctx context.Context) {
// 	retryDelay := time.Second

// 	for ctx.Err() == nil {
// 		ch, err := h.Redis.Subscribe(ctx, topic.WSEventsChannel)
// 		if err != nil {
// 			log.Printf("WS event subscribe failed channel=%s: %v", topic.WSEventsChannel, err)
// 			if !waitForRetry(ctx, retryDelay) {
// 				return
// 			}
// 			if retryDelay < 30*time.Second {
// 				retryDelay *= 2
// 			}
// 			continue
// 		}

// 		log.Printf("WS subscribed channel=%s", topic.WSEventsChannel)
// 		channelClosed := false
// 		for !channelClosed {
// 			select {
// 			case <-ctx.Done():
// 				return
// 			case msg, ok := <-ch:
// 				if !ok {
// 					channelClosed = true
// 					continue
// 				}
// 				retryDelay = time.Second
// 				h.handlePublishedEvent(msg)
// 			}
// 		}

// 		log.Printf("WS event subscription closed channel=%s", topic.WSEventsChannel)
// 		if !waitForRetry(ctx, retryDelay) {
// 			return
// 		}
// 		if retryDelay < 30*time.Second {
// 			retryDelay *= 2
// 		}
// 	}
// }

// func waitForRetry(ctx context.Context, delay time.Duration) bool {
// 	timer := time.NewTimer(delay)
// 	defer timer.Stop()

// 	select {
// 	case <-ctx.Done():
// 		return false
// 	case <-timer.C:
// 		return true
// 	}
// }

// func (h *WSHandler) handlePublishedEvent(raw []byte) {
// 	var targeted sharedEvent.TargetedWSEvent
// 	if err := json.Unmarshal(raw, &targeted); err != nil {
// 		log.Printf("invalid targeted WS event: %v", err)
// 		return
// 	}
// 	if len(targeted.UserIDs) == 0 || !json.Valid(targeted.Event) {
// 		log.Printf("invalid targeted WS event: missing recipients or event")
// 		return
// 	}

// 	seen := make(map[int64]struct{}, len(targeted.UserIDs))
// 	for _, userID := range targeted.UserIDs {
// 		if userID <= 0 {
// 			continue
// 		}
// 		if _, exists := seen[userID]; exists {
// 			continue
// 		}
// 		seen[userID] = struct{}{}
// 		h.pushEvent(userID, targeted.Event)
// 	}
// }

// type Event struct {
// 	Type string          `json:"type"`
// 	Data json.RawMessage `json:"data"`
// }

// func (h *WSHandler) handleEvent(userID int64, msg []byte) {
// 	var evt Event
// 	if err := json.Unmarshal(msg, &evt); err != nil {
// 		log.Println("invalid event:", err)
// 		return
// 	}

// 	handler, ok := h.handlers[evt.Type]
// 	if !ok {
// 		log.Println("unknown event type:", evt.Type)
// 		return
// 	}

// 	// panic-safe từng handler
// 	defer func() {
// 		if r := recover(); r != nil {
// 			log.Println("event handler panic:", evt.Type, r)
// 		}
// 	}()

// 	handler(userID, evt, msg)
// }

// func (h *WSHandler) HandleMessageRead(userID int64, evt Event, raw []byte) {
// 	if !h.Hub.IsOnline(userID) {
// 		return
// 	}

// 	h.Hub.Send(userID, raw)

// 	log.Printf("[WS][%s][MARK_READ] → userId=%d payload=%s\n",
// 		time.Now().Format(time.RFC3339),
// 		userID,
// 		string(raw),
// 	)

// }

// func (h *WSHandler) keepAlive(ctx context.Context, cancel context.CancelFunc, conn net.Conn, userID int64) {
// 	ticker := time.NewTicker(30 * time.Second)
// 	defer ticker.Stop()

// 	for {
// 		select {
// 		case <-ctx.Done():
// 			return

// 		case <-ticker.C:
// 			if err := WritePing(conn); err != nil {
// 				log.Println("ping failed:", userID)
// 				cancel()
// 				return
// 			}
// 		}
// 	}
// }

// func WritePing(conn net.Conn) error {
// 	frame := []byte{0x89, 0x00}
// 	_, err := conn.Write(frame)
// 	return err
// }
