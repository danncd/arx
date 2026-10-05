package download

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type File struct {
	Name   string `json:"name"`
	URL    string `json:"url"`
	Size   int64  `json:"size"`
	SHA256 string `json:"sha256,omitempty"`
}

var Client = &http.Client{Transport: &http.Transport{Proxy: http.ProxyFromEnvironment, ResponseHeaderTimeout: 30 * time.Second}, CheckRedirect: func(req *http.Request, via []*http.Request) error {
	if len(via) >= 10 || req.URL.Scheme != "https" {
		return errors.New("Invalid download redirect")
	}
	return nil
}}

func Transfer(ctx context.Context, file File, destination string, progress func(int64)) error {
	return TransferChecked(ctx, file, destination, progress, nil)
}

func TransferChecked(ctx context.Context, file File, destination string, progress func(int64), checkSpace func(int64) error) error {
	if file.Size <= 0 {
		return errors.New("Download size is unavailable")
	}
	if err := os.MkdirAll(filepath.Dir(destination), 0700); err != nil {
		return err
	}
	if info, err := os.Stat(destination); err == nil && info.Size() == file.Size {
		if err := verify(ctx, destination, file); err == nil {
			progress(file.Size)
			return nil
		}
	}
	partial := destination + ".part"
	offset := int64(0)
	if info, err := os.Stat(partial); err == nil {
		offset = info.Size()
	}
	if offset > file.Size {
		offset = 0
	}
	if checkSpace != nil {
		if err := checkSpace(file.Size - offset); err != nil {
			return err
		}
	}
	if offset < file.Size {
		request, err := http.NewRequestWithContext(ctx, "GET", file.URL, nil)
		if err != nil {
			return err
		}
		request.Header.Set("Accept-Encoding", "identity")
		if offset > 0 {
			request.Header.Set("Range", fmt.Sprintf("bytes=%d-", offset))
		}
		response, err := Client.Do(request)
		if err != nil {
			return err
		}
		defer response.Body.Close()
		if response.StatusCode != 200 && response.StatusCode != 206 {
			return fmt.Errorf("Download failed (HTTP %d)", response.StatusCode)
		}
		if response.StatusCode == 206 {
			expected := fmt.Sprintf("bytes %d-", offset)
			if !strings.HasPrefix(response.Header.Get("Content-Range"), expected) || !strings.HasSuffix(response.Header.Get("Content-Range"), fmt.Sprintf("/%d", file.Size)) {
				return errors.New("Download range changed; restart the download")
			}
		} else {
			if offset > 0 && checkSpace != nil {
				if err := checkSpace(file.Size); err != nil {
					return err
				}
			}
			offset = 0
		}
		flags := os.O_CREATE | os.O_WRONLY
		if offset == 0 {
			flags |= os.O_TRUNC
		}
		output, err := os.OpenFile(partial, flags, 0600)
		if err != nil {
			return err
		}
		if _, err = output.Seek(offset, io.SeekStart); err != nil {
			output.Close()
			return err
		}
		buffer := make([]byte, 256<<10)
		progress(offset)
		for {
			if ctx.Err() != nil {
				err = ctx.Err()
				break
			}
			var n int
			n, err = response.Body.Read(buffer)
			if n > 0 {
				if offset+int64(n) > file.Size {
					err = errors.New("Download exceeds its declared size")
					break
				}
				if _, writeErr := output.Write(buffer[:n]); writeErr != nil {
					err = writeErr
					break
				}
				offset += int64(n)
				progress(offset)
			}
			if err != nil {
				break
			}
		}
		syncErr := output.Sync()
		closeErr := output.Close()
		if err != nil && err != io.EOF {
			return err
		}
		if syncErr != nil {
			return syncErr
		}
		if closeErr != nil {
			return closeErr
		}
		if offset != file.Size {
			return errors.New("Download ended before the file was complete")
		}
	}
	if err := verify(ctx, partial, file); err != nil {
		if ctx.Err() == nil {
			os.Remove(partial)
		}
		return err
	}
	return os.Rename(partial, destination)
}

func verify(ctx context.Context, path string, file File) error {
	if file.SHA256 == "" {
		return nil
	}
	input, err := os.Open(path)
	if err != nil {
		return err
	}
	defer input.Close()
	hash := sha256.New()
	buffer := make([]byte, 1<<20)
	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		n, err := input.Read(buffer)
		if n > 0 {
			hash.Write(buffer[:n])
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
	}
	if hex.EncodeToString(hash.Sum(nil)) != file.SHA256 {
		return errors.New("Download checksum did not match")
	}
	return nil
}

func Remaining(ctx context.Context, file File, destination string) (int64, error) {
	if file.Size <= 0 {
		return 0, errors.New("Download size is unavailable")
	}
	if info, err := os.Stat(destination); err == nil && info.Mode().IsRegular() && info.Size() == file.Size {
		if err := verify(ctx, destination, file); err == nil {
			return 0, nil
		}
		if ctx.Err() != nil {
			return 0, ctx.Err()
		}
	}
	if info, err := os.Stat(destination + ".part"); err == nil && info.Mode().IsRegular() && info.Size() <= file.Size {
		if info.Size() < file.Size {
			return file.Size - info.Size(), nil
		}
		if err := verify(ctx, destination+".part", file); err == nil {
			return 0, nil
		}
		if ctx.Err() != nil {
			return 0, ctx.Err()
		}

		if err := os.Remove(destination + ".part"); err != nil {
			return 0, err
		}
	}
	return file.Size, nil
}
