package upstream

import (
	"encoding/xml"
	"fmt"
	myhttp "github.com/kyzrfranz/bundestag-api/internal/http"
	"net/url"
)

type XMLFetcher struct {
	Url *url.URL
}

func (p *XMLFetcher) Fetch() ([]byte, error) {
	data, err := myhttp.FetchUrl(p.Url)
	if err != nil {
		return nil, err
	}

	var xmlData interface{}
	if err = xml.Unmarshal(data, &xmlData); err != nil {
		return nil, fmt.Errorf("invalid XML response: %w", err)
	}

	return data, nil
}
