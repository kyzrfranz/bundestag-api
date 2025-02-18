package util

import (
	"fmt"
	v1 "github.com/kyzrfranz/buntesdach/api/v1"
	"strings"
)

func LongSalutation(bio v1.PoliticianBio) string {
	// return a correctly gendered salutation including AcademitTitle and or Nobility Title if set
	genderedSalutation := "Hallo"
	appellation := bio.LastName
	if strings.ToLower(bio.Gender) == "männlich" {
		genderedSalutation = "Sehr geehrter Herr"
	} else if strings.ToLower(bio.Gender) == "weiblich" {
		genderedSalutation = "Sehr geehrte Frau"
	} else {
		genderedSalutation = "Hallo"
		appellation = fmt.Sprintf("%s %s", bio.FirstName, bio.LastName)
	}
	academicTitle := ""
	if bio.AcademicTitle != "" {
		academicTitle = bio.AcademicTitle + " "
	}

	nobilityTitle := ""
	if bio.NobilityTitle != "" {
		nobilityTitle = bio.NobilityTitle + " "
	}

	return fmt.Sprintf("%s %s%s%s", genderedSalutation, academicTitle, nobilityTitle, appellation)
}

func ShortSalutation(bio v1.PoliticianBio) string {
	academicTitle := ""
	if bio.AcademicTitle != "" {
		academicTitle = bio.AcademicTitle + " "
	}

	nobilityTitle := ""
	if bio.NobilityTitle != "" {
		nobilityTitle = bio.NobilityTitle + " "
	}

	return fmt.Sprintf("%s%s%s %s", academicTitle, nobilityTitle, bio.FirstName, bio.LastName)
}
