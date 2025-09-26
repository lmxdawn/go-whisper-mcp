package pkg

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"os/exec"
)

// EnsureFFmpeg 检查系统是否安装了 ffmpeg。
func EnsureFFmpeg() error {
	_, err := exec.LookPath("ffmpeg")
	if err != nil {
		return fmt.Errorf("ffmpeg not found in PATH: %w", err)
	}
	return nil
}

// DecodeF32 一次性内存管道：任意媒体 -> 16kHz/mono float32 PCM（不落盘）
func DecodeF32(ctx context.Context, in string) ([]float32, error) {
	if err := EnsureFFmpeg(); err != nil {
		return nil, err
	}
	args := []string{"-i", in, "-vn", "-ac", "1", "-ar", "16000", "-f", "f32le", "pipe:1"}
	var stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, "ffmpeg", args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	cmd.Stderr = &stderr

	if err := cmd.Start(); err != nil {
		return nil, err
	}
	raw, err := io.ReadAll(stdout)
	if err != nil {
		_ = cmd.Wait()
		return nil, err
	}
	if err := cmd.Wait(); err != nil {
		return nil, fmt.Errorf("ffmpeg: %w: %s", err, stderr.String())
	}
	if len(raw)%4 != 0 {
		return nil, fmt.Errorf("unexpected f32le length %d", len(raw))
	}
	n := len(raw) / 4
	out := make([]float32, n)
	for i := 0; i < n; i++ {
		bits := binary.LittleEndian.Uint32(raw[i*4 : i*4+4])
		out[i] = math.Float32frombits(bits)
	}
	return out, nil
}

// StreamF32 流式内存管道：按块把 float32 PCM 推给回调；更省内存。
// chunkSamples：每次回调的采样点个数（如 16000 = 1s）。
func StreamF32(ctx context.Context, in string, chunkSamples int, onChunk func([]float32) error) error {
	if onChunk == nil {
		return errors.New("onChunk is nil")
	}
	if chunkSamples <= 0 {
		chunkSamples = 16000
	}
	if err := EnsureFFmpeg(); err != nil {
		return err
	}
	args := []string{"-i", in, "-vn", "-ac", "1", "-ar", "16000", "-f", "f32le", "pipe:1"}
	var stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, "ffmpeg", args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	cmd.Stderr = &stderr

	if err := cmd.Start(); err != nil {
		return err
	}

	byteBuf := make([]byte, chunkSamples*4)
	floatBuf := make([]float32, chunkSamples)

	// 连续读 stdout，并把 bytes -> float32 转给回调
	for {
		n, err := io.ReadFull(stdout, byteBuf)
		if err != nil {
			if errors.Is(err, io.ErrUnexpectedEOF) || errors.Is(err, io.EOF) {
				// 末尾不足一个块：n 可能为 0 或非整块
				if n > 0 {
					if n%4 != 0 {
						return fmt.Errorf("partial tail not aligned to 4 bytes: %d", n)
					}
					remain := n / 4
					for i := 0; i < remain; i++ {
						bits := binary.LittleEndian.Uint32(byteBuf[i*4 : i*4+4])
						floatBuf[i] = math.Float32frombits(bits)
					}
					if err2 := onChunk(floatBuf[:remain]); err2 != nil {
						_ = cmd.Wait()
						return err2
					}
				}
				break
			}
			_ = cmd.Wait()
			return err
		}
		// 满块
		for i := 0; i < chunkSamples; i++ {
			bits := binary.LittleEndian.Uint32(byteBuf[i*4 : i*4+4])
			floatBuf[i] = math.Float32frombits(bits)
		}
		if err := onChunk(floatBuf); err != nil {
			_ = cmd.Wait()
			return err
		}
	}

	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("ffmpeg: %w: %s", err, stderr.String())
	}
	return nil
}
