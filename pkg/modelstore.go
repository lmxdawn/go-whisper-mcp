package pkg

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// EnsureModelInDir: 静默下载（老接口，兼容）
func EnsureModelInDir(ctx context.Context, modelsDir, spec string) (localPath string, downloaded bool, err error) {
	return EnsureModelInDirWithProgress(ctx, modelsDir, spec, nil)
}

// Progress 控制台进度条选项（传 nil 表示不显示）
type Progress struct {
	Enabled        bool          // 开/关
	Out            io.Writer     // 默认 os.Stderr
	BarWidth       int           // 进度条宽度，默认 40
	UpdateInterval time.Duration // 刷新间隔，默认 200ms
}

// EnsureModelInDirWithProgress: 带进度条下载
func EnsureModelInDirWithProgress(ctx context.Context, modelsDir, spec string, prog *Progress) (localPath string, downloaded bool, err error) {
	if modelsDir == "" {
		modelsDir = "./models"
	}
	filename := normalizeSpecToFilename(spec)
	filename = filepath.Base(filename)
	localPath = filepath.Join(modelsDir, filename)

	// 已存在
	if fi, e := os.Stat(localPath); e == nil && fi.Size() > 0 {
		return localPath, false, nil
	}
	if err = os.MkdirAll(modelsDir, 0o755); err != nil {
		return "", false, fmt.Errorf("mkdir %s: %w", modelsDir, err)
	}

	urls := candidateURLs(filename)
	tmp := localPath + ".part"

	// 默认超时
	if ctx == nil {
		tctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
		defer cancel()
		ctx = tctx
	}

	var lastErr error
	for _, u := range urls {
		if e := downloadTo(ctx, u, tmp, prog); e != nil {
			lastErr = e
			continue
		}
		if e := os.Rename(tmp, localPath); e != nil {
			return "", false, fmt.Errorf("rename: %w", e)
		}
		return localPath, true, nil
	}
	if lastErr == nil {
		lastErr = errors.New("no candidate url worked")
	}
	return "", false, fmt.Errorf("download %s failed: %w", filename, lastErr)
}

// ---------- 内部逻辑 ----------

func normalizeSpecToFilename(spec string) string {
	s := strings.TrimSpace(spec)
	low := strings.ToLower(s)
	if strings.HasSuffix(low, ".bin") || strings.HasSuffix(low, ".gguf") {
		return s
	}
	aliases := map[string]string{
		"tiny":           "ggml-tiny.bin",
		"tiny.en":        "ggml-tiny.en.bin",
		"base":           "ggml-base.bin",
		"base.en":        "ggml-base.en.bin",
		"small":          "ggml-small.bin",
		"small.en":       "ggml-small.en.bin",
		"medium":         "ggml-medium.bin",
		"medium.en":      "ggml-medium.en.bin",
		"large":          "ggml-large-v2.bin",
		"large-v2":       "ggml-large-v2.bin",
		"large-v3":       "ggml-large-v3.bin",
		"large-v3-turbo": "ggml-large-v3-turbo.bin",
	}
	if v, ok := aliases[low]; ok {
		return v
	}
	return s + ".bin"
}

func candidateURLs(filename string) []string {
	base := "https://huggingface.co/ggerganov/whisper.cpp/resolve/main/"
	return []string{
		base + filename,
		// 按需添加镜像
	}
}

func downloadTo(ctx context.Context, url, dst string, prog *Progress) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "whisper-go-modelstore/1.1")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("http get %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		io.Copy(io.Discard, resp.Body)
		return fmt.Errorf("bad status %s for %s", resp.Status, url)
	}

	f, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer func() {
		f.Close()
		if err != nil {
			_ = os.Remove(dst)
		}
	}()

	var reader io.Reader = resp.Body
	var pw *progressWriter
	if prog != nil && prog.Enabled {
		pw = newProgressWriter(prog, resp.ContentLength, url)
		reader = io.TeeReader(resp.Body, pw)
	}

	_, err = io.Copy(f, reader)
	if pw != nil {
		pw.finish(err == nil)
	}
	if err != nil {
		return fmt.Errorf("write: %w", err)
	}
	return nil
}

// ---------- 进度条 ----------

type progressWriter struct {
	out      io.Writer
	total    int64 // -1 表示未知
	written  int64
	start    time.Time
	last     time.Time
	interval time.Duration
	width    int
	spi      int
	urlName  string
}

func newProgressWriter(p *Progress, total int64, url string) *progressWriter {
	out := p.Out
	if out == nil {
		out = os.Stderr
	}
	width := p.BarWidth
	if width <= 0 {
		width = 40
	}
	iv := p.UpdateInterval
	if iv <= 0 {
		iv = 200 * time.Millisecond
	}
	return &progressWriter{
		out:      out,
		total:    total,
		start:    time.Now(),
		last:     time.Time{},
		interval: iv,
		width:    width,
		urlName:  shortName(url),
	}
}

func (w *progressWriter) Write(b []byte) (int, error) {
	n := len(b)
	w.written += int64(n)
	now := time.Now()
	if w.last.IsZero() || now.Sub(w.last) >= w.interval {
		w.print(false)
		w.last = now
	}
	return n, nil
}

func (w *progressWriter) finish(ok bool) {
	w.print(true)
}

func (w *progressWriter) print(final bool) {
	spinners := []rune{'|', '/', '-', '\\'}
	if final {
		fmt.Fprint(w.out, "\r")
	} else {
		fmt.Fprintf(w.out, "\r")
	}
	if w.total > 0 {
		ratio := float64(w.written) / float64(w.total)
		if ratio > 1 {
			ratio = 1
		}
		done := int(ratio * float64(w.width))
		if done > w.width {
			done = w.width
		}
		bar := strings.Repeat("█", done) + strings.Repeat("░", w.width-done)
		elapsed := time.Since(w.start)
		speed := humanBytes(float64(w.written) / elapsed.Seconds())
		eta := "-"
		if w.written > 0 {
			remain := time.Duration(float64(w.total-w.written) / float64(w.written) * float64(elapsed))
			eta = durShort(remain)
		}
		fmt.Fprintf(w.out, "[%s] %6.2f%%  %s / %s  %s/s  ETA %s  %s",
			bar, ratio*100,
			humanBytes(float64(w.written)), humanBytes(float64(w.total)),
			speed, eta, w.urlName,
		)
		if final {
			fmt.Fprintln(w.out)
		}
	} else {
		// 未知大小：显示已传/速度/旋转指示
		elapsed := time.Since(w.start)
		speed := humanBytes(float64(w.written) / elapsed.Seconds())
		ch := spinners[w.spi%len(spinners)]
		w.spi++
		fmt.Fprintf(w.out, "[%c] %s  %s/s  %s", ch, humanBytes(float64(w.written)), speed, w.urlName)
		if final {
			fmt.Fprintln(w.out)
		}
	}
}

func humanBytes(b float64) string {
	const (
		KB = 1024
		MB = 1024 * KB
		GB = 1024 * MB
	)
	switch {
	case b >= GB:
		return fmt.Sprintf("%.2f GiB", b/GB)
	case b >= MB:
		return fmt.Sprintf("%.2f MiB", b/MB)
	case b >= KB:
		return fmt.Sprintf("%.2f KiB", b/KB)
	default:
		return fmt.Sprintf("%.0f B", b)
	}
}

func durShort(d time.Duration) string {
	if d < 0 {
		d = -d
	}
	s := int(d.Seconds() + 0.5)
	h := s / 3600
	m := (s % 3600) / 60
	s2 := s % 60
	if h > 0 {
		return fmt.Sprintf("%02d:%02d:%02d", h, m, s2)
	}
	return fmt.Sprintf("%02d:%02d", m, s2)
}

func shortName(u string) string {
	if i := strings.LastIndex(u, "/"); i >= 0 && i+1 < len(u) {
		return u[i+1:]
	}
	return u
}
