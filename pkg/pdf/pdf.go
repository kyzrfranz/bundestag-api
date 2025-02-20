package pdf

import (
	"bytes"
	"fmt"
	"github.com/SebastiaanKlippert/go-wkhtmltopdf"
	"html/template"
	"log"
)

type LetterData struct {
	SenderName       string
	SenderAddress    Address
	RecipientName    string
	RecipientAddress Address
	Salutation       string
	Party            string
}

type Address struct {
	Label   string
	Street  string
	Number  int
	ZipCode int
	City    string
}

func Generate(data LetterData, staticFolder string) ([]byte, error) {

	tpl, err := template.ParseFiles(fmt.Sprintf("%s/letter.tpl.html", staticFolder))
	if err != nil {
		return nil, err
	}
	var htmlBuffer bytes.Buffer
	err = tpl.Execute(&htmlBuffer, data)

	// Create new PDF generator
	pdfg, err := wkhtmltopdf.NewPDFGenerator()
	if err != nil {
		log.Fatal(err)
	}

	// Set global options
	pdfg.Dpi.Set(300)
	pdfg.Orientation.Set(wkhtmltopdf.OrientationPortrait)
	pdfg.PageSize.Set(wkhtmltopdf.PageSizeA4)

	// Create a new input page from an URL
	page := wkhtmltopdf.NewPageReader(bytes.NewReader(htmlBuffer.Bytes()))

	// Set options for this page
	page.FooterRight.Set("powered by www.stoppt-scheinselbstaendigkeit.de - [page]")
	page.FooterFontSize.Set(10)
	page.Zoom.Set(1)

	// Add to document
	pdfg.AddPage(page)

	// Create PDF document in internal buffer
	err = pdfg.Create()
	if err != nil {
		log.Fatal(err)
	}

	return pdfg.Buffer().Bytes(), nil

}
