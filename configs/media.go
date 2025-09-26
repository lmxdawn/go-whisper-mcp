package configs

import (
	"os"
	"path/filepath"
)

const (
	MediaDir = "whisper_media"
)

func GetMediaPath() string {
	return filepath.Join(os.TempDir(), MediaDir)
}
