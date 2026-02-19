package helpers

import (
	"fmt"
	ws "github.com/fasthttp/websocket"
)

func CloseConn(conn *ws.Conn) error {
	if err := conn.Close(); err != nil {
		fmt.Println("[ERR] Attempt to close connection failed:", err)
		return err
	}

	return nil
}