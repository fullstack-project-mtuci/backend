package ocr

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"path"
	"strings"

	"backend/internal/config"
)

// RecognizeRequest encapsulates file content to send to OCR provider.
type RecognizeRequest struct {
	Filename    string
	ContentType string
	Data        []byte
}

// RecognizeResult contains parsed fields from OCR provider.
type RecognizeResult struct {
	Date       string  `json:"date"`
	Amount     float64 `json:"amount"`
	Currency   string  `json:"currency"`
	Vendor     string  `json:"vendor"`
	TaxAmount  float64 `json:"tax_amount"`
	Confidence float64 `json:"confidence"`
	RawJSON    json.RawMessage
}

// Adapter performs OCR recognition.
type Adapter interface {
	Recognize(ctx context.Context, req RecognizeRequest) (*RecognizeResult, error)
}

type httpAdapter struct {
	client *http.Client
	cfg    config.OCRConfig
}

// NewHTTPAdapter builds HTTP-based OCR adapter.
func NewHTTPAdapter(cfg config.OCRConfig) Adapter {
	return &httpAdapter{
		client: &http.Client{Timeout: cfg.Timeout},
		cfg:    cfg,
	}
}

func (a *httpAdapter) Recognize(ctx context.Context, req RecognizeRequest) (*RecognizeResult, error) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", fileNameOrDefault(req.Filename))
	if err != nil {
		return nil, fmt.Errorf("create form file: %w", err)
	}
	if _, err := part.Write(req.Data); err != nil {
		return nil, fmt.Errorf("write file data: %w", err)
	}
	_ = writer.WriteField("content_type", req.ContentType)
	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("close writer: %w", err)
	}

	apiURL := strings.TrimRight(a.cfg.BaseURL, "/") + "/recognize"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, &body)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}

	httpReq.Header.Set("Content-Type", writer.FormDataContentType())
	if a.cfg.APIKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+a.cfg.APIKey)
	}

	resp, err := a.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("perform request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("ocr provider error: %s", respBody)
	}

	var parsed RecognizeResult
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}
	parsed.RawJSON = json.RawMessage(respBody)

	return &parsed, nil
}

func fileNameOrDefault(name string) string {
	if name == "" {
		return path.Base("receipt.jpg")
	}
	return path.Base(name)
}
