package client

import (
	"fmt"
	"github.com/adagit94/RTIO/helpers"
	ws "github.com/fasthttp/websocket"
	"time"
)

func InitClientReader(conn *ws.Conn, pingInterval time.Duration, pongWait time.Duration, handleIncomingMessage func(msgType int, msg []byte), onClose func(err error)) {
	var readMessages = func() {
		var outErr error

		defer func() {
			err := helpers.CloseConn(conn)

			if err == nil {
				onClose(outErr)
			}
		}()

		conn.SetPongHandler(func(appData string) error {
			conn.SetReadDeadline(time.Time{})
			return nil
		})

		for {
			msgType, msg, err := conn.ReadMessage()

			if err != nil {
				if ws.IsUnexpectedCloseError(err, ws.CloseMessage, ws.CloseGoingAway, ws.CloseAbnormalClosure, ws.CloseNormalClosure) {
					fmt.Println("[ERR] Attempt to read message failed:", err)
				}

				outErr = err
				return
			}

			handleIncomingMessage(msgType, msg)
		}
	}

	var schedulePings = func() {
		var outErr error
		ticker := time.NewTicker(pingInterval)

		defer func() {
			err := helpers.CloseConn(conn)

			if err == nil {
				ticker.Stop()
				onClose(outErr)
			}

		}()

		conn.SetPongHandler(func(appData string) error {
			conn.SetReadDeadline(time.Time{})
			return nil
		})

		for {
			<-ticker.C

			if err := conn.WriteMessage(ws.PingMessage, nil); err != nil {
				fmt.Println("[ERR] Attempt to write ping message failed:", err)
			} else {
				conn.SetReadDeadline(time.Now().Add(pongWait))
			}
		}
	}

	go readMessages()
	go schedulePings()
}
