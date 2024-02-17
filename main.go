package main

import (
	"context"

	"github.com/charmbracelet/log"

	"github.com/BurntSushi/toml"
	"github.com/dropbox/dropbox-sdk-go-unofficial/v6/dropbox"
	"github.com/dropbox/dropbox-sdk-go-unofficial/v6/dropbox/files"
	"golang.org/x/oauth2"
)

type Config struct {
	Paths Paths
	Auth  Auth
}

type Paths struct {
	SrcPath  string
	CopyPath string
	MovePath string
}

type Auth struct {
	ClientID     string
	ClientSecret string
	RefreshToken string
}

type mover struct {
	client files.Client
	paths  Paths
}

const formatDisplay = "%s from %-40s to %s"

func newMover(token string, paths Paths) *mover {
	config := dropbox.Config{
		Token: token,
	}
	client := files.New(config)
	return &mover{
		client: client,
		paths:  paths,
	}
}

func main() {

	log.Info("Welcome, let's move some files!")

	var config Config

	if _, err := toml.DecodeFile("settings.toml", &config); err != nil {
		log.Fatal("Could not read settings file.\nError: ", err)
	}

	token := getToken(config.Auth.ClientID, config.Auth.ClientSecret, config.Auth.RefreshToken)

	mv := newMover(token, config.Paths)
	files := mv.listFiles()
	mv.copyFiles(files)
	mv.moveFiles(files)
}

func (mv mover) copyFiles(files []*files.FileMetadata) {
	if mv.paths.CopyPath == "" {
		log.Warn("No copy path.")
	}
	for _, f := range files {
		copyPath := mv.createCopyFilePath(f.Name)

		log.Infof(formatDisplay, "copying ", f.PathDisplay, copyPath)
		if err := mv.copyFile(f.PathLower, copyPath); err != nil {
			log.Fatalf("Could not copy file (%s) to (%s)\n Error: %s", f.PathDisplay, copyPath, err)
		}
	}
}

func (mv mover) moveFiles(files []*files.FileMetadata) {
	if mv.paths.MovePath == "" {
		log.Warn("No move path.")
	}
	for _, f := range files {
		mvPath := mv.createMoveFilePath(f.Name)

		log.Infof(formatDisplay, "moving ", f.PathDisplay, mvPath)
		if err := mv.moveFile(f.PathLower, mvPath); err != nil {
			log.Fatalf("Could not move file (%s) to (%s)\n Error: %s", f.PathDisplay, mvPath, err)
		}
	}
}

func (mv mover) listFiles() []*files.FileMetadata {
	files, err := mv.listAllFilesInFolder(mv.paths.SrcPath)
	if err != nil {
		log.Fatal("Could not list files: ", err)
	}
	if len(files) < 1 {
		log.Fatal("No source files. Are the settings correct?")
	}

	return files
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
		log.Fatal("Failed to refresh token: ", err)
	}

	return token.AccessToken
}

func (mv mover) createCopyFilePath(name string) string {
	return mv.paths.CopyPath + "/" + name
}

func (mv mover) createMoveFilePath(name string) string {
	return mv.paths.MovePath + "/" + name
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
