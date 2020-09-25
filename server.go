package main

import (
	"context"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"sync"
)

const publishTemplate = `
<!DOCTYPE html>
<html lang="en">
<head>
	<meta charset="UTF-8">
	<meta name="viewport" content="width=device-width, initial-scale=1.0">
	<title>Document</title>
</head>
<body onload="window.open('/content/post/{{ .Name }}', '_blank');">>
	<h1>Publish {{.Name}}?</h1>
	<p><button onclick="window.location('/publish?post={{ .Name }}');">Publish</button></p>
	<p><button onclick="window.location('/abort?post={{ .Name }}');">Abort</button></p>
</body>
</html>
`

var t = template.Must(template.New("publish").Parse(publishTemplate))

type server struct {
	*sync.WaitGroup
	hugo *HugoRepo
}

func serverError(msgTemplate string, w http.ResponseWriter, err error) bool {
	if err != nil {
		//write error
		msg := fmt.Sprintf(msgTemplate, err)
		log.Println(msg)
		w.WriteHeader(http.StatusInternalServerError)
		_, err = w.Write([]byte(msg))
		if err != nil {
			log.Println("error writing error message to response body: %s", err)
		}
		return true
	}
	return false
}

func (s *server) startHttpServer (ctx context.Context, port string) error {

	http.HandleFunc("/upload", func(w http.ResponseWriter, req *http.Request) {
		success := func(err error) bool {
			return !serverError("error handling upload %s", w, err)
		}

		err := req.ParseMultipartForm(32 << 20) // limit your max input length!
		//get file from form
		file, _, err := req.FormFile("userfile")
		if !success(err) {return}
		defer file.Close()

		//get front matter from form
		title := strings.TrimSpace(req.FormValue("title"))
		tags := strings.ToLower(strings.TrimSpace(req.FormValue("tags")))
		description := strings.TrimSpace(req.FormValue("description"))

		//create post in repo
		if !success(s.hugo.New(file, title, tags, description, "Sarah Lehman")) {
			return
		}

		if !success(t.Execute(w, struct{Name string}{s.hugo.onDeck})) {
			return
		}
		w.Header().Add("content-type", "text/html")
		log.Printf("sucessfully staged %s for publishing", s.hugo.onDeck)
	})

	http.HandleFunc("/publish", func(w http.ResponseWriter, req *http.Request) {
		post := req.URL.Query().Get("post")
		success := func(err error) bool {
			return ! serverError( "error handling publish: %s", w, err)
		}
		if !success(s.hugo.Deploy()) {return}
		msg := fmt.Sprint("successfully published ", post)
		_, err := w.Write([]byte(msg))
		if err != nil {
			log.Println("error writing success to publish repsonse: ", err)
		}
	})

	http.HandleFunc("/abort", func(w http.ResponseWriter, req *http.Request) {
		post := req.URL.Query().Get("post")
		success := func(err error) bool {
			return ! serverError( "error handling abort: %s", w, err)
		}
		if !success(s.hugo.Abort()) {return}
		msg := fmt.Sprint("successfully aborted publishing ", post)
		_, err := w.Write([]byte(msg))
		if err != nil {
			log.Println("error writing success to abort repsonse: ", err)
		}
	})

	http.Handle("/new/", http.FileServer(http.Dir("./static")))

	hugoServer, err := url.Parse("127.0.0.1:1313")
	if err != nil {
		log.Fatal("error parsing local hugo url:", err)
	}
	http.Handle("/", httputil.NewSingleHostReverseProxy(hugoServer))
	log.Fatal(http.ListenAndServe(":8080", nil))
}
