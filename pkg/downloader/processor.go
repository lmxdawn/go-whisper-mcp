package downloader

import (
	"fmt"

	"go-whisper-mcp/configs"
)

// MediaProcessor 媒体处理器
type MediaProcessor struct {
	downloader *MediaDownloader
}

// NewMediaProcessor 创建媒体处理器
func NewMediaProcessor() *MediaProcessor {
	return &MediaProcessor{
		downloader: NewMediaDownloader(configs.GetMediaPath()),
	}
}

// ProcessMedias 处理媒体列表，返回本地文件路径
// 支持两种输入格式：
// 1. URL格式 (http/https开头) - 自动下载到本地
// 2. 本地文件路径 - 直接使用
func (p *MediaProcessor) ProcessMedias(medias []string) ([]string, error) {
	var localPaths []string
	var urlsToDownload []string

	// 分离URL和本地路径
	for _, image := range medias {
		if IsMediaURL(image) {
			urlsToDownload = append(urlsToDownload, image)
		} else {
			// 本地路径直接添加
			localPaths = append(localPaths, image)
		}
	}

	// 批量下载URL媒体
	if len(urlsToDownload) > 0 {
		downloadedPaths, err := p.downloader.DownloadMedias(urlsToDownload)
		if err != nil {
			return nil, fmt.Errorf("failed to download medias: %w", err)
		}
		localPaths = append(localPaths, downloadedPaths...)
	}

	if len(localPaths) == 0 {
		return nil, fmt.Errorf("no valid medias found")
	}

	return localPaths, nil
}
