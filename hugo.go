package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/go-git/go-git/v5/plumbing/object"
	"io"
	"io/ioutil"
	"log"
	"os/exec"
	"path"
	"regexp"
	"strings"
	"time"

	"github.com/go-git/go-git/v5"
	githttp "github.com/go-git/go-git/v5/plumbing/transport/http"
)

var PandocLoc = "pandoc"
var r = regexp.MustCompile("[^a-zA-Z0-9\\s]+")

//GetDocContent takes path to file assumed to be
//a docx file to be converted to commonmark via
//pandoc
func getDocContent(c io.Reader) ([]byte, error) {
	outbuf := new(bytes.Buffer)
	cmd := exec.Command(PandocLoc, "-f", "docx", "-t", "commonmark", "-o", "-")
	log.Println(cmd.String())
	cmd.Stdin = c
	cmd.Stdout = outbuf
	err := cmd.Run()
	if err != nil {
		switch e := err.(type) {
		case *exec.ExitError:
			//pandoc outputs error messages to stdout - not stderr
			err = fmt.Errorf("%s: %s", e.Error(), outbuf.String())
		}
		return nil, err
	}
	return outbuf.Bytes(), nil
}

type frontMatter struct {
	Title       string    `json:"title"`
	Author      string    `json:"author,omitempty"`
	Date        time.Time `json:"date"`
	Summary string    `json:"summary,omitempty"`
	Tags        []string  `json:"tags"`
}

func (fm *frontMatter) Json() ([]byte, error) {
	return json.MarshalIndent(fm, "", "    ")
}

//Post is a blog post
type post struct {
	content     []byte
	frontMatter *frontMatter
}

//newPost returns a post
func newPost(c io.Reader, title, tags string, summary string, author string) (*post, error) {
	doc, err := getDocContent(c)
	if err != nil {
		return nil, err
	}
	return &post{
		content: doc,
		frontMatter: &frontMatter{
			Title:       title,
			Author:      author,
			Date:        time.Now(),
			Summary: summary,
			Tags:        strings.Split(tags, " "),
		},
	}, nil
}

//Bytes returns the post as a single
//byte array
func (p *post) Bytes() ([]byte, error) {
	b, err := p.frontMatter.Json()
	if err != nil {
		return nil, err
	}
	return append(b, p.content...), nil
}

type HugoRepo struct {
	path   string
	repo   *git.Repository
	auth   *githttp.BasicAuth
	name   string
	email  string
	onDeck string
	test bool
}

func (h *HugoRepo) StartServer(ctx context.Context, stopped chan<- struct{}) (chan error, error) {
	cmd := exec.CommandContext(ctx, "hugo", "server", "--watch=true")
	cmd.Dir = h.path
	err := cmd.Start()
	if err != nil {
		return nil, err
	}
	c := make(chan error, 1)
	go func() {
		c <- cmd.Wait()
		log.Println("hugo server stopped")
		close(stopped)
	}()
	return c, nil
}

func NewHugoRepo(path string, username, token string) (*HugoRepo, error) {
	repo, err := git.PlainOpen(path)
	if err != nil {
		return nil, err
	}
	return &HugoRepo{
		path: path,
		repo: repo,
		auth: &githttp.BasicAuth{
			Username: username,
			Password: token,
		},
	}, nil
}

func (h *HugoRepo) New(c io.Reader, title, tags, summary, author string) error {
	//reset in case there are any lingering changes
	err := h.Abort()
	if err != nil {
		return errors.New("abort: " + err.Error())
	}

	//get work tree
	wt, err := h.repo.Worktree()
	if err != nil {
		return errors.New("worktree: " + err.Error())
	}

	//create post file
	post, err := newPost(c, title, tags, summary, author)
	if err != nil {
		return errors.New("newPost: " + err.Error())
	}
	mdname := strings.ReplaceAll(strings.ToLower(r.ReplaceAllString(post.frontMatter.Title, "")), " ", "-")
	//set file name as
	fname := "content/post/" + mdname + ".md"
	b, err := post.Bytes()
	if err != nil {
		return errors.New("PostBytes: " + err.Error())
	}
	err = ioutil.WriteFile(path.Join(h.path, fname), b, 0644)
	if err != nil {
		return err
	}

	//stage changes
	_, err = wt.Add(fname)
	if err != nil {
		return err
	}

	h.onDeck = mdname

	return nil
}

func (h *HugoRepo) Deploy() error {
	wt, err := h.repo.Worktree()
	if err != nil {
		return err
	}
	//if index is clean return error
	st, err := wt.Status()
	if err != nil {
		return err
	}
	if st.IsClean() {
		h.onDeck = ""
		return errors.New("index is clean")
	}

	//add commit
	if len(h.onDeck) == 0 {
		return errors.New("went to commit but onDeck is empty")
	}
	_, err = wt.Commit("publish "+h.onDeck, &git.CommitOptions{
		Author: &object.Signature{
			Name:  h.name,
			Email: h.email,
		},
	})

	//push to remote
	if !h.test {
		return h.repo.PushContext(context.TODO(), &git.PushOptions{
			Auth: h.auth,
		})
	}
	return nil
}

func (h *HugoRepo) Abort() error {
	//unstage changes and clean directory
	head, err := h.repo.Head()
	if err != nil {
		return err
	}
	wt, err := h.repo.Worktree()
	if err != nil {
		return err
	}
	return wt.Reset(&git.ResetOptions{
		Commit: head.Hash(),
		Mode:   git.HardReset,
	})
}