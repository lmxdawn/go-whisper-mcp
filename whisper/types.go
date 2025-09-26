package whisper

type TranscribeAudioResult struct {
	Start string `json:"start"`
	End   string `json:"end"`
	Text  string `json:"text"`
}
