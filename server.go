package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"
)

var posturlregxp = regexp.MustCompile(`(?m)\/post\/[a-zA-Z0-9]+`)

type ServerConfig struct {
	//Author name to put on posts
	Author string `json:"author"`
	//Port for cms app to listen on
	Port string `json:"port"`
	//Path to blog git repository
	Path string `json:"path"`
	//Github username
	Username string `json:"username"`
	//Github token
	Token string `json:"token"`
	//Git config name
	Name string `json:"name"`
	//Git
	Email string `json:"email"`
	//enable test mode
	Test bool `json:"test"`
	//base url for hugo test server
	BaseUrl string `json:"baseurl"`
	//Remote url to git repository
	RemoteUrl string `json:"remoteurl"`
}

type postpushfunc func() error

var defaultpostpush postpushfunc = func() error {
	return nil
}

func ReadServerConfig(fpath string) (*ServerConfig, error) {
	log.Println("reading config file " + fpath)
	b, err := ioutil.ReadFile(fpath)
	if err != nil {
		return nil, err
	}
	conf := new(ServerConfig)
	err = json.Unmarshal(b, conf)
	return conf, err
}

type server struct {
	stopped chan struct{}
	hugo    *HugoRepo
	config  *ServerConfig
	PostPush postpushfunc
}

func NewServer(config *ServerConfig) *server {
	return &server{config: config, stopped: make(chan struct{}), PostPush: defaultpostpush}
}

func (s *server) Start(ctx context.Context) (chan error, error) {
	if s.config == nil {
		log.Fatal("server config is nil")
	}

	var err error

	//start servers
	//initialize repo

	//check if repo exists first, if not: clone it
	ok, err := func () (bool, error) {
		_, err := os.Stat(s.config.Path)
		if err == nil { return true, nil }
		if os.IsNotExist(err) { return false, nil }
		return false, err
	}()
	if err != nil {
		log.Fatal("error checking for repo dir: ", err)
	}
	if !ok {
		//get the repo
		if CloneRepo(s.config.RemoteUrl, s.config.Path) != nil {
			log.Fatal("error cloning repo from remoteurl: ", err)
		}
	}

	s.hugo, err = NewHugoRepo(s.config.Path, s.config.Username, s.config.Token, s.config.BaseUrl, s.config.Name, s.config.Email)
	if err != nil {
		return nil, errors.New("error initializing repo: " + err.Error())
	}
	s.hugo.test = s.config.Test
	//start hugo test server
	hugoErr, err := s.hugo.StartServer(ctx, s.stopped)
	if err != nil {
		return nil, errors.New("error starting hugo test server: " + err.Error())
	}
	//start cms webserver
	go s.startHttpServer(s.config.Port)
	return hugoErr, nil
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

func (s *server) startHttpServer(port string) {

	http.HandleFunc("/upload", func(w http.ResponseWriter, req *http.Request) {
		success := func(prefix string, err error) bool {
			return !serverError(fmt.Sprintf("error handling upload: %s: %%s", prefix), w, err)
		}

		err := req.ParseMultipartForm(32 << 20) // limit your max input length!
		if !success("parse form", err) {
			return
		}
		//get file from form
		file, _, err := req.FormFile("userfile")
		if !success("get userfile", err) {
			return
		}
		defer file.Close()

		//get front matter from form
		title := strings.TrimSpace(req.FormValue("title"))
		tags := strings.ToLower(strings.TrimSpace(req.FormValue("tags")))
		summary := strings.TrimSpace(req.FormValue("summary"))

		//create post in repo
		if !success("hugo new", s.hugo.New(file, title, tags, summary, s.config.Author)) {
			return
		}
		//wait for hugo to rebuild
		//TODO: add a channel for this?
		time.Sleep(time.Second * 4)
		//execute publish template
		http.Redirect(w, req, fmt.Sprintf("/post/%s/", s.hugo.onDeck), int(http.StatusTemporaryRedirect))
	})

	http.HandleFunc("/publish", func(w http.ResponseWriter, req *http.Request) {
		success := func(err error) bool {
			return !serverError("error handling publish: %s", w, err)
		}
		post := s.hugo.onDeck
		if !success(s.hugo.Deploy()) {
			return
		}
		if !s.config.Test {
			if !success(s.PostPush()) {
				return
			}
		}
		msg := fmt.Sprint("successfully published ", post)
		_, err := w.Write([]byte(msg))
		if err != nil {
			log.Println("error writing success to publish response: ", err)
		}
	})

	http.HandleFunc("/abort", func(w http.ResponseWriter, req *http.Request) {
		post := req.URL.Query().Get("post")
		success := func(err error) bool {
			return !serverError("error handling abort: %s", w, err)
		}
		if !success(s.hugo.Abort()) {
			return
		}
		msg := fmt.Sprint("successfully aborted publishing ", post)
		_, err := w.Write([]byte(msg))
		if err != nil {
			log.Println("error writing success to abort response: ", err)
		}
	})

	http.Handle("/new/", http.StripPrefix("/new/", http.FileServer(http.Dir("./static"))))
	proxy := httputil.NewSingleHostReverseProxy(&url.URL{
		Scheme: "http",
		Host: "localhost:1313",
	})
	proxy.ModifyResponse = func(response *http.Response) error {
		response.Header.Set("Access-Control-Allow-Origin", "*")
		if response.StatusCode != http.StatusOK {
			return nil
		}
		//get original request url
		url := response.Request.URL

		//if url == basepath then this is returning homepage, add the "new post button"
		log.Println("requested url: ", url.String())
		if url.String() == "http://localhost:1313/" {
			log.Println("injecting html into response")
			doc, err := goquery.NewDocumentFromReader(response.Body)
			if err != nil {
				return err
			}
			doc.Find("#navMenu").AppendHtml(`<li class="theme-switch-item">
            <a href="/new/" title="New Post">
                <i class="fa fa-file fa-fw" aria-hidden="true"></i>
            </a>
        </li>`)
			html, err := doc.Html()
			if err != nil {
				return err
			}
			response.Body = ioutil.NopCloser(strings.NewReader(html))
			response.Header["Content-Length"] = []string{fmt.Sprint(len(html))}
		}
		if posturlregxp.MatchString(url.String()) {
			//if this is a post
			urlparts := strings.Split(string(url.String()[:len(url.String())-1]), "/")
			postname := urlparts[len(urlparts)-1]
			if postname == s.hugo.onDeck {
				doc, err := goquery.NewDocumentFromReader(response.Body)
				if err != nil {
					return err
				}//inject abort / publish buttons
				doc.Find("#navMenu").AppendHtml(`<li class="theme-switch-item">
            <a href="/publish" title="Publish Post">
                <i class="fa fa-paper-plane fa-fw" aria-hidden="true"></i>
            </a>
        	</li>
			<li class="theme-switch-item">
            <a href="/abort" title="Abort Publish">
                <i class="fa fa-ban fa-fw" aria-hidden="true"></i>
            </a>
        	</li>`)
				html, err := doc.Html()
				if err != nil {
					return err
				}
				response.Body = ioutil.NopCloser(strings.NewReader(html))
				response.Header["Content-Length"] = []string{fmt.Sprint(len(html))}
			}
		}
		return nil
	}
	http.Handle("/", proxy)

	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", s.config.Port), nil))
}
