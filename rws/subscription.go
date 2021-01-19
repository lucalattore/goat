package rws

import (
	"fmt"
	"log"
	"strings"
)

// WSChannelParams describes channel customization
type WSChannelParams struct {
	TopicPrefix string
}

// Subscribe is the function to handle subscription to topic
func (p *WSChannelParams) Subscribe(c *Client, r *map[string]interface{}) interface{} {
	if ch, ok := (*r)["topic"].(string); ok {
		if strings.HasPrefix(ch, p.TopicPrefix+":") {
			log.Println("Client", c.ID, "subscribing to topic", ch)
			err := c.psc.Subscribe(ch)
			if err != nil {
				log.Println(err)
			}
		}
	} else if chs, ok := (*r)["topic"].([]interface{}); ok {
		topics := make([]interface{}, 0)
		for _, topic := range chs {
			if strings.HasPrefix(fmt.Sprintf("%v", topic), p.TopicPrefix+":") {
				topics = append(topics, topic)
			}
		}

		if len(topics) > 0 {
			log.Println("Client", c.ID, "subscribing to topics", topics)
			err := c.psc.Subscribe(topics...)
			if err != nil {
				log.Println(err)
			}
		}
	}

	return nil
}

// Unsubscribe is the function to handle unsubscribe from topic
func (p *WSChannelParams) Unsubscribe(c *Client, r *map[string]interface{}) interface{} {
	if ch, ok := (*r)["topic"].(string); ok {
		if strings.HasPrefix(ch, p.TopicPrefix+":") {
			log.Println("Client", c.ID, "unsubscribing from topic", ch)
			err := c.psc.Unsubscribe(ch)
			if err != nil {
				log.Println(err)
			}
		}
	} else if chs, ok := (*r)["topic"].([]interface{}); ok {
		topics := make([]interface{}, 0)
		for _, topic := range chs {
			if strings.HasPrefix(fmt.Sprintf("%v", topic), p.TopicPrefix+":") {
				topics = append(topics, topic)
			}
		}

		if len(topics) > 0 {
			log.Println("Client", c.ID, "unsubscribing from topics", topics)
			err := c.psc.Unsubscribe(topics...)
			if err != nil {
				log.Println(err)
			}
		} else {
			err := c.psc.PUnsubscribe(p.TopicPrefix + ":*")
			if err != nil {
				log.Println(err)
			}
		}
	}

	return nil
}
