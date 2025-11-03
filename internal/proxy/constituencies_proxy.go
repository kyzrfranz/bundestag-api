package proxy

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	v1 "github.com/kyzrfranz/bundestag-api/api/v1"
	myHttp "github.com/kyzrfranz/bundestag-api/internal/http"
	"github.com/kyzrfranz/bundestag-api/internal/rest"
	"github.com/kyzrfranz/bundestag-api/pkg/resources"
	"github.com/samber/lo"
)

type SearchResult struct {
	Results []struct {
		Id   string `json:"id"`
		Text string `json:"text"`
	} `json:"results"`
}

type ConstProxy struct {
	proxyUrl string
	repo     resources.Repository[v1.PersonListEntry]
}

func NewConstituencyProxy(proxyUrl string, politiciansRepo resources.Repository[v1.PersonListEntry]) *ConstProxy {
	return &ConstProxy{proxyUrl: proxyUrl, repo: politiciansRepo}
}

func (proxy *ConstProxy) ConstituencySearch(w http.ResponseWriter, req *http.Request) {
	zipcode := req.PathValue("zipcode")
	constituencies, status := proxy.readConsituencies(zipcode)
	if status != http.StatusOK {
		http.Error(w, "Failed to read constituencies", status)
		return
	}

	if err := rest.MarshalResponse(w, constituencies); err != nil {
		http.Error(w, "Failed to marshal response", http.StatusInternalServerError)
		return
	}

}

func (proxy *ConstProxy) ConstituencyPoliticianSearch(w http.ResponseWriter, req *http.Request) {
	zipcode := req.PathValue("zipcode")
	constituencies, status := proxy.readConsituencies(zipcode)
	if status != http.StatusOK {
		http.Error(w, "Failed to read constituencies", status)
		return
	}

	politicians := proxy.repo.List(req.Context())
	result := lo.Filter(politicians, func(p v1.PersonListEntry, _ int) bool {
		//filter all p.Constituency.Number iin constituencies
		return lo.SomeBy(constituencies, func(c v1.Constituency) bool {
			return c.Number == p.Constituency.Number
		})
	})

	if err := rest.MarshalResponse(w, result); err != nil {
		http.Error(w, "Failed to marshal response", http.StatusInternalServerError)
		return
	}
}

func (proxy *ConstProxy) readConsituencies(zipcode string) ([]v1.Constituency, int) {
	query, err := url.Parse(fmt.Sprintf("%s?term=%s&_type=query&q=%s", proxy.proxyUrl, zipcode, zipcode))
	if err != nil {
		return nil, http.StatusInternalServerError
	}
	data, err := myHttp.FetchUrlAsBrowser(query)
	if err != nil {
		return nil, http.StatusNotFound
	}
	var result SearchResult
	err = json.Unmarshal(data, &result)
	if err != nil {
		fmt.Printf("Failed to unmarshal response: %v\n", err)
		return nil, http.StatusInternalServerError
	}

	var constituencies []v1.Constituency
	for _, r := range result.Results {
		parts := strings.Split(r.Text, " - ")
		idParts := strings.Split(r.Id, "*~*")
		constituencies = append(constituencies, v1.Constituency{Number: idParts[0], Name: strings.Trim(strings.Join(parts, ","), " ")})
	}

	return constituencies, http.StatusOK
}
