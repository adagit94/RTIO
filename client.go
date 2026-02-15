package rtio

import (
	"fmt"
	"time"
	ws "github.com/fasthttp/websocket"
)

type Client struct {
	Hub             *Hub
	Conn            *ws.Conn
	ReadSizeLimit   int64
	PingInterval    time.Duration
	PongWait        time.Duration
	MessagesToSend  chan []byte
	ValidateMessage func(msg []byte, client *Client) (bool, []byte)
}

func (c *Client) CloseConn() {
	if err := c.Conn.Close(); err != nil {
		fmt.Println("[ERR] Attempt to close connection failed:", err)
	}
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
			if ws.IsUnexpectedCloseError(err, ws.CloseGoingAway, ws.CloseAbnormalClosure) {
				fmt.Println("[ERR] Attempt to read message failed:", err)
			}

			return
		}

		if isValid, vMsg := c.ValidateMessage(msg, c); isValid {
			for client := range c.Hub.Clients {
				if client == c {
					continue
				}

				select {
				case client.MessagesToSend <- vMsg:

				default:
					fmt.Println("[ERR] Failed to sent mesage into the clients channel.")
					c.Hub.ClearClient(client)
				}
			}
		}
	}
}

func (c *Client) WriteMessages() {
	ticker := time.NewTicker(c.PingInterval)

	defer func() {
		ticker.Stop()
		c.Hub.Unsubscribe <- c
		c.CloseConn()
	}()

	for {
		select {
		case msg, ok := <-c.MessagesToSend:
			if !ok {
				c.Conn.WriteMessage(ws.CloseMessage, []byte{})
				return
			}

			writer, err := c.Conn.NextWriter(ws.TextMessage)

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
