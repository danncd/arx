package main

import (
	"fmt"
	"go/parser"
	"go/scanner"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
)

func lexical(source []byte) ([][]string, [][2]int) {
	set := token.NewFileSet()
	file := set.AddFile("", -1, len(source))
	var scan scanner.Scanner
	scan.Init(file, source, nil, scanner.ScanComments)
	tokens := [][]string{}
	comments := [][2]int{}
	for {
		pos, kind, literal := scan.Scan()
		if kind == token.EOF {
			break
		}
		if kind == token.COMMENT {
			start := file.Offset(pos)
			comments = append(comments, [2]int{start, start + len(literal)})
			continue
		}
		tokens = append(tokens, []string{kind.String(), literal})
	}
	return tokens, comments
}
func main() {
	root, err := filepath.Abs(os.Args[1])
	if err != nil {
		panic(err)
	}
	strip := len(os.Args) > 2 && os.Args[2] == "--strip"
	failures := []string{}
	removed := 0
	err = filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if entry.Name() == "node_modules" || entry.Name() == ".build" || entry.Name() == "dist" || entry.Name() == ".git" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		before, comments := lexical(data)
		if len(comments) > 0 && strip {
			for i := len(comments) - 1; i >= 0; i-- {
				c := comments[i]
				data = append(data[:c[0]], data[c[1]:]...)
			}
			after, _ := lexical(data)
			if !reflect.DeepEqual(before, after) {
				return fmt.Errorf("comment removal changed tokens: %s", path)
			}
			if err := os.WriteFile(path, data, 0644); err != nil {
				return err
			}
			removed += len(comments)
		} else if len(comments) > 0 {
			failures = append(failures, "Source comments: "+path)
		}
		f, err := parser.ParseFile(token.NewFileSet(), path, data, 0)
		if err != nil {
			return err
		}
		relative, _ := filepath.Rel(root, path)
		if strings.HasPrefix(relative, "backend/internal/") && !strings.HasPrefix(relative, "backend/internal/app/") && !strings.HasPrefix(relative, "backend/internal/transport/") {
			for _, i := range f.Imports {
				p, _ := strconv.Unquote(i.Path.Value)
				if p == "arx/internal/app" || strings.HasPrefix(p, "arx/internal/transport") {
					failures = append(failures, "Feature imports composition/transport: "+relative)
				}
			}
		}
		if strings.HasPrefix(relative, "backend/internal/platform/") {
			for _, i := range f.Imports {
				p, _ := strconv.Unquote(i.Path.Value)
				if strings.HasPrefix(p, "arx/internal/") && !strings.HasPrefix(p, "arx/internal/platform/") {
					failures = append(failures, "Platform imports feature: "+relative)
				}
			}
		}
		return nil
	})
	if err != nil {
		panic(err)
	}
	if len(failures) > 0 {
		fmt.Fprintln(os.Stderr, strings.Join(failures, "\n"))
		os.Exit(1)
	}
	fmt.Printf("Go architecture/comment policy passed; removed %d comment tokens\n", removed)
}
