package web

import (
	"net/http"
	"os/exec"
	"runtime"
	"time"

	"../builder"
)

type Router struct {
	port          string
	level3Builder *builder.Builder
}

func NewRouter(port string, level3Builder *builder.Builder) *Router {
	return &Router{
		port:          port,
		level3Builder: level3Builder,
	}
}

func (router *Router) index(w http.ResponseWriter, r *http.Request) {
	data, err := router.level3Builder.SnapshotBytes()
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
