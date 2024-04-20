package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"

	"github.com/markbates/pkger"
	"github.com/sirupsen/logrus"
)

var (
	listenAll = flag.Bool("all", false, "listen to any address or just localhost. localhost by default")
	portFlag  = flag.Int("port", 8080, "server port")
)

var log = logrus.New()

func main() {
	flag.Parse()
	log.Out = os.Stdout
	pkger.Include("/server")
	pkger.Include("/public")

	server := socketServer(NewActionRouter())
	go func() {
		if err := server.Serve(); err != nil {
			log.Fatalf("socketio listen error: %s\n", err)
		}
	}()
	defer server.Close()

	http.Handle("/socket.io/", server)
	http.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		// original API pinged, keep it?
		w.WriteHeader(http.StatusOK)
	})
	http.Handle("/", http.FileServer(pkger.Dir("/public")))

	addr := fmt.Sprintf("localhost:%d", *portFlag)
	if *listenAll {
		addr = fmt.Sprintf(":%d", *portFlag)
	}
	fmt.Printf("Listening on http://%s\n", addr)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", *portFlag), nil))
}
