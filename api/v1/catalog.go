package v1

import "net/url"

type PersonCatalog struct {
	DocumentInfo  DocumentInfo      `json:"documentInfo" xml:"dokumentInfo"`
	DeleteRestore DeleteRestore     `json:"deleteRestore" xml:"deleteRestore"`
	Persons       []PersonListEntry `json:"persons" xml:"mdbs>mdb"`
}

func (c PersonCatalog) GetItems() []PersonListEntry {
	return c.Persons
}

type DeleteRestore struct {
	DeleteFlag string `json:"deleteFlag" xml:"deleteFlag"`
	DeleteDate string `json:"deleteDate" xml:"deleteDate"`
}

type PersonListEntry struct {
	Faction              string       `json:"faction" xml:"fraktion,attr"`
	Id                   ID           `json:"id" xml:"mdbID"`
	Name                 MdbName      `json:"name" xml:"mdbName"`
	BioURL               string       `json:"bioUrl" xml:"mdbBioURL"`
	InfoXMLURL           string       `json:"infoXmlUrl" xml:"mdbInfoXMLURL"`
	InfoXMLURLMitmischen string       `json:"infoXmlUrlMitmischen,omitempty" xml:"mdbInfoXMLURLMitmischen"`
	State                string       `json:"state" xml:"mdbLand"`
	Constituency         Constituency `json:"constituency,omitempty" xml:"mdbWahlkreis"`
	Elected              string       `json:"elected,omitempty" xml:"mdbGewaehlt"`
	PhotoURL             string       `json:"photoUrl" xml:"mdbFotoURL"`
	PhotoLargeURL        string       `json:"photoLargeUrl" xml:"mdbFotoGrossURL"`
	PhotoLastChanged     string       `json:"photoLastChanged" xml:"mdbFotoLastChanged"`
	PhotoChangedDateTime string       `json:"photoChangedDateTime" xml:"mdbFotoChangedDateTime"`
	ImageAltText         string       `json:"imageAltText" xml:"imageAltText"`
	LastChanged          string       `json:"lastChanged" xml:"lastChanged"`
	ChangedDateTime      string       `json:"changedDateTime" xml:"changedDateTime"`
}

func (c PersonListEntry) GetId() string {
	return c.Id.Value
}

func (c PersonListEntry) GetDetailUrl() *url.URL {
	dUrl, _ := url.Parse(c.InfoXMLURL)
	return dUrl
}

type CommitteeCatalog struct {
	DocumentInfo DocumentInfo         `xml:"dokumentInfo" json:"dokumentInfo"`
	Committees   []CommitteeListEntry `xml:"ausschuesse>ausschuss" json:"committees"`
}

func (c CommitteeCatalog) GetItems() []CommitteeListEntry {
	return c.Committees
}

type CommitteeListEntry struct {
	Id                   string `xml:"id,attr" json:"id"`
	Live                 int    `xml:"live" json:"live"`
	Name                 string `xml:"ausschussName" json:"committeeName"`
	ShortName            string `xml:"ausschussKurzName" json:"committeeShortName"`
	Teaser               string `xml:"ausschussTeaser" json:"committeeTeaser"`
	LastChanged          string `xml:"lastChanged" json:"lastChanged"`
	ChangedDateTime      string `xml:"changedDateTime" json:"changedDateTime"`
	DetailXML            string `xml:"ausschussDetailXML" json:"committeeDetailXml"`
	ImageURL             string `xml:"imageURL" json:"imageUrl"`
	ImageGrossURL        string `xml:"imageGrossURL" json:"imageGrossUrl"`
	ImageXL              string `xml:"imageXL" json:"imageXL"`
	ImageXXL             string `xml:"imageXXL" json:"imageXXL"`
	ImageCopyright       string `xml:"imageCopyright" json:"imageCopyright"`
	ImageLastChanged     string `xml:"imageLastChanged" json:"imageLastChanged"`
	ImageChangedDateTime string `xml:"imageChangedDateTime" json:"imageChangedDateTime"`
	ImageAltText         string `xml:"imageAltText" json:"imageAltText"`
}

func (c CommitteeListEntry) GetId() string {
	return c.Id
}

func (c CommitteeListEntry) GetDetailUrl() *url.URL {
	dUrl, _ := url.Parse(c.DetailXML)
	return dUrl
}
