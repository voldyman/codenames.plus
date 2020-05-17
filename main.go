package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
)

var (
	portFlag = flag.Int("port", 8080, "server port")
)

func main() {
	flag.Parse()
	server := socketServer(NewCodeNames())
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
