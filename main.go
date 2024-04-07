package main

import (
	"context"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

type PartOfSpeach string

const (
	unknown   PartOfSpeach = "unknown"
	noun      PartOfSpeach = "noun"
	adjective PartOfSpeach = "adjective"
	verb      PartOfSpeach = "verb"
	adverb    PartOfSpeach = "adverb"
)

type WordPage struct {
	Filename string

	Word         string
	PartOfSpeach PartOfSpeach
}

type PathItem struct {
	path string
	err  error
}

func GetFilePaths(ctx context.Context) <-chan PathItem {
	result := make(chan PathItem)

	go func() {
		defer close(result)
		dirPath := "./us/definition/english"
		err := filepath.WalkDir(dirPath, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			if !d.IsDir() {
				select {
				case <-ctx.Done():
					return ctx.Err()
				case result <- PathItem{path: path}:
				}
			}
			return nil
		})
		if err != nil {
			select {
			case <-ctx.Done():
			case result <- PathItem{err: err}:
			}
		}
	}()

	return result
}

func GetPossiblePartOfSpeaches(ctx context.Context) (map[string][]string, error) {
	resultSet := make(map[string][]string)

	pathItems := GetFilePaths(ctx)

	for pathItem := range pathItems {
		if pathItem.err != nil {
			return nil, pathItem.err
		}

		f, err := os.Open(pathItem.path)
		if err != nil {
			return nil, err
		}
		r, err := goquery.NewDocumentFromReader(f)
		if err != nil {
			return nil, err
		}

		webtopg := r.Find(".webtop")
		word := webtopg.Find("h1.headword").Text()
		pos := webtopg.Find("span.pos").Text()

		posList := strings.Split(pos, ", ")
		for _, v := range posList {
			resultSet[v] = append(resultSet[v], word)
		}

		// resultSet[pos] = append(resultSet[pos], word)
	}

	return resultSet, nil
}

func main() {
	r, err := GetPossiblePartOfSpeaches(context.Background())
	if err != nil {
		log.Fatal(err.Error())
	}
	for k, v := range r {
		fmt.Println(k, v[:1], len(v))
	}
}
