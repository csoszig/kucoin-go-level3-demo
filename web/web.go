package web

import (
	"net/http"
	"os/exec"
	"runtime"
	"time"

	order_book "github.com/Kucoin/kucoin-go-level3-demo"
)

type Router struct {
	port        string
	l3OrderBook *order_book.Level3OrderBook
}

func NewRouter(port string, l3OrderBook *order_book.Level3OrderBook) *Router {
	return &Router{
		port:        port,
		l3OrderBook: l3OrderBook,
	}
}

func (router *Router) index(w http.ResponseWriter, r *http.Request) {
	data, err := router.l3OrderBook.SnapshotBytes()
	if err != nil {
		w.Write([]byte(err.Error()))
		return
	}

	w.Write(data)
	return
}

func (router *Router) Handle() {
	http.HandleFunc("/", router.index)

	if runtime.GOOS == "darwin" {
		go func() {
			time.Sleep(time.Second)
			exec.Command("open", "http://localhost:"+router.port).Run()
		}()
	}

	if err := http.ListenAndServe(":"+router.port, nil); err != nil {
		panic(err)
	}
}
