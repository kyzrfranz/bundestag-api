package rest

import (
	"encoding/json"
	"fmt"
	v1 "github.com/kyzrfranz/buntesdach/api/v1"
	myHttp "github.com/kyzrfranz/buntesdach/internal/http"
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

const baseUrl = "https://www.bundestag.de/ajax/filterlist/de/533302-533302/plz-ort-autocomplete"

func Find(w http.ResponseWriter, req *http.Request) {
	zipcode := req.PathValue("zipcode")

	query, err := url.Parse(fmt.Sprintf("%s?term=%s&_type=query&q=%s", baseUrl, zipcode, zipcode))
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

	if err = marshalResponse(w, constituencies); err != nil {
		http.Error(w, "Failed to marshal response", http.StatusInternalServerError)
		return
	}

}
