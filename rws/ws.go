package rws

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/gomodule/redigo/redis"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/lucalattore/goat/sso"
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 30 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer.
	maxMessageSize = 512
)

var (
	newline = []byte{'\n'}
	space   = []byte{' '}
)

// Client is a middleman between the websocket connection and the request engine.
type Client struct {
	// Unique Client ID
	ID string

	// Authentication data
	AuthData sso.AuthData

	// The websocket connection.
	conn *websocket.Conn

	// Buffered channel of inbound messages.
	inbound chan []byte

	// Buffered channel of outbound messages.
	outbound chan []byte

	// Redis PubSub
	psc *redis.PubSubConn
}

// Dispatcher keep registerd function to handle ws request
type Dispatcher struct {
	h map[string]func(client *Client, req *map[string]interface{}) interface{}
}

// NewDispatcher create a new dispatcher
func NewDispatcher() *Dispatcher {
	return &Dispatcher{h: make(map[string]func(client *Client, req *map[string]interface{}) interface{})}
}

// NewError creates a new error struct
func NewError(code string, text string) interface{} {
	m := make(map[string]interface{})
	m["type"] = "error"
	m["code"] = code
	if text != "" {
		m["msg"] = text
	}

	return &m
}

// NewReply create a new reply message
func NewReply(t string, sender string) map[string]interface{} {
	m := make(map[string]interface{})
	m["type"] = t
	if sender != "" {
		m["sender"] = sender
	}

	return m
}

// Wrap add the type attribute to the current payload
func Wrap(t string, sender string, payload interface{}) map[string]interface{} {
	if m, ok := payload.(map[string]interface{}); ok {
		m["type"] = t
		if sender != "" {
			m["sender"] = sender
		}

		return m
	}

	m := NewReply(t, sender)
	m["paylaod"] = payload
	return m
}

// HandleFunc registers a function to handle message
func (dispatcher *Dispatcher) HandleFunc(t string, f func(client *Client, req *map[string]interface{}) interface{}) {
	dispatcher.h[t] = f
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
} // use default options

// WSHandler handles websocket requests
type WSHandler struct {
	RedisPool *redis.Pool
	FilterOut bool
}

// ServeWS is the function to handle websocket request. You have to register it into your http mux
func (h *WSHandler) ServeWS(dispatcher *Dispatcher, w http.ResponseWriter, r *http.Request) {
	authData, _ := r.Context().Value(sso.AuthContextKey).(sso.AuthData)
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("upgrade:", err)
		return
	}

	id := uuid.New().String()
	log.Println("Client", id, "connected as user", authData.PreferredUsername())
	client := &Client{ID: id, conn: conn, inbound: make(chan []byte), outbound: make(chan []byte), AuthData: authData}
	done := make(chan bool, 1)
	go client.listener(h.RedisPool, h.FilterOut, done)
	go client.process(dispatcher)
	go client.write()
	go client.receive(done)
}

// receive pumps messages from the websocket connection to the hub.
//
// The application runs receive in a per-connection goroutine. The application
// ensures that there is at most one reader on a connection by executing all
// reads from this goroutine.
func (c *Client) receive(done chan bool) {
	defer func() {
		done <- true
		c.conn.Close()
		c.psc.Unsubscribe()
		close(c.outbound)
		close(c.inbound)
		log.Println("Receiver terminated", c.ID)
	}()
	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error { c.conn.SetReadDeadline(time.Now().Add(pongWait)); return nil })
	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Println("Client", c.ID, "error: ", err)
			}
			break
		}
		message = bytes.TrimSpace(bytes.Replace(message, newline, space, -1))
		if len(message) > 0 {
			log.Printf("received message: '%s' by client %s", message, c.ID)
			c.inbound <- message
		}
	}
}

// write pumps messages from the hub to the websocket connection.
//
// A goroutine running write is started for each connection. The
// application ensures that there is at most one writer to a connection by
// executing all writes from this goroutine.
func (c *Client) write() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
		log.Println("Writer terminated", c.ID)
	}()
	for {
		select {
		case message, ok := <-c.outbound:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// The hub closed the channel.
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			log.Println("sending message to client ", c.ID)
			w.Write(message)

			// Add queued messages to the current websocket message.
			n := len(c.outbound)
			for i := 0; i < n; i++ {
				log.Println("sending message to client ", c.ID)
				w.Write(<-c.outbound)
			}

			if err := w.Close(); err != nil {
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func (c *Client) process(dispatcher *Dispatcher) {
	for {
		select {
		case message, ok := <-c.inbound:
			if !ok {
				log.Println("Processor terminated", c.ID)
				return
			}

			log.Printf("processing message: '%s'", message)

			var input map[string]interface{}
			var output interface{}
			err := json.Unmarshal(message, &input)
			if err != nil {
				log.Println(err)
				output = NewError("400", "Invalid Request")
			} else if t, ok := input["type"].(string); ok {
				if f, ok := dispatcher.h[t]; ok {
					output = f(c, &input)
				} else {
					output = NewError("400", "Unknown Request '"+t+"'")
				}
			}

			if output != nil {
				response, err := json.Marshal(output)
				if err != nil {
					log.Println("Error marshaling outgoing mesasge", output)
				} else {
					c.outbound <- response
				}
			}
		}
	}
}

// listener receive pubilshed message in redis channel subscribed by the client
func (c *Client) listener(rdb *redis.Pool, filterOut bool, done chan bool) {
	cmap := make(map[string]bool)
	cmap["client:"+c.ID] = true

	for {
		rconn := rdb.Get()
		c.psc = &redis.PubSubConn{Conn: rconn}

		var err error
		for ch := range cmap {
			if err = c.psc.Subscribe(ch); err != nil {
				log.Println("Got error", err)
				break
			}
		}

		log.Println("Client", c.ID, "listening")
	loop:
		for rconn.Err() == nil && err == nil {
			switch x := c.psc.Receive().(type) {
			case error:
				log.Println("Client", c.ID, "; redis Listener got error", x)

			case redis.Message:
				log.Println("Client", c.ID, "received message from channel", x.Channel)
				if filterOut {
					if data := c.filter(x.Data); data != nil {
						c.outbound <- data
					}
				} else if x.Data != nil {
					c.outbound <- x.Data
				}

			case redis.Subscription:
				log.Println("Client", c.ID, "received subsciption", x)
				switch x.Kind {
				case "subscribe", "psubscribe":
					cmap[x.Channel] = true
				case "unsubscribe", "punsubscribe":
					delete(cmap, x.Channel)
					if x.Count == 0 {
						break loop
					}
				}
			}
		}

		rconn.Close()

		select {
		case <-done:
			log.Println("Client", c.ID, "listener terminated")
			return
		default:
			break
		}
	}
}

func (c *Client) filter(data []byte) []byte {
	var m map[string]interface{}
	err := json.Unmarshal(data, &m)
	if err != nil {
		log.Println(err)
		return nil
	}

	if sender, ok := m["sender"].(string); ok && sender == c.ID {
		return nil
	}

	return data
}
