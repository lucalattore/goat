package rws

import (
	"fmt"
	"log"
	"strings"
)

// Subscribe is the function to handle subscription to topic
func Subscribe(c *Client, r *map[string]interface{}) interface{} {
	if ch, ok := (*r)["topic"].(string); ok {
		if strings.HasPrefix(ch, "sheet:") {
			log.Println("Client", c.ID, "subscribing to topic", ch)
			err := c.psc.Subscribe(ch)
			if err != nil {
				log.Println(err)
			}
		}
	} else if chs, ok := (*r)["topic"].([]interface{}); ok {
		topics := make([]interface{}, 0)
		for _, topic := range chs {
			if strings.HasPrefix(fmt.Sprintf("%v", topic), "sheet:") {
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
func Unsubscribe(c *Client, r *map[string]interface{}) interface{} {
	if ch, ok := (*r)["topic"].(string); ok {
		if strings.HasPrefix(ch, "sheet:") {
			log.Println("Client", c.ID, "unsubscribing from topic", ch)
			err := c.psc.Unsubscribe(ch)
			if err != nil {
				log.Println(err)
			}
		}
	} else if chs, ok := (*r)["topic"].([]interface{}); ok {
		topics := make([]interface{}, 0)
		for _, topic := range chs {
			if strings.HasPrefix(fmt.Sprintf("%v", topic), "sheet:") {
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
			err := c.psc.PUnsubscribe("sheet:*")
			if err != nil {
				log.Println(err)
			}
		}
	}

	return nil
}
