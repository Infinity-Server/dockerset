package main

import (
  "os"
  "fmt"
  "io/fs"
  "path/filepath"
  "encoding/json"

  _ "net/http/pprof" //nolint:gosec

  "github.com/brahma-adshonor/gohook"
  "github.com/navidrome/navidrome/ui"
  "github.com/navidrome/navidrome/cmd"
  "github.com/navidrome/navidrome/log"
  "github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/conf/buildtags"
  "github.com/navidrome/navidrome/scanner/metadata"
)

func FakeLyrics(t metadata.Tags) string {
  log.Debug("Hook fired", "func", "metadata.Tags.Lyrics")

  lyrics := RealLyrics(t)
  mediaFile := t.FilePath()
  mediaExt := filepath.Ext(mediaFile)
  lyricFile := fmt.Sprintf("%s.%s", mediaFile[:len(mediaFile) - len(mediaExt)], "lrc")

  lyricData, err := os.ReadFile(lyricFile)
  if err != nil {
    return lyrics
  }

  var lyricList model.LyricList
  if uErr := json.Unmarshal([]byte(lyrics), &lyricList); uErr != nil {
    return lyrics
  }

  parsedLrc, pErr := model.ToLyrics("xxx", string(lyricData))
  if pErr != nil {
    return lyrics
  }

  lyricList = append(lyricList, *parsedLrc)
  res, e := json.Marshal(lyricList)
  if e != nil {
    return lyrics
  }

  return string(res)
}

func RealLyrics(metadata.Tags) string {
  return ""
}

func FakeNewTag(filePath string, fileInfo os.FileInfo, tags metadata.ParsedTags) metadata.Tags {
  log.Debug("Hook fired", "func", "metadata.NewTag")
  t := RealNewTag(filePath, fileInfo, tags)
  gohook.HookMethod(t, "Lyrics", FakeLyrics, RealLyrics)
  return t
}

func RealNewTag(filePath string, fileInfo os.FileInfo, tags metadata.ParsedTags) metadata.Tags {
  return metadata.Tags{}
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

//goland:noinspection GoBoolExpressions
func main() {
	_ = buildtags.NETGO

  gohook.Hook(ui.BuildAssets, FakeBuildAssets, nil)
  gohook.Hook(metadata.NewTag, FakeNewTag, RealNewTag)
  cmd.Execute()
}
