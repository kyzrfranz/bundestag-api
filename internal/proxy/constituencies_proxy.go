package proxy

import (
	"encoding/json"
	"fmt"
	v1 "github.com/kyzrfranz/bundestag-api/api/v1"
	myHttp "github.com/kyzrfranz/bundestag-api/internal/http"
	"github.com/kyzrfranz/bundestag-api/internal/rest"
	"net/http"
	"net/url"
	"strings"
)

type SearchResult struct {
	Results []struct {
		Id   string `json:"id"`
		Text string `json:"text"`
	} `json:"results"`
}

type ConstProxy struct {
	proxyUrl string
}

func NewConstituencyProxy(proxyUrl string) *ConstProxy {
	return &ConstProxy{proxyUrl: proxyUrl}
}

func (proxy *ConstProxy) ConstituencySearch(w http.ResponseWriter, req *http.Request) {
	zipcode := req.PathValue("zipcode")

	query, err := url.Parse(fmt.Sprintf("%s?term=%s&_type=query&q=%s", proxy.proxyUrl, zipcode, zipcode))
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	data, err := myHttp.FetchUrlAsBrowser(query)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	var result SearchResult
	err = json.Unmarshal(data, &result)
	if err != nil {
		fmt.Printf("Failed to unmarshal response: %v\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	var constituencies []v1.Constituency
	for _, r := range result.Results {
		parts := strings.Split(r.Text, " - ")
		idParts := strings.Split(r.Id, "*~*")
		constituencies = append(constituencies, v1.Constituency{Number: idParts[0], Name: strings.Trim(strings.Join(parts, ","), " ")})
	}

	if err = rest.MarshalResponse(w, constituencies); err != nil {
		http.Error(w, "Failed to marshal response", http.StatusInternalServerError)
		return
	}

}
