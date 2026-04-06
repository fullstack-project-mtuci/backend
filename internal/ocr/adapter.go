package ocr

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"path"
	"strings"
	"time"

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
	ReceiptID string `json:"receipt_id"`
	Draft     struct {
		VendorName   string          `json:"vendor_name"`
		PurchaseDate string          `json:"purchase_date"`
		TotalAmount  float64         `json:"total_amount"`
		Currency     string          `json:"currency"`
		TaxAmount    float64         `json:"tax_amount"`
		ModelName    string          `json:"model_name"`
		Status       string          `json:"status"`
		RawText      string          `json:"raw_text_like_output"`
		Source       string          `json:"source"`
		Items        json.RawMessage `json:"items"`
	} `json:"draft"`
	RawPrediction    json.RawMessage        `json:"raw_prediction"`
	ModelName        string                 `json:"model_name"`
	Status           string                 `json:"status"`
	ProcessedAt      time.Time              `json:"processed_at"`
	ProcessingTimeMs int                    `json:"processing_time_ms"`
	Metadata         map[string]interface{} `json:"metadata"`
	RawJSON          json.RawMessage
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
		client: &http.Client{Timeout: time.Minute * 2},
		cfg:    cfg,
	}
}

func (a *httpAdapter) Recognize(ctx context.Context, req RecognizeRequest) (*RecognizeResult, error) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	partHeaders := textproto.MIMEHeader{}
	partHeaders.Set("Content-Disposition", fmt.Sprintf(`form-data; name="%s"; filename="%s"`, "file", fileNameOrDefault(req.Filename)))
	contentType := req.ContentType
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	partHeaders.Set("Content-Type", contentType)

	part, err := writer.CreatePart(partHeaders)
	if err != nil {
		return nil, fmt.Errorf("create form file: %w", err)
	}
	if _, err := part.Write(req.Data); err != nil {
		return nil, fmt.Errorf("write file data: %w", err)
	}
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
	fmt.Println(string(respBody))

	return &parsed, nil
}

func fileNameOrDefault(name string) string {
	if name == "" {
		return path.Base("receipt.jpg")
	}
	return path.Base(name)
}
