package git

import (
	"fmt"
	"io/fs"
	"io/ioutil"
)

//RepoHandler implements blogposter.RepoHandler interface
type RepoHandler struct {
	r    *Repository
	opts RepoHandlerOpts
	path string
}

type RepoHandlerOpts struct {
	GitUser     string
	GitPassword string
	AuthorName  string
	AuthorEmail string
}

//NewRepoHandler returns a new RepoHandler
func NewRepoHandler(path string, opts RepoHandlerOpts) (*RepoHandler, error) {
	r, err := Open(path)
	if err != nil {
		return nil, err
	}
	return &RepoHandler{r: r, opts: opts, path: path}, nil
}

//Push git pushes changes to remote
func (rh *RepoHandler) Push() error {
	return Push(rh.r, rh.opts.GitUser, rh.opts.GitPassword)
}

//Pull git pulls changes from origin
func (rh *RepoHandler) Pull() error {
	return Pull(rh.r, "origin")
}

//Clean removes all untracked files from root. Set recursive
//to true to clean all untracked files rescursively
func (rh *RepoHandler) Clean(recursive bool) error {
	if recursive {
		return CleanDir(rh.r)
	}
	return Clean(rh.r)
}

//Commit git adds then git commits the changed file(s) at path
func (rh *RepoHandler) Commit(path string, msg string) error {
	err := AddPath(rh.r, path)
	if err != nil {
		return fmt.Errorf("error adding %s to git: %v", path, err)
	}
	if err = Commit(rh.r, msg, rh.opts.AuthorName, rh.opts.AuthorEmail); err != nil {
		//unstage changes
		if err2 := ResetHead(rh.r); err2 != nil {
			return fmt.Errorf("error unstaging changes while recovering from commit error: %v:\n\t%v", err, err2)
		}
		return err
	}
	return nil
}

func (rh *RepoHandler) WriteFile(filename string, data []byte) error {
	return ioutil.WriteFile(rh.path + "/" + filename, data, fs.ModePerm)
}

func (rh *RepoHandler) ReadFile(filename string) ([]byte, error) {
	return ioutil.ReadFile(filename)
}
