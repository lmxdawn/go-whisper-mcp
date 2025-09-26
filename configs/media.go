package configs

import (
	"os"
)

func GetMediaPath() string {
	mediaDir := "./whisper_media"
	if s := os.Getenv("MEDIA_DIR"); len(s) > 0 {
		mediaDir = s
	}
	return mediaDir
}
