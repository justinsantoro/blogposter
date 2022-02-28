package main

type RepoHandler interface {
	Clean(recursive bool) error
	Commit(path, msg string) error
	Push() error
	Pull() error
	WriteFile(path string, b []byte) error
	ReadFile(path string) ([]byte, error)
}
