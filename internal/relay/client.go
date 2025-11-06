package relay

import (
	"encoding/json"
	"fmt"
	"log"

	"golang.org/x/net/websocket"
)

type Client struct {
	servers  []string
	conn     *websocket.Conn
	deviceID string
	onMessage func([]byte)
}

type Message struct {
	Type     string          `json:"type"`
	DeviceID string          `json:"device_id,omitempty"`
	Target   string          `json:"target,omitempty"`
	Data     json.RawMessage `json:"data,omitempty"`
}

func NewClient(servers []string, deviceID string) *Client {
	return &Client{
		servers:  servers,
		deviceID: deviceID,
	}
}

func (c *Client) Connect() error {
	for _, server := range c.servers {
		conn, err := websocket.Dial(server+"/relay", "", "http://localhost/")
		if err != nil {
			continue
		}
		
		c.conn = conn
		
		// Register device
		regMsg := Message{
			Type:     "register",
			DeviceID: c.deviceID,
		}
		
		if err := websocket.JSON.Send(c.conn, regMsg); err != nil {
			conn.Close()
			continue
		}
		
		log.Printf("Connected to relay: %s", server)
		go c.listen()
		return nil
	}
	
	return fmt.Errorf("failed to connect to any relay server")
}

func (c *Client) SendToDevice(targetID string, data []byte) error {
	if c.conn == nil {
		return fmt.Errorf("not connected to relay")
	}
	
	msg := Message{
		Type:   "relay",
		Target: targetID,
		Data:   json.RawMessage(data),
	}
	
	return websocket.JSON.Send(c.conn, msg)
}

func (c *Client) OnMessage(handler func([]byte)) {
	c.onMessage = handler
}

func (c *Client) listen() {
	for {
		var msg Message
		if err := websocket.JSON.Receive(c.conn, &msg); err != nil {
			log.Printf("Relay connection lost: %v", err)
			break
		}
		
		if msg.Type == "relay" && c.onMessage != nil {
			c.onMessage([]byte(msg.Data))
		}
	}
}

func (c *Client) Close() {
	if c.conn != nil {
		c.conn.Close()
	}
}
