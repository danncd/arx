package library

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
)

var shardName = regexp.MustCompile(`(?i)^(.*)-[0-9]{5}-of-([0-9]{5})\.gguf$`)

func (e Entry) ModelFiles() ([]string, error) {
	files := []string{e.Path}
	if match := shardName.FindStringSubmatch(e.Path); match != nil {
		count, _ := strconv.Atoi(match[2])
		if count < 1 || count > 1024 {
			return nil, errors.New("Invalid model shard count")
		}
		files = nil
		for i := 1; i <= count; i++ {
			files = append(files, fmt.Sprintf("%s-%05d-of-%s.gguf", match[1], i, match[2]))
		}
	}
	if e.Projector != "" {
		files = append(files, e.Projector)
	}
	return files, nil
}
func (e Entry) FileSize() (int64, error) {
	files, err := e.ModelFiles()
	if err != nil {
		return 0, err
	}
	var size int64
	for _, file := range files {
		stat, err := os.Stat(file)
		if err != nil {
			return 0, err
		}
		if !stat.Mode().IsRegular() {
			return 0, errors.New("Choose a model file")
		}
		size += stat.Size()
	}
	return size, nil
}
func DeleteImported(entry Entry, others []Entry) error {
	files, err := entry.ModelFiles()
	if err != nil {
		return err
	}
	for _, file := range files {
		if !filepath.IsAbs(file) {
			return errors.New("Invalid imported model path")
		}
		stat, err := os.Lstat(file)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return err
		}
		if !stat.Mode().IsRegular() {
			return errors.New("Imported model path is no longer a regular file")
		}
		for _, other := range others {
			if other.ID == entry.ID {
				continue
			}
			held, err := other.ModelFiles()
			if err != nil {
				return err
			}
			for _, path := range held {
				info, err := os.Stat(path)
				if err == nil && os.SameFile(stat, info) {
					return errors.New("This model shares files with another library entry")
				}
			}
		}
	}
	for _, file := range files {
		if err := os.Remove(file); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	return nil
}
