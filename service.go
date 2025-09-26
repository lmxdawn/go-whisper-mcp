package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/go-audio/wav"
	pkg "go-whisper-mcp/pkg"
	"go-whisper-mcp/pkg/downloader"
	"go-whisper-mcp/whisper"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// WhisperService 小红书业务服务
type WhisperService struct{}

// TranscribeRequest 转换请求
type TranscribeRequest struct {
	InPaths   []string `json:"in_paths" binding:"required"` // 转换的路径
	Model     string   `json:"model"`                       // 模型名称
	Lang      string   `json:"lang"`
	Threads   int      `json:"t"` // 并发数量
	ModelsDir string   `json:"models_dir"`
}

// TranscribeResponse 转换返回
type TranscribeResponse struct {
	Path      string                          `json:"path"`
	IsSuccess bool                            `json:"is_success"`
	DurationS string                          `json:"duration_s"`
	Segments  []whisper.TranscribeAudioResult `json:"segments"`
}

// TranscribeBatchResponse 批量转换返回
type TranscribeBatchResponse struct {
	ModelPath string                `json:"model_path"`
	Language  string                `json:"language"`
	Threads   int                   `json:"threads"`
	DurationS string                `json:"duration_s"`
	Results   []*TranscribeResponse `json:"results"`
}

// NewWhisperService 创建whisper服务实例
func NewWhisperService() *WhisperService {
	return &WhisperService{}
}

func (s *WhisperService) Transcribe(ctx context.Context, req *TranscribeRequest) (*TranscribeBatchResponse, error) {

	inPaths := req.InPaths
	modelSpec := req.Model
	lang := req.Lang
	threads := req.Threads
	modelsDir := req.ModelsDir

	// 下载资源
	mediaProcessor := downloader.NewMediaProcessor()
	inPathFiles, err := mediaProcessor.ProcessMedias(inPaths)
	if err != nil {
		return nil, err
	}

	// 1) 模型就绪
	prog := &pkg.Progress{Enabled: true}
	modelPath, _, err := pkg.EnsureModelInDirWithProgress(ctx, modelsDir, modelSpec, prog)
	if err != nil {
		return nil, fmt.Errorf("ensure model: %w", err)
	}

	start := time.Now()
	results := make([]*TranscribeResponse, 0, len(inPathFiles))
	for _, path := range inPaths {
		batch, err := s.transcribeAudioBatch(ctx, modelPath, lang, threads, path)
		isSuccess := true
		if err != nil {
			isSuccess = false
			batch = &TranscribeResponse{}
		}
		results = append(results, &TranscribeResponse{
			Path:      path,
			IsSuccess: isSuccess,
			DurationS: batch.DurationS,
			Segments:  batch.Segments,
		})
	}

	return &TranscribeBatchResponse{
		ModelPath: modelPath,
		Language:  lang,
		Threads:   threads,
		DurationS: time.Since(start).String(),
		Results:   results,
	}, nil

}

func (s *WhisperService) transcribeAudioBatch(ctx context.Context, modelPath string, lang string, threads int, inPath string) (*TranscribeResponse, error) {

	// 2) 解码到 16k/mono/float32
	var data []float32
	var err error
	ext := strings.ToLower(filepath.Ext(inPath))
	if ext == ".wav" {
		if s, e := readWavMono16ToF32(inPath); e == nil {
			data = s
		} else {
			var e2 error
			data, e2 = pkg.DecodeF32(ctx, inPath)
			if e2 != nil {
				return nil, fmt.Errorf("decode wav: %v; ffmpeg fallback: %w", e, e2)
			}
		}
	} else {
		data, err = pkg.DecodeF32(ctx, inPath)
		if err != nil {
			return nil, fmt.Errorf("ffmpeg decode: %w", err)
		}
	}

	start := time.Now()
	transcribeAudio := whisper.NewTranscribeAudio()
	result, err := transcribeAudio.Transcribe(modelPath, lang, threads, data)
	if err != nil {
		return nil, err
	}

	return &TranscribeResponse{
		DurationS: time.Since(start).String(),
		Segments:  result,
	}, nil
}

// WAV 专用路径（必须 16k/mono/16-bit）
func readWavMono16ToF32(path string) ([]float32, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	dec := wav.NewDecoder(f)
	if !dec.IsValidFile() {
		return nil, errors.New("invalid wav")
	}

	// 这里直接拿到 *audio.IntBuffer，不要做类型断言
	ib, err := dec.FullPCMBuffer()
	if err != nil {
		return nil, err
	}
	if ib == nil || ib.Data == nil {
		return nil, errors.New("empty pcm buffer")
	}
	if ib.SourceBitDepth != 16 {
		return nil, fmt.Errorf("expect 16-bit PCM, got %d-bit", ib.SourceBitDepth)
	}
	if ib.Format == nil || ib.Format.NumChannels != 1 {
		return nil, fmt.Errorf("expect mono channel, got %d", ib.Format.NumChannels)
	}
	if ib.Format.SampleRate != 16000 {
		return nil, fmt.Errorf("expect 16kHz sample rate, got %d", ib.Format.SampleRate)
	}

	ints := ib.Data
	out := make([]float32, len(ints))
	const scale = 1.0 / 32768.0
	for i, v := range ints {
		out[i] = float32(v) * scale
	}
	return out, nil
}
