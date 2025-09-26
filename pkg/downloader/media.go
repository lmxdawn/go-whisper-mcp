package downloader

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/h2non/filetype"
)

// MediaDownloader 媒体下载器
type MediaDownloader struct {
	savePath   string
	httpClient *http.Client
}

// NewMediaDownloader 创建媒体下载器
func NewMediaDownloader(savePath string) *MediaDownloader {
	// 确保保存目录存在
	if err := os.MkdirAll(savePath, 0755); err != nil {
		panic(fmt.Sprintf("failed to create save path: %v", err))
	}

	return &MediaDownloader{
		savePath: savePath,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// DownloadMedia 下载媒体
// 返回本地文件路径
func (d *MediaDownloader) DownloadMedia(mediaURL string) (string, error) {
	// 验证URL格式
	if !d.isValidMediaURL(mediaURL) {
		return "", errors.New("invalid media URL format")
	}

	// 下载媒体数据
	resp, err := d.httpClient.Get(mediaURL)
	if err != nil {
		return "", errors.Join(err, errors.New("failed to download media"))
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download failed with status: %d", resp.StatusCode)
	}

	// 读取媒体数据
	mediaData, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", errors.Join(err, errors.New("failed to read media data"))
	}

	// 检测媒体格式
	kind, err := filetype.Match(mediaData)
	if err != nil {
		return "", errors.Join(err, errors.New("failed to detect file type"))
	}

	if !filetype.IsImage(mediaData) && !filetype.IsVideo(mediaData) {
		return "", errors.New("downloaded file is not a valid media")
	}

	// 生成唯一文件名
	fileName := d.generateFileName(mediaURL, kind.Extension)
	filePath := filepath.Join(d.savePath, fileName)

	// 如果文件已存在，直接返回路径
	if _, err := os.Stat(filePath); err == nil {
		return filePath, nil
	}

	// 保存到文件
	if err := os.WriteFile(filePath, mediaData, 0644); err != nil {
		return "", errors.Join(err, errors.New("failed to save media"))
	}

	return filePath, nil
}

// DownloadMedias 批量下载媒体
func (d *MediaDownloader) DownloadMedias(mediaURLs []string) ([]string, error) {
	var localPaths []string
	var errs []error

	for _, mediaURL := range mediaURLs {
		localPath, err := d.DownloadMedia(mediaURL)
		if err != nil {
			errs = append(errs, fmt.Errorf("failed to download %s: %w", mediaURL, err))
			continue
		}
		localPaths = append(localPaths, localPath)
	}

	if len(errs) > 0 {
		return localPaths, fmt.Errorf("download errors occurred: %v", errs)
	}

	return localPaths, nil
}

// isValidMediaURL 检查是否为有效的媒体URL
func (d *MediaDownloader) isValidMediaURL(rawURL string) bool {
	// 检查是否以http/https开头
	if !strings.HasPrefix(strings.ToLower(rawURL), "http://") &&
		!strings.HasPrefix(strings.ToLower(rawURL), "https://") {
		return false
	}

	// 检查URL格式
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return false
	}

	return parsedURL.Scheme != "" && parsedURL.Host != ""
}

// generateFileName 生成唯一的文件名
func (d *MediaDownloader) generateFileName(mediaURL, extension string) string {
	// 使用URL的SHA256哈希作为文件名，确保唯一性
	hash := sha256.Sum256([]byte(mediaURL))
	hashStr := fmt.Sprintf("%x", hash)

	// 取前16位哈希值作为文件名
	shortHash := hashStr[:16]

	// 添加时间戳确保更好的唯一性
	timestamp := time.Now().Unix()

	return fmt.Sprintf("img_%s_%d.%s", shortHash, timestamp, extension)
}

// IsMediaURL 判断字符串是否为媒体URL
func IsMediaURL(path string) bool {
	return strings.HasPrefix(strings.ToLower(path), "http://") ||
		strings.HasPrefix(strings.ToLower(path), "https://")
}
