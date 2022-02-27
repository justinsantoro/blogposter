package main

type RepoHandler interface {
	Clean(recursive bool) error
	Commit(path, msg string) error
	Push() error
	Pull() error
}