package result

type Result struct {
	Success       bool   `json:"success"`
	Error         string `json:"error"`
	VideoID       string `json:"videoId"`
	KeywordsURL   string `json:"keywordsUrl"`
	TranscriptURL string `json:"transcriptUrl"`
}
