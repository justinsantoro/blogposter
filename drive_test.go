package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"testing"
)

var gdrive *GDriveClient
var configFile = &GAPIConfig{
	PrivateKeyID: os.Getenv(envKeyConfigGAPIPrivateKeyID),
	PrivateKey:   os.Getenv(envKeyConfigGAPIPrivateKey),
	Email:        os.Getenv(envKeyConfigGAPIEmail),
	TokenURL:     os.Getenv(envKeyConfigGAPITokenURL),
}
var testFileID = "1AJQX6aglI_nWCzK5NK2uqrX9bnbswXVoQEhlgQ3NUCY"

func TestCreateDrive(t *testing.T) {
	g, err := NewGDriveCli(context.Background(), configFile)
	if err != nil {
		t.Fatal("error initializing google drive client: ", err)
	}
	gdrive = g
}

func TestListDrive(t *testing.T) {
	lscall := gdrive.Drives.List()
	ls, err := lscall.Do()
	if err != nil {
		t.Fatal("error getting Drives list: ", err)
	}
	b, err := ls.MarshalJSON()
	bi := new(bytes.Buffer)
	err = json.Indent(bi, b, "", "    ")
	if err != nil {
		t.Fatal("error indenting json")
	}
	fmt.Printf("DriveList: %s \n", bi)
}

func TestListFiles(t *testing.T) {
	files, err := gdrive.ListFiles()
	if err != nil {
		t.Fatal("error listing files: ", err)
	}
	for _, f := range files {
		if f.MimeType == folderMIME {
			t.Errorf("file list contains folder: %s", f.Name)
		}
	}
}

func TestDownload(t *testing.T) {
	body, err := gdrive.GetFile(testFileID)
	if err != nil {
		t.Errorf("error downloading file: %s", err)
	}
	defer body.Close()
	content, err := getDocContent(body)
	if err != nil {
		t.Errorf("error converting document content: %s", err)
	}
	fmt.Printf("%s", content)
}
