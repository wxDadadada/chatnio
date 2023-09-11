package api

import (
	"chat/types"
	"chat/utils"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"github.com/spf13/viper"
	"io"
	"log"
	"net/http"
	"strings"
)

func processLine(buf []byte) []string {
	data := strings.Trim(string(buf), "\n")
	rep := strings.NewReplacer(
		"data: {",
		"\"data\": {",
	)
	data = rep.Replace(data)
	array := strings.Split(data, "\n")
	resp := make([]string, 0)
	for _, item := range array {
		item = strings.TrimSpace(item)
		if !strings.HasPrefix(item, "{") {
			item = "{" + item
		}
		if !strings.HasSuffix(item, "}}") {
			item = item + "}"
		}

		if item == "{data: [DONE]}" || item == "{data: [DONE]}}" || item == "{[DONE]}" {
			break
		} else if item == "{data:}" || item == "{data:}}" {
			continue
		}

		var form types.ChatGPTStreamResponse
		if err := json.Unmarshal([]byte(item), &form); err != nil {
			if err := json.Unmarshal([]byte(item[:len(item)-1]), &form); err != nil {
				log.Println(item, err)
			}
		}
		choices := form.Data.Choices
		if len(choices) > 0 {
			resp = append(resp, choices[0].Delta.Content)
		}
	}
	return resp
}

func NativeStreamRequest(model string, endpoint string, apikeys string, messages []types.ChatGPTMessage, token int, callback func(string)) {
	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

	client := &http.Client{}
	req, err := http.NewRequest("POST", endpoint+"/chat/completions", utils.ConvertBody(types.ChatGPTRequest{
		Model:    strings.Replace(model, "reverse", "free", -1),
		Messages: messages,
		MaxToken: token,
		Stream:   true,
	}))
	if err != nil {
		fmt.Println(err)
		return
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+GetRandomKey(apikeys))

	res, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer res.Body.Close()

	for {
		buf := make([]byte, 20480)
		n, err := res.Body.Read(buf)

		if err == io.EOF {
			break
		}
		if err != nil {
			log.Println(err)
		}

		for _, item := range processLine(buf[:n]) {
			callback(item)
		}
	}
}

func StreamRequest(enableGPT4 bool, isProPlan bool, messages []types.ChatGPTMessage, token int, callback func(string)) {
	var model string
	if isProPlan {
		model = "gpt-4-reverse" // using reverse engine
	} else {
		model = "gpt-4"
	}
	if enableGPT4 {
		NativeStreamRequest(model, viper.GetString("openai.gpt4_endpoint"), viper.GetString("openai.gpt4"), messages, token, callback)
	} else {
		NativeStreamRequest("gpt-3.5-turbo-16k-0613", viper.GetString("openai.user_endpoint"), viper.GetString("openai.user"), messages, token, callback)
	}
}
