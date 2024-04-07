package upload

import (
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"

	"github.com/PuerkitoBio/goquery"
	"golang.org/x/sync/errgroup"
)

func Must[T any](value T, err error) T {
	if err != nil {
		log.Fatal(err.Error())
	}
	return value
}

func GetLinks(ctx context.Context) <-chan string {
	content := Must(os.Open("./words.html"))

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

	parsedRel, err := url.Parse(relativePath)
	if err != nil {
		return nil, err
	}

	resolvedURL := parsedURL.ResolveReference(parsedRel)

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, resolvedURL.String(), nil)

	if err != nil {
		return nil, err
	}
	request.Header.Add("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7")
	request.Header.Add("Accept-Encoding", "gzip, deflate, br, zstd")
	request.Header.Add("Accept-Language", "en-US;q=0.8,en;q=0.7")
	request.Header.Add("Cache-Control", "max-age=0")
	request.Header.Add("Connection", "keep-alive")
	request.Header.Add("DNT", "1")
	request.Header.Add("Host", "www.oxfordlearnersdictionaries.com")
	request.Header.Add("Sec-Fetch-Dest", "document")
	request.Header.Add("Sec-Fetch-Mode", "navigate")
	request.Header.Add("Sec-Fetch-Site", "same-origin")
	request.Header.Add("Sec-Fetch-User", "?1")
	request.Header.Add("Upgrade-Insecure-requestuests", "1")
	request.Header.Add("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/123.0.0.0 Safari/537.36")
	request.Header.Add("sec-ch-ua", "Google")
	request.Header.Add("sec-ch-ua-mobile", "?0")
	request.Header.Add("sec-ch-ua-platform", "macOS")

	response, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("invalid status_code: %d, url: %s", response.StatusCode, resolvedURL)
	}

	gzipReader, err := gzip.NewReader(response.Body)
	if err != nil {
		return nil, err
	}
	defer gzipReader.Close()

	content, err := io.ReadAll(gzipReader)
	if err != nil {
		return nil, err
	}
	return content, nil
}

func SavePage(path string, content []byte) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.Write(content)
	if err != nil {
		return err
	}
	return nil
}

// /us/definition/english/a_1
func HandleUrl(ctx context.Context, relUrl string) error {
	dir, filePath := filepath.Split(relUrl + ".html")
	err := os.MkdirAll("."+dir, 0775)
	if err != nil {
		return err
	}

	_, err = os.Stat("." + filepath.Join(dir, filePath))
	if !errors.Is(err, os.ErrNotExist) {
		return nil
	}

	os.Stat("." + filepath.Join(dir, filePath))

	content, err := GetMeaningPage(ctx, relUrl)
	if err != nil {
		return err
	}

	err = SavePage("."+filepath.Join(dir, filePath), content)
	if err != nil {
		return err
	}

	return nil
}

func DownloadPagesFromList(ctx context.Context) {
	ch := GetLinks(ctx)

	group, errCtx := errgroup.WithContext(ctx)
	group.SetLimit(2)

	for v := range ch {
		v := v
		group.Go(func() error {
			return HandleUrl(errCtx, v)
		})
	}
	err := group.Wait()
	if err != nil {
		log.Fatal(err.Error())
	}
}
