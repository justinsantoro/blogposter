package git

import "fmt"

//RepoHandler implements blogposter.RepoHandler interface
type RepoHandler struct {
	r    *Repository
	opts RepoHandlerOpts
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
	return &RepoHandler{r: r, opts: opts}, nil
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
	return Commit(rh.r, msg, rh.opts.AuthorName, rh.opts.AuthorEmail)
}
