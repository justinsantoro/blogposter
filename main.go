package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
)

func main() {

	//test := flag.Bool("t", false, "enable test mode (no push)")
	//flag.Parse()

	server := NewServer(&ServerConfig{
		Author:   "Sarah",
		Port:     "8080",
		Path:     "F:\\Sarah\\Repos\\smalltownkitten",
		Username: "sarahlehman",
		Token:    "",
		Test: true,
		Name: "Sarah Lehman",
		Email: "sarah@smalltownkitten.com",
		BaseUrl: "http://192.168.1.101:8080",
		RemoteUrl: "https://github.com/sarahlehman/smalltownkitten",
	})

	PandocLoc = "C:\\Users\\jzs\\scoop\\shims\\pandoc.exe"

	ctx, cancel := context.WithCancel(context.Background())
	c, err := server.Start(ctx)
	if err != nil {
		log.Fatal("start: ", err)
	}

	//wait for SIGINT
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT)
	log.Println("waiting")

	select {
	case <-sig:
		log.Println("received SIGINT. Shutting down...")
		cancel()
		<-server.stopped
	case err := <-c:
		log.Fatal("hugo server stopped unexpectedly: ", err)
	}
}
