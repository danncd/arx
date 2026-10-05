package library

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
)

type Metadata struct {
	Name         string
	Architecture string
	Context      int
	Numbers      map[string]uint64
	NumberArrays map[string][]uint64
	Template     string
}

type ggufReader struct {
	reader io.Reader
}

func (r *ggufReader) read(value any) error { return binary.Read(r.reader, binary.LittleEndian, value) }
func (r *ggufReader) text() (string, error) {
	var length uint64
	if err := r.read(&length); err != nil {
		return "", err
	}
	if length > 4<<20 {
		return "", errors.New("GGUF metadata string is too large")
	}
	data := make([]byte, length)
	_, err := io.ReadFull(r.reader, data)
	return string(data), err
}
func (r *ggufReader) value(kind uint32, depth int) (any, error) {
	if depth > 1 {
		return nil, errors.New("Unsupported GGUF metadata array")
	}
	switch kind {
	case 8:
		return r.text()
	case 9:
		var element uint32
		var count uint64
		if err := r.read(&element); err != nil {
			return nil, err
		}
		if err := r.read(&count); err != nil {
			return nil, err
		}
		if count > 2_000_000 {
			return nil, errors.New("GGUF metadata array is too large")
		}
		var numbers []uint64
		if count <= 1024 {
			numbers = make([]uint64, 0, count)
		}
		for i := uint64(0); i < count; i++ {
			value, err := r.value(element, depth+1)
			if err != nil {
				return nil, err
			}
			if numbers != nil {
				if number, ok := value.(uint64); ok {
					numbers = append(numbers, number)
				} else {
					numbers = nil
				}
			}
		}
		return numbers, nil
	case 0, 1, 7:
		var v uint8
		err := r.read(&v)
		return uint64(v), err
	case 2, 3:
		var v uint16
		err := r.read(&v)
		return uint64(v), err
	case 4, 5, 6:
		var v uint32
		err := r.read(&v)
		return uint64(v), err
	case 10, 11, 12:
		var v uint64
		err := r.read(&v)
		return v, err
	}
	return nil, errors.New("Unsupported GGUF metadata type")
}

func Inspect(path string) (Metadata, error) {
	file, err := os.Open(path)
	if err != nil {
		return Metadata{}, err
	}
	defer file.Close()
	r := ggufReader{reader: io.LimitReader(file, 64<<20)}
	magic := make([]byte, 4)
	if _, err := io.ReadFull(r.reader, magic); err != nil {
		return Metadata{}, err
	}
	if string(magic) != "GGUF" {
		return Metadata{}, errors.New("Choose a GGUF model file")
	}
	var version uint32
	var tensors, count uint64
	if err := r.read(&version); err != nil {
		return Metadata{}, err
	}
	if version != 2 && version != 3 {
		return Metadata{}, errors.New("Unsupported GGUF version")
	}
	if err := r.read(&tensors); err != nil {
		return Metadata{}, err
	}
	if err := r.read(&count); err != nil {
		return Metadata{}, err
	}
	if count > 100000 {
		return Metadata{}, errors.New("GGUF metadata is too large")
	}
	result := Metadata{Numbers: map[string]uint64{}, NumberArrays: map[string][]uint64{}}
	for i := uint64(0); i < count; i++ {
		key, err := r.text()
		if err != nil {
			return result, err
		}
		var kind uint32
		if err := r.read(&kind); err != nil {
			return result, err
		}
		value, err := r.value(kind, 0)
		if err != nil {
			return result, fmt.Errorf("Invalid GGUF metadata: %w", err)
		}
		if n, ok := value.(uint64); ok {
			result.Numbers[key] = n
		}
		if numbers, ok := value.([]uint64); ok && (strings.HasSuffix(key, ".attention.head_count_kv") || strings.HasSuffix(key, ".attention.sliding_window_pattern")) {
			result.NumberArrays[key] = numbers
		}
		switch key {
		case "general.name":
			result.Name, _ = value.(string)
		case "general.architecture":
			result.Architecture, _ = value.(string)
		case "tokenizer.chat_template":
			result.Template, _ = value.(string)
		}
	}
	if n := result.Numbers[result.Architecture+".context_length"]; n <= 1<<24 {
		result.Context = int(n)
	}
	return result, nil
}
