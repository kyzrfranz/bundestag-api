package http

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

const (
	ErrResourceNotFound = "resource not found: %s"
)

func FetchUrl(url *url.URL) ([]byte, error) {
	res, err := http.Get(url.String())
	if err != nil {
		return nil, err
	}

	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(ErrResourceNotFound, url.String())
	}

	// Read the response body
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	return body, nil
}

func FetchUrlAsBrowser(url *url.URL) ([]byte, error) {
	// Create a new GET request.
	req, err := http.NewRequest("GET", url.String(), nil)
	if err != nil {
		return nil, err
	}

	// Set headers to mimic a browser.
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) "+
		"AppleWebKit/537.36 (KHTML, like Gecko) Chrome/112.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "application/json, text/javascript, */*; q=0.01")
	// Optionally add a Referer header if the API expects one.
	req.Header.Set("Referer", "https://www.bundestag.de/")

	// Use http.DefaultClient (or create a custom client if needed)
	client := http.DefaultClient
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	return handleResponse(*url, *res)
}

func handleResponse(url url.URL, res http.Response) ([]byte, error) {
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch URL %s: status %s", url.String(), res.Status)
	}

	// Read the response body
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	return body, nil
}

type RWCache interface {
	Read() ([]byte, error)
	Write(data []byte) error
}

func FetchCachedUrl(url *url.URL, cache RWCache) ([]byte, error) {
	// Read or create the cache file
	data, err := cache.Read()
	if err != nil {
		return nil, err
	}

	// Initialize the cache map from the file
	c := make(map[string][]byte)
	if len(data) > 0 {
		if err := json.Unmarshal(data, &c); err != nil {
			return nil, fmt.Errorf("failed to unmarshal cache file: %w", err)
		}
	}

	// Check for a cache hit
	entry, hit := c[url.String()]
	if hit {
		return entry, nil
	}

	// Fetch data if not cached
	data, err = FetchUrl(url)
	if err != nil {
		return nil, err
	}

	// Store in the cache map
	c[url.String()] = data

	// Write updated cache to file
	cacheData, err := json.Marshal(c)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal cache data: %w", err)
	}

	if err := cache.Write(cacheData); err != nil {
		return nil, err
	}

	return data, nil
}
