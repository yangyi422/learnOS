package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"syscall"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"lifeos-media-agent/internal/source"
	"lifeos-media-agent/internal/state"
)

func scanAll(ctx context.Context, name string, emit func(source.Item) error) error {
	if name != "all" {
		return scan(ctx, name, emit)
	}
	return errors.Join(scan(ctx, "calibre", emit), scan(ctx, "jellyfin", emit))
}

func synchronize(ctx context.Context, name string, limit int) (failure error) {
	if name != "all" && name != "calibre" && name != "jellyfin" {
		return fmt.Errorf("unknown source")
	}
	base, err := lifeOSURL()
	if err != nil {
		return err
	}
	token, err := keychain(ctx, "lifeos-media-agent-lifeos-token")
	if err != nil {
		return err
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	dir := filepath.Join(home, "Library", "Application Support", "LifeOSMediaAgent")
	if err = os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	lock, err := os.OpenFile(filepath.Join(dir, "sync.lock"), os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return err
	}
	defer lock.Close()
	if err = syscall.Flock(int(lock.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		return fmt.Errorf("cannot acquire sync lock: %w", err)
	}
	db, err := sql.Open("sqlite3", filepath.Join(dir, "state.db"))
	if err != nil {
		return err
	}
	defer db.Close()
	queue, err := state.Init(ctx, db)
	if err != nil {
		return err
	}
	run, err := queue.Start(ctx)
	if err != nil {
		return err
	}
	defer func() {
		finish, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer cancel()
		failure = errors.Join(failure, queue.Finish(finish, run, failure))
	}()
	cache := filepath.Join(dir, "covers")
	if err = os.MkdirAll(cache, 0700); err != nil {
		return err
	}
	destination := fmt.Sprintf("%x", sha256.Sum256([]byte(base.String()+"\x00"+token)))
	staged := 0
	scanErr := scanAll(ctx, name, func(item source.Item) error {
		if limit > 0 && staged >= limit {
			return nil
		}
		staged++
		hash := ""
		if item.ReadCover != nil {
			data, err := item.ReadCover(ctx)
			if err != nil {
				return err
			}
			hash = fmt.Sprintf("%x", sha256.Sum256(data))
			if err = os.WriteFile(filepath.Join(cache, hash), data, 0600); err != nil {
				return err
			}
		}
		payload, err := json.Marshal(item)
		if err != nil {
			return err
		}
		return queue.Stage(ctx, []state.Item{{Source: item.Source, SourceID: item.SourceID, ExternalID: item.ExternalID, MetadataHash: fmt.Sprintf("%x", sha256.Sum256(payload)), CoverHash: hash, Payload: payload}})
	})
	client := &http.Client{Timeout: 30 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	count := 0
	origins, err := queue.Origins(ctx, name)
	if err != nil {
		return errors.Join(scanErr, err)
	}
	for _, origin := range origins {
		for {
			size := 50
			if limit > 0 && size > limit-count {
				size = limit - count
			}
			if size == 0 {
				break
			}
			batch, err := queue.Pending(ctx, origin[0], origin[1], size)
			if err != nil {
				return errors.Join(scanErr, err)
			}
			if len(batch) == 0 {
				break
			}
			items := make([]json.RawMessage, len(batch))
			for i, item := range batch {
				items[i] = item.Payload
				if item.CoverHash != "" {
					id, err := ensureCover(ctx, client, base, token, cache, destination, item.CoverHash, queue)
					if err != nil {
						return errors.Join(scanErr, err)
					}
					var payload map[string]json.RawMessage
					if err = json.Unmarshal(item.Payload, &payload); err != nil {
						return err
					}
					payload["cover_id"], err = json.Marshal(id)
					if err != nil {
						return err
					}
					items[i], err = json.Marshal(payload)
					if err != nil {
						return err
					}
				}
			}
			body, err := json.Marshal(items)
			if err != nil {
				return errors.Join(scanErr, err)
			}
			req, err := http.NewRequestWithContext(ctx, http.MethodPost, base.JoinPath("api/v1/library/media-agent/items").String(), bytes.NewReader(body))
			if err != nil {
				return errors.Join(scanErr, err)
			}
			req.Header.Set("Authorization", "Bearer "+token)
			req.Header.Set("Content-Type", "application/json")
			response, err := client.Do(req)
			if err != nil {
				return errors.Join(scanErr, err)
			}
			if response.StatusCode != http.StatusOK {
				response.Body.Close()
				return errors.Join(scanErr, fmt.Errorf("LifeOS returned HTTP %d", response.StatusCode))
			}
			var receipt struct{ Data struct{ Accepted int } }
			decodeErr := json.NewDecoder(io.LimitReader(response.Body, 65536)).Decode(&receipt)
			response.Body.Close()
			if decodeErr != nil || receipt.Data.Accepted != len(batch) {
				return errors.Join(scanErr, fmt.Errorf("LifeOS did not acknowledge the entire batch"))
			}
			if err := queue.Acknowledge(ctx, batch); err != nil {
				return errors.Join(scanErr, err)
			}
			count += len(batch)
		}
	}
	fmt.Printf("acknowledged=%d\n", count)
	return scanErr
}
