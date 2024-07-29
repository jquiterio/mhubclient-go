/*
 * @file: client.go
 * @author: Jorge Quitério
 * @copyright (c) 2021 Jorge Quitério
 * @license: MIT
 */

package mhuclientgo

import (
	"crypto/tls"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

type Message struct {
	SubscriberID string `json:"subscriber_id"`
	Topic        string `json:"topic"`
	Payload      []byte `json:"payload"`
}

type HubClient struct {
	SubscriberID string
	Topics       []string
	Handler      func(Message)
	Parser       func(string) (Message, bool)
	Address      *net.TCPAddr
	Conn         *tls.Conn
	Debug        bool
}

func newTlsConfig() *tls.Config {
	cert, err := tls.LoadX509KeyPair("certs/client.pem", "certs/client.key")
	if err != nil {
		panic(err)
	}
	config := tls.Config{
		Certificates:       []tls.Certificate{cert},
		InsecureSkipVerify: true,
	}
	return &config
}

func NewMessage(subscriberID, topic string, payload []byte) *Message {
	return &Message{
		SubscriberID: subscriberID,
		Topic:        topic,
		Payload:      payload,
	}
}

// String message returns message as string
// Format: subscriber_id.topic.payload
func (m *Message) String() string {
	return fmt.Sprintf("%s.%s.%s\n", m.SubscriberID, m.Topic, m.Payload)
}

func NewHubClient(address string) *HubClient {
	addr, err := net.ResolveTCPAddr("tcp", address)
	if err != nil {
		panic(err)
	}
	h := &HubClient{
		Address: addr,
		Debug:   os.Getenv("DEBUB") == "true",
	}
	return h
}

func (h *HubClient) Publish(topic string, payload []byte) {
	defer h.Conn.Close()
	m := NewMessage(h.SubscriberID, topic, payload)
	msg := m.String()
	err := h.Connect()
	if err != nil {
		return
	}
	go h.Conn.Write([]byte(msg))
}

func defaultParser(msg string) (m *Message, ok bool) {
	msgSplit := strings.Split(msg, ".")
	payload := []byte(msgSplit[2] + "." + msgSplit[3])
	m = NewMessage(msgSplit[0], msgSplit[1], payload)
	return m, true
}

func (h *HubClient) parse(msg string) (m *Message, ok bool) {
	if h.Parser == nil {
		return defaultParser(msg)
	}
	*m, ok = h.Parser(msg)
	return m, ok
}

func (h *HubClient) getmessages() {
	//defer h.Conn.Close()
	for {
		b := make([]byte, 1024)
		_, err := h.Conn.Read(b)
		if err != nil {
			if h.Debug {
				println("error reading: ", err)
			}
			break
		} else {
			inMsg := string(b)
			if h.Debug {
				println("got msg: ", inMsg)
			}

			m, ok := h.parse(inMsg)
			if !ok {
				if h.Debug {
					println("error parsing message: ", inMsg)
				}
				return
			}
			go h.Handler(*m)
		}
		//time.Sleep(1 * time.Second)
	}
}

func (h *HubClient) GetMessages() {
	println("subscriber_id: ", h.SubscriberID)
	for {
		// err := h.Connect()
		// if err == nil {
		// 	h.getmessages()
		// }
		if err := h.Connect(); err == nil {
			h.getmessages()
		} else {
			time.Sleep(10 * time.Second)
		}
	}
}

func (h *HubClient) Connect() error {
	config := newTlsConfig()
	addr := net.JoinHostPort(h.Address.IP.String(), strconv.Itoa(h.Address.Port))
	c, err := tls.Dial("tcp", addr, config)
	if err == nil {
		h.Conn = c
	} else if h.Debug {
		println("error connecting: ", err)
		return err
	}
	return nil
}

// USAGE:
/*
	 debug := os.Getenv("DEBUG") == "true"
	 hub_addr := os.Getenv("HUB_ADDR")

	 handler := func(m Message) {
		 if debug {
			 println("sub: ", m.SubscriberID)
			 println("topic: ", m.Topic)
			 println("payload: ", m.Payload)
		 }
	 }
	 h := NewHubClient(hub_addr)
	 h.SubscriberID = "3456"
	 h.Handler = handler
*/
