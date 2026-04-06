package handlers

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"

	"backend/internal/config"
	"backend/internal/dto"
	"backend/internal/middleware"
	"backend/internal/models"
	"backend/internal/ocr"
	"backend/internal/repositories"
	"backend/internal/storage"
)

// ReceiptHandler handles receipt uploads and OCR.
type ReceiptHandler struct {
	storage  *storage.Client
	receipts *repositories.ReceiptRepository
	ocr      ocr.Adapter
	cfg      *config.Config
}

// NewReceiptHandler constructs handler.
func NewReceiptHandler(storage *storage.Client, receipts *repositories.ReceiptRepository, ocrAdapter ocr.Adapter, cfg *config.Config) *ReceiptHandler {
	return &ReceiptHandler{
		storage:  storage,
		receipts: receipts,
		ocr:      ocrAdapter,
		cfg:      cfg,
	}
}

// Upload handles receipt upload with OCR.
func (h *ReceiptHandler) Upload(c *fiber.Ctx) error {
	user := middleware.GetUser(c)
	if user == nil {
		return fiber.ErrUnauthorized
	}

	fileHeader, err := c.FormFile("file")
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "file is required")
	}

	if fileHeader.Size == 0 || fileHeader.Size > h.cfg.MaxUploadSizeBytes {
		return fiber.NewError(fiber.StatusBadRequest, "file size exceeds allowed limit")
	}

	data, contentType, err := readFileData(fileHeader)
	if err != nil {
		return err
	}

	if !isAllowedMime(contentType) {
		return fiber.NewError(fiber.StatusBadRequest, "unsupported file type")
	}

	checksum := fmt.Sprintf("%x", sha256.Sum256(data))
	ctx := requestContext(c)
	path, err := h.storage.Upload(ctx, "", bytes.NewReader(data), int64(len(data)), contentType)
	if err != nil {
		return err
	}

	file := &models.ReceiptFile{
		UploadedBy:       user.ID,
		StoragePath:      path,
		OriginalFilename: fileHeader.Filename,
		MimeType:         contentType,
		FileSize:         int64(len(data)),
		Checksum:         checksum,
	}

	if err := h.receipts.CreateFile(ctx, file); err != nil {
		return err
	}

	resp := fiber.Map{"file": file}
	if draft := h.runOCR(ctx, file, data, contentType); draft != nil {
		resp["ocrDraft"] = draft
	}

	return c.Status(fiber.StatusCreated).JSON(resp)
}

// List returns recent receipts for the user.
func (h *ReceiptHandler) List(c *fiber.Ctx) error {
	user := middleware.GetUser(c)
	if user == nil {
		return fiber.ErrUnauthorized
	}

	limit := 20
	if raw := c.Query("limit"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	ctx := requestContext(c)
	files, err := h.receipts.ListFilesByUser(ctx, user.ID, limit)
	if err != nil {
		return err
	}

	items := make([]fiber.Map, 0, len(files))
	for _, file := range files {
		entry := fiber.Map{
			"id":                file.ID,
			"original_filename": file.OriginalFilename,
			"mime_type":         file.MimeType,
			"file_size":         file.FileSize,
			"checksum":          file.Checksum,
			"created_at":        file.CreatedAt,
		}
		if url, err := h.storage.Presign(ctx, file.StoragePath); err == nil {
			entry["download_url"] = url.String()
		}
		items = append(items, entry)
	}

	return c.JSON(fiber.Map{"items": items})
}

func (h *ReceiptHandler) runOCR(ctx context.Context, file *models.ReceiptFile, data []byte, contentType string) *dto.OCRDraftResponse {
	if h.ocr == nil {
		return nil
	}

	result, err := h.ocr.Recognize(ctx, ocr.RecognizeRequest{
		Filename:    file.OriginalFilename,
		ContentType: contentType,
		Data:        data,
	})
	if err != nil {
		log.Printf("ocr recognition failed: %v", err)
		return nil
	}

	rec := &models.ReceiptRecognition{
		ReceiptFileID:   file.ID,
		RawResponseJSON: result.RawJSON,
		Status:          "completed",
		ProcessedAt:     ptrTime(time.Now()),
	}

	normalizedDate, parsedDate := normalizePurchaseDate(result.Draft.PurchaseDate)
	if parsedDate != nil {
		rec.ExtractedDate = parsedDate
	}
	if result.Draft.TotalAmount != 0 {
		rec.ExtractedAmount = &result.Draft.TotalAmount
	}
	if result.Draft.Currency != "" {
		rec.ExtractedCurrency = result.Draft.Currency
	}
	if result.Draft.VendorName != "" {
		rec.ExtractedVendor = result.Draft.VendorName
	}
	if result.Draft.TaxAmount != 0 {
		rec.ExtractedTax = &result.Draft.TaxAmount
	}

	if err := h.receipts.CreateRecognition(ctx, rec); err != nil {
		log.Printf("failed to persist OCR result: %v", err)
	}

	return &dto.OCRDraftResponse{
		ExpenseDate: normalizedDate,
		Amount:      result.Draft.TotalAmount,
		Currency:    result.Draft.Currency,
		Vendor:      result.Draft.VendorName,
		Tax:         result.Draft.TaxAmount,
		Raw:         jsonRawToInterface(result.RawJSON),
	}
}

func readFileData(header *multipart.FileHeader) ([]byte, string, error) {
	src, err := header.Open()
	if err != nil {
		return nil, "", fiber.NewError(fiber.StatusBadRequest, "failed to open file")
	}
	defer src.Close()

	data, err := io.ReadAll(src)
	if err != nil {
		return nil, "", fiber.NewError(fiber.StatusBadRequest, "failed to read file")
	}

	contentType := header.Header.Get("Content-Type")
	if contentType == "" {
		contentType = http.DetectContentType(data)
	}

	return data, contentType, nil
}

func isAllowedMime(mime string) bool {
	switch strings.ToLower(mime) {
	case "image/jpeg", "image/png", "image/jpg":
		return true
	default:
		return false
	}
}

func ptrTime(t time.Time) *time.Time {
	return &t
}

func jsonRawToInterface(raw []byte) interface{} {
	if len(raw) == 0 {
		return nil
	}
	var v interface{}
	if err := json.Unmarshal(raw, &v); err != nil {
		return nil
	}
	return v
}

func normalizePurchaseDate(value string) (string, *time.Time) {
	if value == "" {
		return "", nil
	}
	layouts := []string{
		"2006-01-02",
		time.RFC3339,
		"02/01/2006",
		"01/02/2006",
		"02.01.2006",
		"01.02.2006",
		"2006/01/02",
		"02-01-2006",
		"01-02-2006",
	}
	for _, layout := range layouts {
		if parsed, err := time.Parse(layout, value); err == nil {
			date := parsed.Format("2006-01-02")
			parsedDate := parsed
			return date, &parsedDate
		}
	}
	return "", nil
}
