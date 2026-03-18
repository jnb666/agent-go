// Web based chat interface with tool calling
package main

import (
	"context"
	"embed"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
)

//go:embed assets
var assets embed.FS

func main() {
	port := 8000
	var serve, debug bool
	flag.IntVar(&port, "port", port, "listen port for web server")
	flag.BoolVar(&serve, "serve", false, "serve webserver on all interfaces - default is localhost only")
	flag.BoolVar(&debug, "debug", false, "enable debug logging")
	flag.Parse()
	if debug {
		log.SetLevel(log.DebugLevel)
	}

	http.Handle("/", fsHandler())
	ctx, wsCancel := context.WithCancel(context.Background())
	http.HandleFunc("/websocket", websocketHandler(ctx))

	// launch web server in background
	var server http.Server
	go func() {
		var host string
		if serve {
			host, _ = os.Hostname()
			server.Addr = fmt.Sprintf(":%d", port)
		} else {
			host = "localhost"
			server.Addr = fmt.Sprintf("localhost:%d", port)
		}
		log.Infof("Serving website at http://%s:%d", host, port)
		err := server.ListenAndServe()
		if !errors.Is(err, http.ErrServerClosed) {
			log.Fatal("HTTP server error: ", err)
		}
	}()

	// shutdown cleanly on signal
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	<-ch
	wsCancel()
	time.Sleep(100 * time.Millisecond)
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Fatal("HTTP shutdown error: ", err)
	}
	time.Sleep(time.Second)
	log.Info("server shutdown")
}

// handler for websocket connections
func websocketHandler(ctx context.Context) http.HandlerFunc {
	var upgrader websocket.Upgrader

	return func(w http.ResponseWriter, r *http.Request) {
		ws, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Error("websocket upgrade: ", err)
			return
		}
		defer func() {
			log.Debugf("%s websocket disconnected", ws.RemoteAddr())
			ws.Close()
		}()

		log.Debugf("%s websocket connected", ws.RemoteAddr())
		client := newClient(ctx, ws)
		defer client.close()
		client.run(ctx)
		if client.error != nil {
			log.Error(client.error)
		}
	}
}

// handler to server static embedded files
func fsHandler() http.Handler {
	sub, err := fs.Sub(assets, "assets")
	if err != nil {
		panic(err)
	}
	return http.FileServer(http.FS(sub))
}
