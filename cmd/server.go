package main

import (
	v1 "github.com/kyzrfranz/buntesdach/api/v1"
	"github.com/kyzrfranz/buntesdach/internal/data"
	"github.com/kyzrfranz/buntesdach/internal/http"
	"github.com/kyzrfranz/buntesdach/internal/proxy"
	"github.com/kyzrfranz/buntesdach/internal/rest"
	"github.com/kyzrfranz/buntesdach/internal/upstream"
	"github.com/kyzrfranz/buntesdach/pkg/resources"
	"log/slog"
	"net/url"
	"os"
)

var (
	logger              *slog.Logger
	constSearchProxyUrl string
)

func main() {

	logger = slog.New(slog.NewJSONHandler(os.Stdout, nil))
	constSearchProxyUrl = stringOrEnv("CONSTITUENCY_PROXY_URL", "https://www.bundestag.de/ajax/filterlist/de/533302-533302/plz-ort-autocomplete")

	dataUrl := mustGetUrl("https://www.bundestag.de/xml/v2/mdb/index.xml") // TODO config
	politicianReader := data.NewCatalogReader[v1.PersonCatalog, v1.PersonListEntry](&upstream.XMLFetcher{Url: dataUrl})

	committeeUrl := mustGetUrl("https://www.bundestag.de/xml/v2/ausschuesse/index.xml") // TODO config
	committeeReader := data.NewCatalogReader[v1.CommitteeCatalog, v1.CommitteeListEntry](&upstream.XMLFetcher{Url: committeeUrl})

	apiServer := http.NewApiServer(8080, logger)

	apiServer.Use(http.MiddlewareRecovery)
	apiServer.Use(http.MiddlewareCORS)

	politicianCatalogHandler := rest.NewHandler[v1.PersonListEntry](resources.NewCatalogueRepo[v1.PersonListEntry](&politicianReader))
	politicianDetailHandler := rest.NewHandler[v1.Politician](resources.NewDetailRepo[v1.Politician](&politicianReader))
	committeeCatalogueHandler := rest.NewHandler[v1.CommitteeListEntry](resources.NewCatalogueRepo[v1.CommitteeListEntry](&committeeReader))
	committeeDetailHandler := rest.NewHandler[v1.CommitteeDetails](resources.NewDetailRepo[v1.CommitteeDetails](&committeeReader))

	apiServer.AddHandler("/politicians", politicianCatalogHandler.List)
	apiServer.AddHandler("/politicians/{id}", politicianCatalogHandler.Get)
	apiServer.AddHandler("/politicians/{id}/bio", politicianDetailHandler.Get)
	apiServer.AddHandler("/committees", committeeCatalogueHandler.List)
	apiServer.AddHandler("/committees/{id}", committeeCatalogueHandler.Get)
	apiServer.AddHandler("/committees/{id}/detail", committeeDetailHandler.Get)

	//proxy for zipcode search
	cProxy := proxy.NewConstituencyProxy(constSearchProxyUrl)
	apiServer.AddHandler("/constituencies/{zipcode}", cProxy.ConstituencySearch)

	apiServer.ListenAndServe()
}

func bail(stage string, err error) {
	logger.Error("server bailing out", slog.String("stage", stage), "error", err)
	os.Exit(1)
}

func mustGetUrl(s string) *url.URL {
	parsedUrl, err := url.Parse(s)
	if err != nil {
		bail("parse data url", err)
	}

	return parsedUrl
}

func stringOrEnv(key string, defaultVal string) (s string) {
	s = os.Getenv(key)
	if s != "" {
		defaultVal = s
	}

	return defaultVal
}
