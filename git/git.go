package git

import (
	"context"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
)

const remoteOrigin = "origin"

type Repository struct {
	*git.Repository
}

//Open opens the git repository at path
func Open(path string) (*Repository, error) {
	r, err := git.PlainOpen(path)
	if err != nil {
		return nil, err
	}
	return &Repository{r}, err
}

//Reset git resets to the given hash
func Reset(r *Repository, hash plumbing.Hash, hard bool) error {
	wt, err := r.Worktree()
	if err != nil {
		return err
	}
	return wt.Reset(&git.ResetOptions{
		Commit: hash,
		Mode:   git.HardReset,
	})
}

func resetHead(r *Repository, hard bool) error {
	h, err := r.Head()
	if err != nil {
		return err
	}
	return Reset(r, h.Hash(), hard)
}

//ResetHead git resets to the head commit
func ResetHead(r *Repository) error {
	return resetHead(r, false)
}

//ResetHeadHard git resets to the head commit with the --hard flag
func ResetHeadHard(r *Repository) error {
	return resetHead(r, true)
}

func clean(r *Repository, dir bool) error {
	wt, err := r.Worktree()
	if err != nil {
		return err
	}

	return wt.Clean(&git.CleanOptions{Dir: dir})
}

//Clean git cleans all untracked files in the repository root
func Clean(r *Repository) error {
	return clean(r, false)
}

//CleanDir git cleans all untracked files with the -d option
func CleanDir(r *Repository) error {
	return clean(r, true)
}

func add(r *Repository, opts git.AddOptions) error {
	wt, err := r.Worktree()
	if err != nil {
		return err
	}
	return wt.AddWithOptions(&opts)
}

//AddPath git adds changes to file at path
func AddPath(r *Repository, path string) error {
	return add(r, git.AddOptions{Path: path})
}

//AddGlob git adds changes to files matching the given glob pattern
func AddGlob(r *Repository, glob string) error {
	return add(r, git.AddOptions{Glob: glob})
}

//AddAll git adds all changes
func AddAll(r *Repository) error {
	return add(r, git.AddOptions{All: true})
}

//PushContext git pushes changes to remote with a context
func PushContext(ctx context.Context, r *Repository, username, password string) error {
	err := r.PushContext(ctx, &git.PushOptions{Auth: &http.BasicAuth{Username: username, Password: password}})
	if err != nil {
		return err
	}
	return nil
}

//Push git pushes changes to remote
func Push(r *Repository, username, password string) error {
	return PushContext(context.Background(), r, username, password)
}

func pull(ctx context.Context, r *Repository, opts *git.PullOptions) error {
	wt, err := r.Worktree()
	if err != nil {
		return err
	}
	err = wt.PullContext(ctx, opts)
	if err != git.NoErrAlreadyUpToDate {
		return err
	}
	return nil
}

//PullContext git pull changes from remote with a context
func PullContext(ctx context.Context, r *Repository, remote string) error {
	return pull(ctx, r, &git.PullOptions{RemoteName: remote})
}

//Pull git pulls changes from remote
func Pull(r *Repository, remote string) error {
	return PullContext(context.Background(), r, remote)
}

func commit(r *Repository, msg string, opts *git.CommitOptions) (plumbing.Hash, error) {
	wt, err := r.Worktree()
	if err != nil {
		return plumbing.Hash{}, err
	}
	return wt.Commit(msg, opts)
}

//Commit git commits changes to the repository with the given message and
//auther name and email. Commit time is set to time.Now().
func Commit(r *Repository, msg, name, email string) error {
	_, err := commit(r, msg, &git.CommitOptions{Author: &object.Signature{Name: name, Email: email, When: time.Now()}})
	return err
}
