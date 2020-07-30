package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type Image struct {
	Filename string
	Caption  string
}

func (i *Image) Download() error {
	url := GetImageUrl(i.Filename)
	dPath := filepath.Join("images", i.Filename)
	out, err := os.Create(dPath)
	if err != nil {
		return err
	}
	defer out.Close()

	resp, err := http.Get(url)
	defer resp.Body.Close()
	_, err = io.Copy(out, resp.Body)

	ext := filepath.Ext(i.Filename)
	if ext == ".svg" {
		pngFilename := strings.Replace(i.Filename, ".svg", ".png", 1)
		cmd := exec.Command("svgexport", dPath, filepath.Join("images", pngFilename), "1x")
		out, _ := cmd.CombinedOutput()
		plog.Infof("svgexport output: %s", string(out))
		i.Filename = pngFilename
	}
	return err
}

func (s *Distribution) Section() string {
	subs := strings.Repeat("sub", s.Sec)
	return fmt.Sprintf("\\%ssection{%s}", subs, s.Name)
}

type Distribution struct {
	Name       string
	Parameters string
	Support    string
	PDF        string
	CDF        string
	Mean       string
	Variance   string
	Notation   string
	Sec        int
	Image      *Image
}
