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
    COPY main.go server.go hugo.go gdrive.go .
    RUN go build -o build/blogposter main.go
    SAVE ARTIFACT build/blogposter /blogposter AS LOCAL build/blogposter

docker:
    FROM klakegg/hugo:0.83.1-pandoc-ci
    COPY +build/build/blogposter /usr/bin/blogposter
    COPY entrypoint.sh /entrypoint.sh
    ENTRYPOINT ["entrypoint.sh"]
    

# add earthfile to smalltown kitten
# with container build
# container build builds scss and puts it
# in static folder
# add container environment to config with sets flag to use
# static css in container header.
# all files minus content and data folder are copied to build as artifacts
# blogposter container then mounts /blogrepo/content and /blogrepo/data as volumes
# git sparse checkout is used to clone just the content and data folders in entrypoint on starup
