package pdf

import (
	"bytes"
	"fmt"
	"github.com/SebastiaanKlippert/go-wkhtmltopdf"
	"html/template"
	"sync"
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

// Define a global or package-level buffer pool.
var bufferPool = sync.Pool{
	New: func() interface{} {
		return new(bytes.Buffer)
	},
}

var pdfGenPool = sync.Pool{
	New: func() interface{} {
		pdfg, err := wkhtmltopdf.NewPDFGenerator()
		if err != nil {
			return nil // or handle the error appropriately
		}
		return pdfg
	},
}

func Generate(data LetterData, tpl *template.Template) ([]byte, error) {

	// Get a buffer from the pool and ensure it's reset.
	htmlBuffer := bufferPool.Get().(*bytes.Buffer)
	htmlBuffer.Reset()
	defer bufferPool.Put(htmlBuffer) // return the buffer to the pool after use

	// Execute the template into the reused buffer.
	if err := tpl.Execute(htmlBuffer, data); err != nil {
		return nil, fmt.Errorf("failed to execute template: %w", err)
	}

	// Create new PDF generator.
	pdfg := pdfGenPool.Get().(*wkhtmltopdf.PDFGenerator)
	defer pdfGenPool.Put(pdfg)

	// Set global options.
	pdfg.Dpi.Set(300)
	pdfg.Orientation.Set(wkhtmltopdf.OrientationPortrait)
	pdfg.PageSize.Set(wkhtmltopdf.PageSizeA4)

	// Create a new input page from the HTML in the buffer.
	page := wkhtmltopdf.NewPageReader(bytes.NewReader(htmlBuffer.Bytes()))

	// Set options for this page.
	page.FooterRight.Set("powered by www.stoppt-scheinselbstaendigkeit.de - [page]")
	page.FooterFontSize.Set(10)
	page.Zoom.Set(1)

	// Add the page to the PDF generator.
	pdfg.AddPage(page)

	// Generate the PDF document.
	if err := pdfg.Create(); err != nil {
		return nil, fmt.Errorf("failed to create PDF: %w", err)
	}

	// Return the generated PDF bytes.
	return pdfg.Buffer().Bytes(), nil
}
