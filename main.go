package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"

	"github.com/PuerkitoBio/goquery"
)

func Must[T any](value T, err error) T {
	if err != nil {
		panic(err)
	}
	return value
}

func GetLinks(ctx context.Context) <-chan string {
	content := Must(os.Open("./content.html"))

	reader := Must(goquery.NewDocumentFromReader(content))

	ch := make(chan string)

	go func() {
		defer close(ch)
		reader.Find("li").Each(func(i int, s *goquery.Selection) {
			href, ok := s.Find("a").Attr("href")
			if ok {
				select {
				case <-ctx.Done():
				case ch <- href:
				}
			}
		})
	}()

	return ch
}

const baseUrl = "https://www.oxfordlearnersdictionaries.com"

var client = http.DefaultClient

func GetMeaningPage(ctx context.Context, relativePath string) ([]byte, error) {
	parsedURL, err := url.Parse(baseUrl)
	if err != nil {
		return nil, err
	}

	resolvedURL := parsedURL.ResolveReference(&url.URL{Path: relativePath})

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, resolvedURL.String(), nil)
	if err != nil {
		return nil, err
	}

	response, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	response.Body.Close()

	content, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	return content, nil
}

func main() {
	ctx := context.Background()

	ch := GetLinks(ctx)

	for v := range ch {
		fmt.Println(v)
		// content := Must(GetMeaningPage(ctx, v))
		// fmt.Println(content)
		// time.Sleep(5 * time.Second)
	}
}
