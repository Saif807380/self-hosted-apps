package tracker

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	maxCoverBytes = 4 << 20 // IGDB's 2x covers are ~100–300 KB
	// IGDB serves the same image at several sizes; 2x keeps covers sharp on
	// a phone. t_cover_big is what catalog stores.
	coverSize   = "t_cover_big"
	coverSizeHD = "t_cover_big_2x"
)

var coverHTTP = &http.Client{Timeout: 30 * time.Second}

// FillBlanks gives games the cover and studio they're missing, from their
// IGDB match (or just the given game's). Nothing you've set is replaced.
func (t *Tracker) FillBlanks(ctx context.Context, id *uuid.UUID) error {
	if err := t.FillStudios(ctx, id); err != nil {
		return err
	}
	return t.FillCovers(ctx, id)
}

// FillStudios sets a blank studio to the game's publisher (see StudioFor).
func (t *Tracker) FillStudios(ctx context.Context, id *uuid.UUID) error {
	targets, err := t.Store.GamesMissingStudio(ctx, id)
	if err != nil {
		return err
	}
	for _, target := range targets {
		if err := t.Store.SetStudioIfEmpty(ctx, target.ID, StudioFor(target.Publishers, target.Developers)); err != nil {
			return err
		}
	}
	return nil
}

// StudioFor picks the name shown as a game's studio: its first publisher,
// or its first developer when IGDB lists no publisher.
func StudioFor(publishers, developers []string) string {
	if len(publishers) > 0 {
		return publishers[0]
	}
	if len(developers) > 0 {
		return developers[0]
	}
	return ""
}

// FillCovers gives games without a cover their IGDB cover (or just the given
// game). Covers are saved like an upload, as /uploads/<uuid>.jpg, so the
// PWA's image sync takes them offline like any other cover.
func (t *Tracker) FillCovers(ctx context.Context, id *uuid.UUID) error {
	if t.UploadsDir == "" {
		return nil
	}
	targets, err := t.Store.GamesMissingCover(ctx, id)
	if err != nil {
		return err
	}
	for _, target := range targets {
		path, err := t.saveCover(ctx, strings.Replace(target.CoverURL, coverSize, coverSizeHD, 1))
		if err != nil {
			t.Log.Warn("cover download failed", "game", target.ID, "err", err)
			continue
		}
		set, err := t.Store.SetCoverIfEmpty(ctx, target.ID, path)
		if err != nil {
			return err
		}
		if !set { // you added one meanwhile
			os.Remove(filepath.Join(t.UploadsDir, filepath.Base(path))) //nolint:errcheck
		}
	}
	return nil
}

// saveCover downloads a JPEG cover into the uploads directory and returns
// its /uploads path.
func (t *Tracker) saveCover(ctx context.Context, url string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	resp, err := coverHTTP.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "image/jpeg" {
		return "", fmt.Errorf("unexpected content type %q", ct)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxCoverBytes+1))
	if err != nil {
		return "", err
	}
	if len(data) > maxCoverBytes {
		return "", fmt.Errorf("cover larger than %d bytes", maxCoverBytes)
	}

	name := uuid.NewString() + ".jpg"
	// Write then rename, so image sync never serves a half-written file.
	tmp := filepath.Join(t.UploadsDir, "."+name+".tmp")
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return "", err
	}
	if err := os.Rename(tmp, filepath.Join(t.UploadsDir, name)); err != nil {
		os.Remove(tmp) //nolint:errcheck
		return "", err
	}
	return "/uploads/" + name, nil
}
