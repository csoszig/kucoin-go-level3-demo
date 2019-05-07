package main

import (
	"flag"

	order_book "github.com/Kucoin/kucoin-go-level3-demo"
	"github.com/Kucoin/kucoin-go-level3-demo/web"
	"github.com/Kucoin/kucoin-go-sdk"
)

func main() {
	symbol, port := getArgs()

	apiService := kucoin.NewApiServiceFromEnv()
	l3OrderBook := order_book.NewLevel3OrderBook(apiService, symbol)
	go l3OrderBook.ReloadOrderBook()

	r := web.NewRouter(port, l3OrderBook)
	go r.Handle()

	websocket(apiService, l3OrderBook)
}

func getArgs() (string, string) {
	symbol := flag.String("s", "BTC-USDT", "symbol")
	port := flag.String("p", "9090", "port")
	flag.Parse()

	return *symbol, *port
}

func websocket(apiService *kucoin.ApiService, l3OrderBook *order_book.Level3OrderBook) {
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

	ch := kucoin.NewSubscribeMessage("/market/level3:"+l3OrderBook.Symbol(), false)
	if err := c.Subscribe(ch); err != nil {
		panic(err)
	}

	for {
		select {
		case err := <-ec:
			c.Stop() // Stop subscribing the WebSocket feed
			panic(err)

		case msg := <-mc:
			//helper.Info("raw message : %s", kucoin.ToJsonString(msg))
			l3OrderBook.Messages <- msg.RawData
		}
	}
}
