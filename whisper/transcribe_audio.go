package whisper

import (
	"errors"
	"fmt"
	wpk "github.com/ggerganov/whisper.cpp/bindings/go/pkg/whisper"
	"io"
	"time"
)

type TranscribeAudio struct{}

func NewTranscribeAudio() *TranscribeAudio {
	return &TranscribeAudio{}
}

func (a *TranscribeAudio) Transcribe(modelPath string, lang string, threads int, data []float32) ([]TranscribeAudioResult, error) {
	// whisper 处理
	model, err := wpk.New(modelPath)
	if err != nil {
		return nil, fmt.Errorf("load model: %w", err)
	}
	defer model.Close()

	wc, err := model.NewContext()
	if err != nil {
		return nil, err
	}
	if lang == "" {
		lang = "auto"
	}
	_ = wc.SetLanguage(lang)
	wc.SetThreads(uint(threads))

	if err := wc.Process(data, nil, nil, nil); err != nil {
		return nil, err
	}

	var segs []TranscribeAudioResult
	for {
		sg, err := wc.NextSegment()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, err
		}
		segs = append(segs, TranscribeAudioResult{
			Start: sg.Start.Truncate(time.Millisecond).String(),
			End:   sg.End.Truncate(time.Millisecond).String(),
			Text:  sg.Text,
		})
	}

	return segs, nil
}
