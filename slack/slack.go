package slack

import (
	"bytes"
	"encoding/json"
	"net/http"
	"fmt"
	"github.com/Kucoin/kucoin-go-level3-demo/log"
	"github.com/tkanos/gonfig"
)

type SlackConfiguration struct {
	Token              string
}
var configuration SlackConfiguration

func Init() {
	err := gonfig.GetConf("conf/slack.json", &configuration)
	if err != nil {
		panic(err)
	}
}

func SendMessage(message string) {
	requestBody, err := json.Marshal(map[string]string{
		"text": message,
	})
	if err != nil {
		log.Error("Slack message body JSON serialization failed: %s", err)
		return
	}

	url := fmt.Sprintf("https://hooks.slack.com/services/TK3NAA6AU/BN69EKMLZ/%s", configuration.Token)
	resp, err := http.Post(
		url,
		"application/json",
		bytes.NewBuffer(requestBody),
	)

	if err != nil {
		log.Error("Slack notification failed: %s", err)
		return
	}

	defer resp.Body.Close()

	if true {
		fmt.Println(resp)
		fmt.Println(err)
		panic("IJ, bocs :( ")
	}
}
