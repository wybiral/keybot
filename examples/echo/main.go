package main

import (
	"github.com/wybiral/keybot"
	"log"
)

func main() {
	// Create new chat API
	chat, err := keybot.NewChatApi()
	if err != nil {
		log.Fatal(err)
	}
	// Listen for messages (forever)
	for msg := range chat.Listen() {
		log.Println(msg.Username + ": " + msg.Body)
		// Echo message back to the conversation
		err = chat.Send(msg.Conversation, msg.Body)
		if err != nil {
			log.Fatal(err)
		}
	}
}
