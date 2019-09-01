package main

import (
	"flag"
	"../builder"
	"../log"
	"../web"
	"github.com/Kucoin/kucoin-go-sdk"

)

func main() {
	symbol, port := getArgs()

	apiService := kucoin.NewApiServiceFromEnv()
	l3OrderBook := builder.NewBuilder(apiService, symbol)
	go l3OrderBook.ReloadOrderBook()

	r := web.NewRouter(port, l3OrderBook)
	go r.Handle()

	websocket(apiService, symbol, l3OrderBook)
	defer log.CloseLogger()
}

func getArgs() (string, string) {
	symbol := flag.String("s", "BTC-USDT", "symbol")
	port := flag.String("p", "9090", "port")
	flag.Parse()

	return *symbol, *port
}

func websocket(apiService *kucoin.ApiService, symbol string, level3Builder *builder.Builder) {
	rsp, err := apiService.WebSocketPublicToken()
	if err != nil {
		panic(err)
	}

	tk := &kucoin.WebSocketTokenModel{}
	if err := rsp.ReadData(tk); err != nil {
		panic(err)
	}

	c := apiService.NewWebSocketClient(tk)

	mc, ec, err := c.Connect()
	if err != nil {
		panic(err)
	}

	ch := kucoin.NewSubscribeMessage("/market/level3:"+symbol, false)
	if err := c.Subscribe(ch); err != nil {
		panic(err)
	}

	for {
		select {
		case err := <-ec:
			c.Stop() // Stop subscribing the WebSocket feed
			panic(err)

		case msg := <-mc:
			log.Info("received msg: %s", kucoin.ToJsonString(msg))
			level3Builder.Messages <- msg.RawData
		}
	}
}
