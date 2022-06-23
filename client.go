/*
 * @file: client.go
 * @author: Jorge Quitério
 * @copyright (c) 2021 Jorge Quitério
 * @license: MIT
 */

package mhuclientgo

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	"github.com/golang/glog"
	"github.com/jquiterio/uuid"
)

type Message struct {
	Topic   string `json:"topic"`
	Payload string `json:"payload"`
}

// func NewMessage(subscriberID, topic, payload string) *Message {
// 	return &Message{
// 		SubscriberID: subscriberID,
// 		ID:           uuid.NewV4().String(),
// 		Topic:        topic,
// 		Data:         data,
// 	}
// }

// func (m *Message) FromMap(msg map[string]interface{}) error {
// 	m.SubscriberID = msg["subscriber_id"].(string)
// 	m.ID = msg["id"].(string)
// 	m.Payload = msg["payload"].(string)
// 	m.Topic = msg["topic"].(string)
// 	return nil
// }

// func (m *Message) ToMap() map[string]interface{} {
// 	return map[string]interface{}{
// 		"subscriber_id": m.SubscriberID,
// 		"id":            m.ID,
// 		"topic":         m.Topic,
// 		"payload":       m.Payload,
// 	}
// }

// func (m *Message) ToJSON() ([]byte, error) {
// 	return json.Marshal(m.ToMap())
// }

type Client struct {
	ClientID       string
	Topics         []string
	HubAddr        string
	MessageHandler func(msg string)
	Conn           *http.Client
	Secure         bool
	Debug          bool
}

func tlsCconfig(ca, crt, key string) (*tls.Config, error) {
	certPool := x509.NewCertPool()
	pem, err := ioutil.ReadFile(ca)
	if err != nil {
		return nil, err
	}
	if ok := certPool.AppendCertsFromPEM(pem); !ok {
		return nil, fmt.Errorf("cannot parse CA certificate")
	}
	cert, err := tls.LoadX509KeyPair(crt, key)
	if err != nil {
		return nil, err
	}
	return &tls.Config{
		RootCAs:      certPool,
		Certificates: []tls.Certificate{cert},
	}, nil
}

func NewHubClient(address string, secure bool) (*Client, error) {
	if len(address) == 0 {
		return nil, fmt.Errorf("no hub address")
	}
	var conn *http.Client
	var proto string
	if secure {
		proto = "https"
		tlsconfig, err := tlsCconfig("ca.pem", "client.pem", "client.key")
		if err != nil {
			glog.Fatal("Error reading certs with files: ca.pem, client.crt, client.key")
			return nil, err
		}
		conn = &http.Client{Transport: &http.Transport{TLSClientConfig: tlsconfig}}
	} else {
		proto = "http"
		conn = http.DefaultClient
	}
	a, err := url.Parse(proto + "://" + address)
	if err != nil {
		glog.Fatal("Hub address must be a valid URL")
		return nil, err
	}

	return &Client{
		HubAddr:  a.String(),
		ClientID: uuid.NewV4().String(),
		Conn:     conn,
	}, nil
}

func (c *Client) AddTopic(topic []string) (ok bool) {
	if len(topic) == 0 {
		glog.Error("no topics to add")
		return
	}
	c.Topics = append(c.Topics, topic...)
	return true
}

func (c *Client) Subscribe() (ok bool) {
	url := fmt.Sprintf("%s/subscribe", c.HubAddr)
	var body []byte
	var topics string
	if len(c.Topics) == 0 {
		glog.Error("no topics to subscribe")
		return false
	} else if len(c.Topics) > 0 {
		topics = strings.Join(c.Topics, ",")
		body = []byte(topics)
	}
	fmt.Println("Topics:" + topics)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		glog.Fatal(err)
		return
	}
	req.Header.Set("X-Subscriber-ID", c.ClientID)
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.Conn.Do(req)
	if err != nil {
		glog.Fatal(err)
		return
	}
	if resp.StatusCode != http.StatusOK {
		glog.Fatal("unexpected status code: ", resp.StatusCode)
	}
	return true
}

func (c *Client) Unsubscribe(topics []string) (ok bool) {
	var url string
	if len(topics) == 0 {
		glog.Error("no topics to unsubscribe")
		return
	}
	if len(topics) > 1 {
		url = fmt.Sprintf("%s/unsubscribe", c.HubAddr)
	} else {
		url = fmt.Sprintf("%s/unsubscribe/%s", c.HubAddr, topics[0])
	}
	body, _ := json.Marshal(topics)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		glog.Fatal(err)
	}
	req.Header.Set("X-Subscriber-ID", c.ClientID)
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.Conn.Do(req)
	if err != nil {
		glog.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		glog.Fatal("unexpected status code: ", resp.StatusCode)
	}
	return true
}

func (c *Client) Publish(topic, payload string) {
	url := fmt.Sprintf("%s/publish/%s", c.HubAddr, topic)
	//message := NewMessage(c.ClientID, topic, "publish", msg)
	body := []byte(payload)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		glog.Fatal(err)
	}
	req.Header.Set("X-Subscriber-ID", c.ClientID)
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.Conn.Do(req)
	if err != nil {
		glog.Fatal(err)
	}
	if resp.StatusCode != http.StatusCreated {
		glog.Fatal("unexpected status code: ", resp.StatusCode)
	}
}

// func (c *Client) PublishPlain(msg string) {
// 	var topic string
// 	msgSplit := strings.Split(msg, ".")
// 	if len(msgSplit) == 3 {
// 		topic = msgSplit[0]
// 		action := msgSplit[1]
// 		objid := msgSplit[2]
// 		msg := Message{
// 			SubscriberID: c.ClientID,
// 			Topic:        topic,
// 			Data:         action + "." + objid,
// 		}
// 		body, err := msg.ToJSON()

// 		if err != nil {
// 			glog.Fatal(err)
// 		}
// 		url := fmt.Sprintf("%s/publish/%s", c.HubAddr, topic)

// 		req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
// 		if err != nil {
// 			glog.Fatal(err)
// 		}
// 		req.Header.Set("X-Subscriber-ID", c.ClientID)
// 		req.Header.Set("Content-Type", "application/json")
// 		resp, err := c.Conn.Do(req)
// 		if err != nil {
// 			glog.Fatal(err)
// 		}
// 		if resp.StatusCode != http.StatusCreated {
// 			glog.Fatal("unexpected status code: ", resp.StatusCode)
// 		}
// 	}
// }

func (c *Client) GetPlainMessages() {
	url := c.HubAddr
	req, err := http.NewRequest("GET", url, nil)
	if err == nil {
		req.Header.Set("X-Subscriber-ID", c.ClientID)
		resp, err := c.Conn.Do(req)
		for {
			if err == nil {
				if resp.StatusCode == http.StatusOK {
					bodybytes, err := io.ReadAll(resp.Body)
					if err == nil {
						msg := string(bodybytes)
						fmt.Println(msg)
						// msgSplit := strings.Split(msg, ".")
						// if len(msgSplit) == 3 {
						// 	topic := msgSplit[0]
						// 	action := msgSplit[1]
						// 	objid := msgSplit[2]
						// 	if c.MessageHandler != nil {
						// 		message := Message{
						// 			Topic: topic,
						// 			Data:  action + "." + objid,
						// 		}
						// 		c.MessageHandler(message)
						// 	}
						// }
						c.MessageHandler(msg)
					}
				}
			}
		}
	}
}

// func (c *Client) GetMessages() {
// 	url := c.HubAddr
// 	req, err := http.NewRequest("GET", url, nil)
// 	if err != nil {
// 		glog.Info("Error on creating the Request ", url, ":", err)
// 	}
// 	req.Header.Set("X-Subscriber-ID", c.ClientID)
// 	req.Header.Set("Content-Type", "application/json")
// 	resp, err := c.Conn.Do(req)
// 	if err != nil {
// 		glog.Info("Error on sending the Request ", url, ":", err)
// 	}
// 	dec := json.NewDecoder(resp.Body)
// 	for {
// 		var message Message
// 		err := dec.Decode(&message)
// 		if err == nil {
// 			if c.MessageHandler != nil {
// 				c.MessageHandler(message)
// 			}
// 			if c.Debug {
// 				glog.Info(fmt.Sprintf("%s: %s", message.Topic, message.Data))
// 			}
// 		}
// 	}
// }

func (c *Client) Me() {
	url := fmt.Sprintf("%s/me", c.HubAddr)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		glog.Fatal("Error on creating the Request ", url, ":", err)
	}
	req.Header.Set("X-Subscriber-ID", c.ClientID)
	resp, err := c.Conn.Do(req)
	if err != nil {
		glog.Fatal("Error on sending the Request ", url, ":", err)
	}
	dec := json.NewDecoder(resp.Body)
	for {
		var message interface{}
		err := dec.Decode(&message)
		if err != nil {
			if err == io.EOF {
				break
			}
			glog.Fatal("Error on decoding the Response message: ", err)
		}
	}
}
