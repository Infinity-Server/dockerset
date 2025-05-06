package main

import (
  "os"
  "fmt"
  "io/fs"
  "path/filepath"

  "github.com/brahma-adshonor/gohook"

	"github.com/spf13/viper"
	"github.com/navidrome/navidrome/ui"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/metadata"
)

func FakePairs(md metadata.Metadata, key model.TagName) []metadata.Pair {
  log.Debug("Hook fired", "func", "metadata.Metadata.Pairs")

	if key != model.TagLyrics {
		return RealPairs(md, key)
	}

  lyrics := RealPairs(md, key)

  mediaFile := md.FilePath()
  mediaExt := filepath.Ext(mediaFile)
  lyricFile := fmt.Sprintf("%s.%s", mediaFile[:len(mediaFile) - len(mediaExt)], "lrc")

  lyricData, err := os.ReadFile(filepath.Join(conf.Server.MusicFolder, lyricFile))
  if err != nil {
    return lyrics
  }

	pair := metadata.Pair(metadata.NewPair("xxx", string(lyricData)))
	lyrics = append(lyrics, pair)

	return lyrics
}

func RealPairs(md metadata.Metadata, key model.TagName) []metadata.Pair {
  return make([]metadata.Pair, 0)
}

func FakeNewTag(filePath string, info metadata.Info) metadata.Metadata {
  log.Debug("Hook fired", "func", "metadata.New")
  t := RealNewTag(filePath, info)
  gohook.HookMethod(t, "Pairs", FakePairs, RealPairs)
  return t
}

func RealNewTag(filePath string, info metadata.Info) metadata.Metadata {
  return metadata.Metadata{}
}

type FakeFS struct {
  Dir string
}

func (f FakeFS) Open(name string) (fs.File, error) {
  log.Debug("Hook fired", "func", "fs.FS.Open")
  dirFS := os.DirFS(f.Dir)
  return dirFS.Open(name)
}

func FakeBuildAssets() fs.FS {
  exe, _ := os.Executable()
  exeDir := filepath.Dir(exe)
  buildDir := filepath.Join(exeDir, "build")
  log.Debug("Use fake fs for frontend", "func", "ui.BuildAssets")
  return FakeFS{
    Dir: buildDir,
  }
}

func init() {
	viper.BindEnv("dbpath")

  gohook.Hook(ui.BuildAssets, FakeBuildAssets, nil)
  gohook.Hook(metadata.New, FakeNewTag, RealNewTag)
}
