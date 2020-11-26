package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"path"
	"syscall"
)

func main() {
	usrDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatal("error getting user home dir: ", err)
	}
	configFile := path.Join(usrDir, "blogposter/config.json")

	test := flag.Bool("t", false, "enable test mode (no push)")
	config := flag.String("config", "", "path to config file (default: $HOME/blogposter/config.json)")
	baseurl := flag.String("BaseURL", "", "hugo server baseurl http://localhost:8080")
	port := flag.String ("p", "", "port for http server")
	flag.Parse()

	if len(*config) > 0 {
		configFile = *config
	}

	conf, err := ReadServerConfig(configFile)
	if err != nil {
		log.Fatal("error reading config file: ", err)
	}

	//override conf with set cmdline flag values
	if (*test) {
		conf.Test = *test
	}
	if (len(*baseurl) > 0) {
		conf.BaseUrl = *baseurl
	}
	if (len(*port) > 0) {
		conf.Port = *port
	}

	server := NewServer(conf)
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
