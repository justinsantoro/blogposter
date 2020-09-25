package main

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/go-git/go-git/v5/plumbing/object"
	"io"
	"io/ioutil"
	"os/exec"
	"path"
	"strings"
	"time"

	"github.com/go-git/go-git/v5"
	githttp "github.com/go-git/go-git/v5/plumbing/transport/http"
)

//GetDocContent takes reader assumed
//to read the content of a .docx document
//converts the content to CommonMark Markdown,
//and returns it as a byte slice
func getDocContent(c io.Reader) ([]byte, error) {
	cmd := exec.Command("pandoc")
	cmd.Stdin = c
	cmd.Args = []string{"-f", "docx", "-t", "commonmark"}
	err := cmd.Run()
	if err != nil {
		return nil, err
	}
	b, err := cmd.CombinedOutput()
	if err != nil {
		return nil, err
	}
	return b, nil
}

type frontMatter struct {
	Title       string    `json:"title"`
	Author      string    `json:"author,omitempty"`
	Date        time.Time `json:"date"`
	Description string    `json:"description,omitempty"`
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
func newPost(c io.Reader, title, tags string, description string, author string) (*post, error) {
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
			Description: description,
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
	path string
	repo *git.Repository
	auth *githttp.BasicAuth
	cancel context.CancelFunc
	c <-chan error
	onDeck string
}

func (h *HugoRepo) StartServer(ctx context.Context) (chan error, error) {
	ctx, h.cancel = context.WithCancel(ctx)
	cmd := exec.CommandContext(ctx, "hugo", "server", "--watch=true")
	cmd.Dir = h.path
	err := cmd.Start()
	if err != nil {
		return nil, err
	}
	c := make(chan error, 0)
	go func(){
		c <- cmd.Wait();
	}()
	return c, nil
}

func (h *HugoRepo) StopServer() error {
	if h.cancel == nil {
		return nil
	}
	h.cancel()
	err := <-h.c

	//cleanup
	h.cancel = nil
	h.c = nil

	return err
}

func NewHugoRepo(path string, username, token string) (*HugoRepo, error) {
	repo, err := git.PlainOpen(path)
	if err != nil {
		return nil, err
	}
	return &HugoRepo{
		path:path,
		repo:repo,
		auth:&githttp.BasicAuth{
			Username:username,
			Password:token,
		},
	}, nil
}

func (h *HugoRepo) New(c io.Reader, title, tags string, description string, author string) error {
	//reset in case there are any lingering changes
	err := h.Abort()
	if err != nil {
		return err
	}

	//get work tree
	wt, err := h.repo.Worktree()
	if err != nil {
		return err
	}

	//create post file
	post, err := newPost(c, title, tags, description, author)
	if err != nil {
		return err
	}
	mdname := strings.ReplaceAll(strings.ToLower(post.frontMatter.Title), " ", "-")
	//set file name as
	fname := "content/post/" + mdname + ".md"
	b, err := post.Bytes()
	if err != nil {
		return err
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

	h.onDeck = mdname;

	return nil
}

func (h *HugoRepo) Deploy() error{
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

	//add commit (what should message be?)
	if len(h.onDeck) == 0 {
		return errors.New("went to commit but onDeck is empty")
	}
	_ , err = wt.Commit("publish " + h.onDeck, &git.CommitOptions{
		Author:    &object.Signature{
			Name:  "kitten",
			Email: "kitten@smalltownkitten.com",
		},
	})

	//push to remote
	return h.repo.PushContext(context.TODO(), &git.PushOptions{
		Auth:       h.auth,
	})
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
		Mode: git.HardReset,
	})
}