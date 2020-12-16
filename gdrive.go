package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"

	//"golang.org/x/oauth2"
	//"golang.org/x/oauth2/google"
	"golang.org/x/oauth2/jwt"
	"io/ioutil"
)

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
	fmt.Printf("%s\n", config.PrivateKey)
	client := config.Client(ctx)
	service, err := drive.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, err
	}
	return &GDriveClient{service}, nil
}

func (g *GDriveClient) ListFiles() ([]*drive.File, error) {
	files, err := g.Service.Files.List().Do()
	if err != nil {
		return nil, err
	}
	b, err := files.MarshalJSON()
	bi := new(bytes.Buffer)
	err = json.Indent(bi, b, "", "    ")
	if err != nil {
		return nil, err
	}
	fmt.Printf("files: %s \n", bi)
	return files.Files, nil
}