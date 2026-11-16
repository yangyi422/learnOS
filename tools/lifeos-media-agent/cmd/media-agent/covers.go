package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"

	"lifeos-media-agent/internal/state"
)

func ensureCover(ctx context.Context, client *http.Client, base *url.URL, token, cache, destination, hash string, queue state.Store) (string, error) {
	id, err := queue.CoverID(ctx, destination, hash)
	if err != nil || id != "" {
		return id, err
	}
	data, err := os.ReadFile(filepath.Join(cache, hash))
	if err != nil {
		return "", err
	}
	if fmt.Sprintf("%x", sha256.Sum256(data)) != hash {
		return "", fmt.Errorf("cached cover hash mismatch")
	}
	var body bytes.Buffer
	form := multipart.NewWriter(&body)
	part, err := form.CreateFormFile("image", "cover")
	if err != nil {
		return "", err
	}
	if _, err = part.Write(data); err != nil {
		return "", err
	}
	if err = form.Close(); err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, base.JoinPath("api/v1/library/media-agent/covers").String(), &body)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", form.FormDataContentType())
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("cover upload returned HTTP %d", resp.StatusCode)
	}
	var receipt struct{ Data struct{ ID string } }
	if err = json.NewDecoder(io.LimitReader(resp.Body, 65536)).Decode(&receipt); err != nil || len(receipt.Data.ID) != 32 {
		return "", fmt.Errorf("invalid cover upload receipt")
	}
	id = receipt.Data.ID
	return id, queue.SaveCoverID(ctx, destination, hash, id)
}
