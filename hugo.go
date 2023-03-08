package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

var PandocLoc = "pandoc"
var r = regexp.MustCompile("[^a-zA-Z0-9\\s]+")
var lbrk = regexp.MustCompile("\\\\\n")
var listblkqtreplace = "  -"
var listblkqt = regexp.MustCompile(`\s\s-\s>`)
var numlistblkqtreplace = "1."
var numlistblkqt = regexp.MustCompile(`\d{1,3}\.\s{1,2}>`)
var listblkqtconinuedreplace = ""
var listblkqtcontinued = regexp.MustCompile(`\s\s\s\s>\s`)

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
	b := outbuf.Bytes()
	//replace gt and lt html placeholders with literals
	b = bytes.ReplaceAll(b, []byte("&gt;"), []byte{'>'})
	b = bytes.ReplaceAll(b, []byte("&lt;"), []byte{'<'})
	//undo pandoc escaping of horizontal line rules
	b = bytes.ReplaceAll(b, []byte("\\---"), []byte("---"))
	//undo pandoc escaping of markdown line breaks
	b = lbrk.ReplaceAll(b, []byte(""))
	//undo pandoc making bulleted list items blockquotes
	b = listblkqt.ReplaceAll(b, []byte(listblkqtreplace))
	//undo pandoc making number list items blockquotes
	b = numlistblkqt.ReplaceAll(b, []byte(numlistblkqtreplace))
	//replace all list block quote line continuations
	b = listblkqtcontinued.ReplaceAll(b, []byte(listblkqtconinuedreplace))
	//undo pandoc escaping of backticks
	b = bytes.ReplaceAll(b, []byte("\\`"), []byte("`"))

	return b, nil
}

type frontMatter struct {
	Title   string    `json:"title"`
	Author  string    `json:"author,omitempty"`
	Date    time.Time `json:"date"`
	Summary string    `json:"summary,omitempty"`
	Tags    []string  `json:"tags,omitemtpy"`
	Img     string    `json:"Img,omitempty"`
}

func (fm *frontMatter) Json() ([]byte, error) {
	return json.MarshalIndent(fm, "", "    ")
}

//TagList returns a space seperated list of tags
func (fm *frontMatter) TagList() string {
	s := ""
	if len(fm.Tags) == 0 {
		return s
	}
	for _, t := range fm.Tags {
		s = s + " " + t
	}
	return s[1:]
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
			Title:   title,
			Author:  author,
			Date:    time.Now(),
			Summary: summary,
			Tags:    strings.Split(tags, " "),
		},
	}, nil
}

func existingPost(b []byte) (*post, error) {
	p := &post{frontMatter: new(frontMatter)}
	//split off front matter
	parts := bytes.SplitAfter(b, []byte{'}'})
	if len(parts) > 2 {
		return nil, errors.New("can't parse post because it contains more than one '}' character")
	}
	err := json.Unmarshal(parts[0], p.frontMatter)
	if err != nil {
		return nil, err
	}
	p.content = parts[1]
	return p, nil
}

//Bytes returns the post as a single
//byte array
func (p *post) Bytes() ([]byte, error) {
	b, err := p.frontMatter.Json()
	//add newline after frontmatter
	b = append(b, byte('\n'))
	if err != nil {
		return nil, err
	}
	return append(b, p.content...), nil
}

func (p *post) Fname() string {
	if p.frontMatter == nil {
		panic("cant get post fname because frontmatter is nil")
	}
	mdname := strings.ReplaceAll(strings.ToLower(r.ReplaceAllString(p.frontMatter.Title, "")), " ", "-")
	//set file name as
	return "content/post/" + mdname + ".md"
}

type HugoRepo struct {
	baseUrl  string
	repo     RepoHandler
	onDeck   *OnDeck
	test     bool
	siteRoot string
}

type OnDeck struct {
	name string
	msg  string
}

func (h *HugoRepo) StartServer(ctx context.Context, stopped chan<- struct{}) (chan error, error) {
	cmd := exec.CommandContext(ctx, "hugo", "server", "--watch=true", "--disableLiveReload", "--bind", "0.0.0.0", "--baseURL", h.baseUrl)
	cmd.Dir = h.siteRoot
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

func NewHugoRepo(baseUrl string, rh RepoHandler) (*HugoRepo, error) {
	return &HugoRepo{
		repo:    rh,
		baseUrl: baseUrl,
	}, nil
}

func (h *HugoRepo) stageChange(post *post) error {

	//pull down any changes on remote
	if err := h.repo.Pull(); err != nil {
		return err
	}

	b, err := post.Bytes()
	if err != nil {
		return errors.New("PostBytes: " + err.Error())
	}
	err = h.repo.WriteFile(post.Fname(), b)
	if err != nil {
		return err
	}

	//set on deck name to just the name of the post file
	//no extension
	nparts := strings.Split(post.Fname(), "/")
	name := nparts[len(nparts)-1]
	name = name[:len(name)-3]
	h.onDeck = &OnDeck{name: name}

	return nil
}

func (h *HugoRepo) New(c io.Reader, title, tags, summary, author string) error {
	//create post file
	post, err := newPost(c, title, tags, summary, author)
	if err != nil {
		return errors.New("newPost: " + err.Error())
	}

	return h.stageChange(post)
}

func (h *HugoRepo) GetPost(name string) (*post, error) {
	//build file name
	b, err := h.repo.ReadFile("content/post/" + name + ".md")
	if err != nil {
		return nil, err
	}
	return existingPost(b)
}

func (h *HugoRepo) Update(c io.Reader, name, title, tags, summary, author string) error {
	post, err := h.GetPost(name)
	if err != nil {
		return err
	}
	npost, err := newPost(c, title, tags, summary, author)
	if err != nil {
		return err
	}
	//copy old post date to new
	npost.frontMatter.Date = post.frontMatter.Date

	return h.stageChange(npost)
}

func (h *HugoRepo) Deploy() error {

	//add commit
	if h.onDeck == nil {
		return errors.New("went to commit but onDeck is empty")
	}

	if err := h.repo.Commit("content/post/"+h.onDeck.name, h.onDeck.msg); err != nil {
		return errors.New("error committing to repo: " + err.Error())
	}

	//push to remote
	if !h.test {
		return h.repo.Push()
	}
	h.onDeck = nil
	return nil
}

func (h *HugoRepo) Abort() error {
	h.onDeck = nil
	return h.repo.Clean(true)
}
