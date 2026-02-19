package server

import (
	"fmt"
	"github.com/adagit94/RTIO/helpers"
	ws "github.com/fasthttp/websocket"
	"time"
)

type Client struct {
	Hub              *Hub
	Conn             *ws.Conn
	PingInterval     time.Duration
	PongWait         time.Duration
	ReadSizeLimit    int64
	WriteMessageType int
	MessagesToSend   chan []byte
	ValidateMessage  func(msg []byte, client *Client) (bool, []byte)
}

func (c *Client) CloseConn() {
	helpers.CloseConn(c.Conn)
}

func (c *Client) ReadMessages() {
	defer func() {
		c.Hub.Unsubscribe <- c
		c.CloseConn()
	}()

	c.Conn.SetReadLimit(c.ReadSizeLimit)
	c.Conn.SetPongHandler(func(appData string) error {
		c.Conn.SetReadDeadline(time.Time{})
		return nil
	})

	for {
		_, msg, err := c.Conn.ReadMessage()

		if err != nil {
			if ws.IsUnexpectedCloseError(err, ws.CloseMessage, ws.CloseGoingAway, ws.CloseAbnormalClosure, ws.CloseNormalClosure) {
				fmt.Println("[ERR] Attempt to read message failed:", err)
			}

			return
		}

		if isValid, vMsg := c.ValidateMessage(msg, c); isValid {
			for client := range c.Hub.Clients {
				if client == c {
					continue
				}

				client.MessagesToSend <- vMsg
			}
		}
	}
}

func (c *Client) WriteMessages() {
	ticker := time.NewTicker(c.PingInterval)

	defer func() {
		c.Hub.Unsubscribe <- c
		ticker.Stop()
		c.CloseConn()
	}()

	for {
		select {
		case msg, ok := <-c.MessagesToSend:
			if !ok {
				c.Conn.WriteMessage(ws.CloseMessage, []byte{})
				return
			}

			writer, err := c.Conn.NextWriter(c.WriteMessageType)

			if err != nil {
				fmt.Println("[ERR] Failed to retrieve next writer:", err)
				return
			}

			if _, err := writer.Write(msg); err != nil {
				fmt.Println("[ERR] Failed to write message:", err)
				return
			}

			if err := writer.Close(); err != nil {
				fmt.Println("[ERR] Failed to close writer:", err)
			}

		case <-ticker.C:
			if err := c.Conn.WriteMessage(ws.PingMessage, nil); err != nil {
				fmt.Println("[ERR] Attempt to write ping message failed:", err)
			} else {
				c.Conn.SetReadDeadline(time.Now().Add(c.PongWait))
			}
		}
	}
}
