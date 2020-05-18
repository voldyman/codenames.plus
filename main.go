package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"

	"github.com/sirupsen/logrus"
)

var (
	portFlag = flag.Int("port", 8080, "server port")
)

var log = logrus.New()

func main() {
	flag.Parse()
	log.Out = os.Stdout

	server := socketServer(NewActionRouter())
	go server.Serve()
	defer server.Close()

	http.Handle("/socket.io/", server)
	http.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		// original API pinged, keep it?
		w.WriteHeader(http.StatusOK)
	})
	http.Handle("/", http.FileServer(http.Dir("public/")))

	fmt.Printf("Listening on http://localhost:%d\n", *portFlag)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", *portFlag), nil))
}
