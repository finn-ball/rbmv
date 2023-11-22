package main

import (
	"context"
	"log"

	"github.com/dropbox/dropbox-sdk-go-unofficial/v6/dropbox"
	"github.com/dropbox/dropbox-sdk-go-unofficial/v6/dropbox/files"
	"golang.org/x/oauth2"
)

var CLIENT_ID = ""
var CLIENT_SECRET = ""
var REFRESH_TOKEN = ""

var srcPath string = ""
var dstPath string = ""
var binFolder string = ""

type mover struct {
	client                          files.Client
	srcFolder, dstFolder, binFolder string
}

func newMover(token, srcFolder, dstFolder string) *mover {
	config := dropbox.Config{
		Token: token,
	}
	client := files.New(config)
	return &mover{
		client:    client,
		srcFolder: srcFolder,
		dstFolder: dstFolder,
		binFolder: binFolder,
	}
}

func main() {

	token := getToken(CLIENT_ID, CLIENT_SECRET, REFRESH_TOKEN)
	mv := newMover(token, srcPath, dstPath)

	files, err := mv.listAllFilesInFolder(srcPath)
	if err != nil {
		log.Fatalln("Could not list files: ", err)
	}
	if len(files) < 1 {
		log.Println("No files to copy")
	}
	width := 50
	for _, f := range files {
		copyPath := mv.createDstFilePath(f.Name)
		log.Printf("Copying %-*s %s", width, f.PathDisplay, copyPath)
		if err := mv.copyFile(f.PathLower, copyPath); err != nil {
			log.Fatalf("Could not copy file (%s) to (%s)\n Error: %s", f.PathDisplay, copyPath, err)
		}
		mvPath := mv.createBinFilePath(f.Name)

		if err := mv.moveFile(f.PathLower, mvPath); err != nil {
			log.Fatalf("Could not move file (%s) to (%s)\n Error: %s", f.PathDisplay, mvPath, err)
		}
	}
}

func getToken(clientID, clientSecret, refreshToken string) string {
	oauthConf := &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Endpoint: oauth2.Endpoint{
			TokenURL: "https://api.dropbox.com/oauth2/token",
		},
	}

	tokenSource := oauthConf.TokenSource(context.Background(), &oauth2.Token{
		RefreshToken: refreshToken,
	})

	token, err := tokenSource.Token()
	if err != nil {
		log.Fatalf("Failed to refresh token: %v", err)
	}

	return token.AccessToken
}

func (mv mover) createDstFilePath(name string) string {
	return mv.dstFolder + "/" + name
}

func (mv mover) createBinFilePath(name string) string {
	return mv.binFolder + "/" + name
}

func (mv mover) moveFile(src, dst string) error {
	arg := files.NewRelocationArg(src, dst)
	_, err := mv.client.MoveV2(arg)
	return err
}

func (mv mover) copyFile(src, dst string) error {
	arg := files.NewRelocationArg(src, dst)
	_, err := mv.client.CopyV2(arg)
	return err
}

func (mv mover) listAllFilesInFolder(folder string) ([]*files.FileMetadata, error) {
	arg := files.NewListFolderArg(folder)
	res, err := mv.client.ListFolder(arg)
	if err != nil {
		return nil, err
	}

	fileList := collectPaths(res)

	if res.HasMore {
		arg := files.NewListFolderContinueArg(res.Cursor)
		res, err := mv.client.ListFolderContinue(arg)
		if err != nil {
			return nil, err
		}
		fileList = append(fileList, collectPaths(res)...)
	}

	return fileList, nil
}

func collectPaths(res *files.ListFolderResult) []*files.FileMetadata {
	var fileList []*files.FileMetadata
	for _, entry := range res.Entries {
		if file, ok := entry.(*files.FileMetadata); ok {
			fileList = append(fileList, file)
		}
	}
	return fileList
}
