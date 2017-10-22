package main

import (
	"fmt"
	"github.com/wybiral/keybot"
	"log"
)

func main() {
	// Create new chat API
	chat, err := keybot.NewChatApi()
	if err != nil {
		log.Fatal(err)
	}
	// Get all unread conversations
	convs, err := chat.GetConversations()
	if err != nil {
		log.Fatal(err)
	}
	msg_count := 0
	conv_count := len(convs)
	for _, conv := range convs {
		// Get all unread messages from a conversation
		msgs, err := chat.GetMessages(conv)
		if err != nil {
			log.Fatal(err)
		}
		msg_count += len(msgs)
	}
	fmt.Printf("You have %d unread message(s) ", msg_count)
	fmt.Printf("from %d conversation(s)\n", conv_count)
}
