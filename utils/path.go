package utils

import (
	"os"
	"path/filepath"
)

type PathType struct {
	Current        string
	Hover          string
	Out            string
	ApplicationOut string
	AssetsOut      string
}

var wd, _ = os.Getwd()

var Path = PathType{
	Current:        wd,
	Hover:          filepath.Join(wd, ".hover"),
	Out:            filepath.Join(wd, ".hover", "out"),
	ApplicationOut: filepath.Join(wd, ".hover", "out", "application"),
	AssetsOut:      filepath.Join(wd, ".hover", "out", "assets"),
}
