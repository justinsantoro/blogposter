package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
)

const (
	// environment varialble keys
	envKeyConfigAuthor           = "BLOGPOSTER_AUTHOR"
	envKeyConfigPort             = "BLOGPOSTER_PORT"
	envKeyConfigPath             = "BLOGPOSTER_PATH"
	envKeyConfigUsername         = "BLOGPOSTER_USERNAME"
	envKeyConfigToken            = "BLOGPOSTER_USERNAME"
	envKeyConfigTest             = "BLOGPOSTER_TEST"
	envKeyConfigName             = "BLOGPOSTER_NAME"
	envKeyConfigEmail            = "BLOGPOSTER_EMAIL"
	envKeyConfigBaseURL          = "BLOGPOSTER_BASEURL"
	envKeyConfigRemoteURL        = "BLOGPOSTER_REMOTEURL"
	envKeyConfigGAPIPrivateKey   = "GAPI_PRIVATE_KEY"
	envKeyConfigGAPIPrivateKeyID = "GAPI_PRIVATE_KEY_ID"
	envKeyConfigGAPIEmail        = "GAPI_EMAIL"
	envKeyConfigGAPITokenURL     = "GAPI_TOKEN_URL"
)

func main() {
	//log to stdout
	log.SetOutput(os.Stdout)

	test := flag.Bool("t", false, "enable test mode (no push)")
	baseurl := flag.String("BaseURL", "", "hugo server baseurl http://localhost:8080")
	port := flag.String("p", "", "port for http server")
	flag.Parse()

	conf := ServerConfig{
		Author:    os.Getenv(envKeyConfigAuthor),
		Port:      os.Getenv(envKeyConfigPort),
		Path:      os.Getenv(envKeyConfigPath),
		Username:  os.Getenv(envKeyConfigUsername),
		Token:     os.Getenv(envKeyConfigToken),
		Name:      os.Getenv(envKeyConfigName),
		Email:     os.Getenv(envKeyConfigEmail),
		BaseUrl:   os.Getenv(envKeyConfigAuthor),
		RemoteUrl: os.Getenv(envKeyConfigBaseURL),
		GAPI: &GAPIConfig{
			PrivateKeyID: os.Getenv(envKeyConfigGAPIPrivateKeyID),
			PrivateKey:   os.Getenv(envKeyConfigGAPIPrivateKey),
			Email:        os.Getenv(envKeyConfigGAPIEmail),
			TokenURL:     os.Getenv(envKeyConfigGAPITokenURL),
		},
	}
	if test, ok := os.LookupEnv(envKeyConfigTest); ok {
		if test != "0" {
			conf.Test = true
		}
	}
	//override conf with set cmdline flag values
	if *test {
		conf.Test = *test
	}
	if len(*baseurl) > 0 {
		conf.BaseUrl = *baseurl
	}
	if len(*port) > 0 {
		conf.Port = *port
	}

	server := NewServer(&conf)
	server.PostPush = func() error {
		return nil
	}
	ctx, cancel := context.WithCancel(context.Background())
	c, err := server.Start(ctx)
	if err != nil {
		log.Fatal("start: ", err)
	}

	//wait for SIGINT
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
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
