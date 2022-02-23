VERSION 0.6
FROM golang:1.17.7-alpine3.14
WORKDIR /blogposter

deps:
    COPY go.mod go.sum ./
	RUN go mod download
    # Output these back in case go mod download changes them.
	SAVE ARTIFACT go.mod AS LOCAL go.mod
	SAVE ARTIFACT go.sum AS LOCAL go.sum

build:
    FROM +deps
    COPY main.go server.go hugo.go gdrive.go ./
    RUN go build -o build/blogposter .
    SAVE ARTIFACT build/blogposter blogposter AS LOCAL build/blogposter

docker:
    IMPORT github.com/sarahlehman/smalltownkitten:add-earthfile
    FROM klakegg/hugo:0.83.1-pandoc-ci
    COPY +build/blogposter /usr/bin/blogposter
    COPY smalltownkitten+build/site ./site
    COPY entrypoint.sh /entrypoint.sh
    ENTRYPOINT ["entrypoint.sh"]
    SAVE IMAGE stkposter:latest

