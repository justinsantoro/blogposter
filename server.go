package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"google.golang.org/api/drive/v3"
)

var posturlregxp = regexp.MustCompile(`(?m)\/post\/[a-zA-Z0-9]+`)
var assetextregexp = regexp.MustCompile(`(?m)(?:(?:.png)|(?:.css)|(?:.js))`)

var input = template.Must(template.New("input").Parse(`<!DOCTYPE html>
<html lang="en">
    <head>
        <meta charset="UTF-8">
        <meta name="viewport" content="width=device-width, initial-scale=1.0">
        <title>Document</title>
    </head>
    <style>
        input {
            width: 100%;
        }
    </style>
    <body>
        <form action="{{ .Action }}" method="post" enctype="multipart/form-data">
            <label for="articleTitle">Post Title:</label>
            <input type="text" id="articleTitle" name="title" value="{{ .Fm.Title }}"> <br>
            <label for="articleDescription">Summary:</label>
            <input type="text" id="articleSummary" name="summary" value="{{ .Fm.Summary }}"> <br>
            <label for="articleTags">Tags:</label>
            <input type="text" id="articleTags" name="tags" value="{{ .Fm.TagList }}"> <br>
            <label for="fileinput">File:</label>
			{{if .DriveFiles }}
			<select id="fileinput" name="drivefile">
				{{ range .DriveFiles }}
				<option value="{{.Id}}">{{.Name}}</option>
				{{end}}
			</select><br>
			<a href="{{.CurrentPath}}?upload=true">direct upload</a>
			{{ else }}
			<input type="file" id="fileinput" name="userfile"> <br>
			{{ end }}
            
            <input type="submit" id="btnSubmit">
			<input type="hidden" name="postname" value="{{ .Postname }}">
        </form>
    </body>
</html>`))

type InputForm struct {
	Action     string
	Fm         *frontMatter
	Postname   string
	DriveFiles []*drive.File
}

func (i *InputForm) CurrentPath() string {
	switch i.Action {
	case "/replace":
		return "/edit"
	case "/upload":
		return "/new"
	default:
		panic("InputForm: unknown Action to path mapping")
	}
}

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
	//google api config for drive integration
	GAPI *GAPIConfig
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
	stopped  chan struct{}
	hugo     *HugoRepo
	config   *ServerConfig
	PostPush postpushfunc
	drive    *GDriveClient
}

func NewServer(config *ServerConfig) *server {
	return &server{config: config, stopped: make(chan struct{}), PostPush: defaultpostpush}
}

func (s *server) Start(ctx context.Context) (chan error, error) {
	if s.config == nil {
		log.Fatal("server config is nil")
	}

	var err error
	//create google drive api client
	if len(s.config.GAPI.PrivateKeyID) > 0 {
		s.drive, err = NewGDriveCli(ctx, s.config.GAPI)
		if err != nil {
			return nil, errors.New("error creating google drive api client: " + err.Error())
		}
	}

	//start servers
	//initialize repo

	//check if repo exists first, if not: clone it
	ok, err := func() (bool, error) {
		_, err := os.Stat(s.config.Path)
		if err == nil {
			return true, nil
		}
		if os.IsNotExist(err) {
			return false, nil
		}
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
			log.Printf("error writing error message to response body: %s\n", err)
		}
		return true
	}
	return false
}

func PostnameFromURL(url string) string {
	//drop off querystring
	url = strings.Split(url, "?")[0]
	//drop trailing forwardslash
	if url[len(url)-1] == '/' {
		url = url[:len(url)-1]
	}
	urlparts := strings.Split(url, "/")
	//drop off querystring
	return urlparts[len(urlparts)-1]
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
		var file io.ReadCloser
		if id := req.FormValue("drivefile"); len(id) > 0 {
			file, err = s.drive.GetFile(id)
			if !success("get drivefile", err) {
				return
			}
		} else {
			//get file from form
			file, _, err = req.FormFile("userfile")
			if !success("get userfile", err) {
				return
			}
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
		//set commit message
		if s.hugo.onDeck == nil {
			success("set commit messge", errors.New("hugo onDeck is nil"))
		}
		s.hugo.onDeck.msg = "published " + s.hugo.onDeck.name
		//get user provided commit msg
		if umsg := req.URL.Query().Get("msg"); len(umsg) > 0 {
			s.hugo.onDeck.msg = umsg
		}
		//wait for hugo to rebuild
		//TODO: add a channel for this?
		time.Sleep(time.Second * 4)
		//execute publish template
		http.Redirect(w, req, fmt.Sprintf("/post/%s/?redirected=1", s.hugo.onDeck.name), int(http.StatusTemporaryRedirect))
	})

	http.HandleFunc("/replace", func(w http.ResponseWriter, req *http.Request) {
		success := func(prefix string, err error) bool {
			return !serverError(fmt.Sprintf("error handling upload: %s: %%s", prefix), w, err)
		}

		err := req.ParseMultipartForm(32 << 20) // limit your max input length!
		if !success("parse form", err) {
			return
		}
		//get file from form
		var file io.ReadCloser
		if id := req.FormValue("drivefile"); len(id) > 0 {
			file, err = s.drive.GetFile(id)
			if !success("get drivefile", err) {
				return
			}
		} else {
			//get file from form
			file, _, err = req.FormFile("userfile")
			if !success("get userfile", err) {
				return
			}
		}
		defer file.Close()

		//get front matter from form
		title := strings.TrimSpace(req.FormValue("title"))
		tags := strings.ToLower(strings.TrimSpace(req.FormValue("tags")))
		summary := strings.TrimSpace(req.FormValue("summary"))
		postname := strings.TrimSpace(req.FormValue("postname"))

		//create post in repo
		if !success("hugo new", s.hugo.Update(file, postname, title, tags, summary, s.config.Author)) {
			return
		}
		//set commit message
		if s.hugo.onDeck == nil {
			success("set commit messge", errors.New("hugo onDeck is nil"))
		}
		s.hugo.onDeck.msg = "updated " + s.hugo.onDeck.name
		//get user provided commit msg
		if umsg := req.URL.Query().Get("msg"); len(umsg) > 0 {
			s.hugo.onDeck.msg = umsg
		}
		//wait for hugo to rebuild
		//TODO: add a channel for this?
		time.Sleep(time.Second * 4)
		//execute publish template
		http.Redirect(w, req, fmt.Sprintf("/post/%s/?redirected=1", s.hugo.onDeck.name), int(http.StatusTemporaryRedirect))
	})

	http.HandleFunc("/publish", func(w http.ResponseWriter, req *http.Request) {
		success := func(err error) bool {
			return !serverError("error handling publish: %s", w, err)
		}

		post := s.hugo.onDeck.name
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

	http.HandleFunc("/new", func(w http.ResponseWriter, req *http.Request) {
		upload := req.URL.Query().Get("upload")
		var err error
		var files []*drive.File
		if s.drive != nil && len(upload) == 0 {
			files, err = s.drive.ListFiles()
			if err != nil {
				serverError("error getting drive files: %s", w, err)
				return
			}
		}
		serverError("error executing template", w, input.Execute(w, &InputForm{
			Action:     "/upload",
			Fm:         new(frontMatter),
			DriveFiles: files,
		}))
	})

	http.HandleFunc("/edit", func(w http.ResponseWriter, req *http.Request) {
		postname := req.URL.Query().Get("post")
		if len(postname) == 0 {
			serverError("%s", w, errors.New("post parameter not set"))
			return
		}
		post, err := s.hugo.GetPost(postname)
		if err != nil {
			serverError("error getting existing post: %s", w, err)
			return
		}
		upload := req.URL.Query().Get("upload")
		var files []*drive.File
		if s.drive != nil && len(upload) == 0 {
			files, err = s.drive.ListFiles()
			if err != nil {
				serverError("error getting drive files: %s", w, err)
				return
			}
		}
		serverError("error executing template", w, input.Execute(w, &InputForm{
			Action:     "/replace",
			Fm:         post.frontMatter,
			Postname:   postname,
			DriveFiles: files,
		}))
	})

	proxy := httputil.NewSingleHostReverseProxy(&url.URL{
		Scheme: "http",
		Host:   "localhost:1313",
	})
	proxy.ModifyResponse = func(response *http.Response) error {
		response.Header.Set("Access-Control-Allow-Origin", "*")
		if response.StatusCode != http.StatusOK {
			return nil
		}
		//get original request url
		url := response.Request.URL
		log.Println("requested url: ", url.String())

		//if this is not an asset request
		if !assetextregexp.MatchString(url.String()) {
			doc, err := goquery.NewDocumentFromReader(response.Body)
			if err != nil {
				return err
			}
			doc.Find("#navSubscribeBtn").AppendHtml(`
            <a href="/new" title="New Post">
                <i class="fa fa-file fa-fw" aria-hidden="true"></i>
            </a>`)

			//if this is a request for a specific post
			if posturlregxp.MatchString(url.String()) {
				postname := PostnameFromURL(url.String())
				editLink := "/edit?post=" + postname
				if s.hugo.onDeck != nil {
					if postname == s.hugo.onDeck.name {
						//change edit link to back button (retains selected document)
						//if redirected directly from new or edit page
						if len(response.Request.URL.Query().Get("redirected")) > 0 {
							editLink = "javascript:history.back()"
						}
						//inject abort / publish buttons
						doc.Find("h1").AppendHtml(`
            <a href="/publish" title="Publish Post">
                <i class="fa fa-paper-plane fa-fw" aria-hidden="true"></i>
            </a>
            <a href="/abort" title="Abort Publish">
                <i class="fa fa-ban fa-fw" aria-hidden="true"></i>
            </a>`)
					}
				}
				doc.Find("h1").AppendHtml(fmt.Sprintf(`
            <a href="%s" title="Edit Post">
                <i class="fa fa-edit fa-fw"></i>
			</a>`, editLink))
			}
			html, err := doc.Html()
			if err != nil {
				return err
			}
			response.Body = ioutil.NopCloser(strings.NewReader(html))
			response.Header["Content-Length"] = []string{fmt.Sprint(len(html))}
		}
		return nil
	}
	http.Handle("/", proxy)

	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", s.config.Port), nil))
}
