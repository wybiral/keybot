package keybot

import (
	"bufio"
	"encoding/json"
	"os/exec"
	"sync"
)

type ChatApi struct {
	w   *bufio.Writer
	r   *bufio.Scanner
	mux sync.Mutex
}

type Message struct {
	Conversation string
	Id           int
	Time         int
	Body         string
	Username     string
	Device       string
}

func NewChatApi() (*ChatApi, error) {
	cmd := exec.Command("keybase", "chat", "api")
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	err = cmd.Start()
	if err != nil {
		return nil, err
	}
	w := bufio.NewWriter(stdin)
	r := bufio.NewScanner(stdout)
	return &ChatApi{w: w, r: r}, nil
}

// Send a message to a conversation
func (api *ChatApi) Send(conv string, msg string) error {
	type message struct {
		Body string `json:"body"`
	}
	type options struct {
		Conversation string  `json:"conversation_id"`
		Message      message `json:"message"`
	}
	type params struct {
		Options options `json:"options"`
	}
	type command struct {
		Method string `json:"method"`
		Params params `json:"params"`
	}
	cmd, err := json.Marshal(command{
		Method: "send",
		Params: params{
			Options: options{
				Conversation: conv,
				Message:      message{Body: msg},
			},
		},
	})
	if err != nil {
		return err
	}
	_, err = api.call(cmd)
	return err
}

// Return readonly channel of all unread messages
func (api *ChatApi) Listen() <-chan Message {
	ch := make(chan Message)
	go func() {
		for {
			conversations, err := api.GetConversations()
			if err != nil {
				continue
			}
			for _, conv := range conversations {
				messages, err := api.GetMessages(conv, false)
				if err != nil {
					continue
				}
				for _, msg := range messages {
					ch <- msg
				}
			}
		}
	}()
	return ch
}

// Return an array of unread conversation IDs
func (api *ChatApi) GetConversations() ([]string, error) {
	request := `{"method":"list","params":{"options":{"unread_only":true}}}`
	response, err := api.call([]byte(request))
	if err != nil {
		return nil, err
	}
	return parseConversations(response)
}

// Unmarshall conversation list and return array of conversation IDs
func parseConversations(bytes []byte) ([]string, error) {
	type conversation struct {
		Id string `json:"id"`
	}
	var response struct {
		Result struct {
			Conversations []conversation `json:"conversations"`
		} `json:"result"`
	}
	err := json.Unmarshal(bytes, &response)
	if err != nil {
		return nil, err
	}
	convs := make([]string, len(response.Result.Conversations))
	for i, conv := range response.Result.Conversations {
		convs[i] = conv.Id
	}
	return convs, nil
}

// Return array of unread messages in conversation
// If peek is true then this won't flag the messages as being read
func (api *ChatApi) GetMessages(conv string, peek bool) ([]Message, error) {
	request, err := createReadRequest(conv, peek)
	if err != nil {
		return nil, err
	}
	response, err := api.call([]byte(request))
	if err != nil {
		return nil, err
	}
	return parseMessages(conv, response)
}

// Create a marshalled JSON request for unread messages for conversation
func createReadRequest(conv string, peek bool) ([]byte, error) {
	type options struct {
		Conversation string `json:"conversation_id"`
		UnreadOnly   bool   `json:"unread_only"`
		Peek         bool   `json:"peek"`
	}
	type params struct {
		Options options `json:"options"`
	}
	type command struct {
		Method string `json:"method"`
		Params params `json:"params"`
	}
	return json.Marshal(command{
		Method: "read",
		Params: params{
			Options: options{
				Conversation: conv,
				UnreadOnly:   true,
				Peek:         peek,
			},
		},
	})
}

// Unmarshall messages and return array
func parseMessages(conv string, bytes []byte) ([]Message, error) {
	type message struct {
		Msg struct {
			Id      int `json:"id"`
			Time    int `json:"sent_at"`
			Content struct {
				Text struct {
					Body string `json:"body"`
				} `json:"text"`
			} `json:"content"`
			Sender struct {
				Username   string `json:"username"`
				DeviceName string `json:"device_name"`
			} `json:"sender"`
		} `json:"msg"`
	}
	var response struct {
		Result struct {
			Messages []message `json:"messages"`
		} `json:"result"`
	}
	err := json.Unmarshal(bytes, &response)
	if err != nil {
		return nil, err
	}
	messages := make([]Message, 0)
	length := len(response.Result.Messages)
	for i := length - 1; i >= 0; i-- {
		msg := response.Result.Messages[i].Msg
		if len(msg.Content.Text.Body) == 0 {
			continue
		}
		messages = append(messages, Message{
			Conversation: conv,
			Id:           msg.Id,
			Time:         msg.Time,
			Body:         msg.Content.Text.Body,
			Username:     msg.Sender.Username,
			Device:       msg.Sender.DeviceName,
		})
	}
	return messages, nil
}

// Make an API call by sending JSON marshalled bytes to keybase and return the
// response as JSON marshalled bytes.
func (api *ChatApi) call(request []byte) ([]byte, error) {
	api.mux.Lock()
	defer api.mux.Unlock()
	_, err := api.w.Write(request)
	if err != nil {
		return nil, err
	}
	err = api.w.Flush()
	if err != nil {
		return nil, err
	}
	api.r.Scan()
	err = api.r.Err()
	if err != nil {
		return nil, err
	}
	return api.r.Bytes(), nil
}
