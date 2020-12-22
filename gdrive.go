package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
	"io"
	"log"

	//"golang.org/x/oauth2"
	//"golang.org/x/oauth2/google"
	"golang.org/x/oauth2/jwt"
	"io/ioutil"
)

const docxMIME = "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
const folderMIME = "application/vnd.google-apps.folder"

type gmarshaler interface {
	MarshalJSON() ([]byte, error)
}

func pprint(m gmarshaler) {
	logerr := func(err error) {
		log.Printf("error pprinting: %s\n", err)
	}
	b, err := m.MarshalJSON()
	if err != nil {
		logerr(err)
	}
	bi := new(bytes.Buffer)
	err = json.Indent(bi, b, "", "    ")
	if err != nil {
		logerr(err)
	}
	fmt.Printf("%s\n", bi)
}

type GDriveClient struct {
	*drive.Service
}

func NewGDriveCli(ctx context.Context, configFile string) (*GDriveClient, error) {
	b, err := ioutil.ReadFile(configFile)
	if err != nil {
		return nil, err
	}
	config := new(jwt.Config)
	err = json.Unmarshal(b, config)
	if err != nil {
		return nil, err
	}
	config.Scopes = []string{drive.DriveScope}
	client := config.Client(ctx)
	service, err := drive.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, err
	}
	return &GDriveClient{service}, nil
}

func minusFolders(f []*drive.File) []*drive.File {
	rmitem := func(s []*drive.File, i int) []*drive.File {
		s[i] = s[len(s)-1]
		return s[:len(s)-1]
	}
	var s []*drive.File
	s = f
	for i, file := range f {
		if file.MimeType == folderMIME {
			s = minusFolders(rmitem(f, i))
			break
		}
	}
	return s
}

func (g *GDriveClient) ListFiles() ([]*drive.File, error) {
	files, err := g.Service.Files.List().Do()
	if err != nil {
		return nil, err
	}
	pprint(files)
	return minusFolders(files.Files), nil
}

func (g *GDriveClient) GetFile(id string) (io.ReadCloser, error) {
	fileresp, err := g.Service.Files.Export(id, docxMIME).Download()
	if err != nil {
		return nil, err
	}
	if fileresp.StatusCode != 200 {
		defer fileresp.Body.Close()
		b, err := ioutil.ReadAll(fileresp.Body)
		if err != nil {
			log.Printf("GetFile: error reading unsuccessful response body: - %s\n", err)
		}
		return nil, errors.New(fmt.Sprintf("download request returned non-successful error code: %d - %s", fileresp.StatusCode, b))
	}
	return fileresp.Body, nil

}