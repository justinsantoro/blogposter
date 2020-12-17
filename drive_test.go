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
var configFile = os.Getenv("GDRIVE_CONFIG")

func TestCreateDrive(t *testing.T) {
	if len(configFile) == 0 {
		t.Fatal("GDRIVE_CONFIG not set")
	}
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
	_, err := gdrive.ListFiles()
	if err != nil {
		t.Fatal("error listing files: ", err)
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