package ws

// import (
// 	"context"
// 	"encoding/json"
// 	"log"
// 	"net"
// 	"time"
// )

// // connect theo device
// type Client struct {
// 	UserID   int64
// 	DeviceID string
// 	Conn     net.Conn
// 	Send     chan []byte
// 	Hub      *Hub
// }

// const MessageChannel = "chat:messages"

// func (c *Client) ReadLoop(ctx context.Context) {
// 	defer func() {
// 		c.Conn.Close()
// 	}()

// 	for {
// 		c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))

// 		// Đọc đúng WebSocket frame
// 		payload, err := ReadFrame(c.Conn)
// 		if err != nil {
// 			log.Println("disconnect:", err)
// 			break
// 		}

// 		// ping/pong sẽ trả nil → skip
// 		if payload == nil {
// 			continue
// 		}

// 		// parse JSON
// 		var evt struct {
// 			Type string `json:"type"`
// 			Data struct {
// 				ConversationID string `json:"conversation_id"`
// 				Sequence       int64  `json:"sequence"`
// 			} `json:"data"`
// 		}

// 		if err := json.Unmarshal(payload, &evt); err != nil {
// 			log.Println("invalid JSON:", err, "payload:", string(payload))
// 			continue
// 		}

// 		log.Printf("ACK: conv=%s seq=%d", evt.Data.ConversationID, evt.Data.Sequence)

// 		switch evt.Type {
// 		case "message_ack":
// 			c.handleAck(evt.Data.ConversationID, evt.Data.Sequence)
// 		}
// 	}
// }

// func (c *Client) WriteLoop(ctx context.Context) {
// 	defer c.Conn.Close()

// 	for {
// 		select {
// 		case <-ctx.Done():
// 			return

// 		case msg, ok := <-c.Send:
// 			if !ok {
// 				return
// 			}

// 			if err := WriteFrame(c.Conn, msg); err != nil {
// 				return
// 			}
// 		}
// 	}
// }

// // sẽ batch lại ở ws để tối ưu publish và worker sẽ tối ưu được hset
// func (c *Client) handleAck(convID string, sequence int64) {
// 	if convID == "" || sequence <= 0 {
// 		return
// 	}

// 	c.Hub.QueueAck(convID, c.UserID, sequence)
// }
