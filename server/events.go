package server

type Event struct {
	Client    *Client
	EventType string
}
