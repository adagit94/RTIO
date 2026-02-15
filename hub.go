package rtio

type Hub struct {
	Clients     map[*Client]bool
	Subscribe   chan *Client
	Unsubscribe chan *Client
}

func (h *Hub) ClearClient(c *Client) {
	delete(h.Clients, c)
	close(c.MessagesToSend)
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.Subscribe:
			h.Clients[client] = true

		case client := <-h.Unsubscribe:
			if _, exists := h.Clients[client]; exists {
				h.ClearClient(client)
			}
		}
	}
}

func CreateHub() *Hub {
	hub := Hub{
		Clients:     make(map[*Client]bool),
		Subscribe:   make(chan *Client),
		Unsubscribe: make(chan *Client),
	}

	return &hub
}
