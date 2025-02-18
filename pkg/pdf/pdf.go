package pdf

import (
	"bytes"
	"fmt"
	"github.com/go-pdf/fpdf"
	"text/template"
)

type LetterData struct {
	SenderName       string
	SenderAddress    string
	RecipientName    string
	RecipientAddress string
	Salutation       string
	Party            string
}

const subject = "Dringender Handlungsbedarf: Reform des Statusfeststellungsverfahrens"

func Generate(data LetterData, staticFolder string) ([]byte, error) {

	tpl, err := template.ParseFiles(fmt.Sprintf("%s/letter.tpl.txt", staticFolder))
	if err != nil {
		return nil, err
	}
	var buffer bytes.Buffer
	err = tpl.Execute(&buffer, data)

	// Create a new PDF document.
	pdf := fpdf.New("P", "mm", "A4", "")

	// Add a UTF-8 compatible font that supports umlauts.
	// Ensure "DejaVuSans.ttf" is available in your working directory or provide the correct path.
	pdf.AddUTF8Font("DejaVu", "", fmt.Sprintf("%s/fonts/DejaVuSans.ttf", staticFolder))
	pdf.AddUTF8Font("DejaVu", "B", fmt.Sprintf("%s/fonts/DejaVuSans-Bold.ttf", staticFolder))
	pdf.AddUTF8Font("DejaVu", "L", fmt.Sprintf("%s/fonts/DejaVuSans-ExtraLight.ttf", staticFolder))

	// Set the font to the added UTF-8 font.
	pdf.SetFont("DejaVu", "L", 12)
	pdf.AddPage()
	// Get page dimensions and margins.
	leftMargin, topMargin, rightMargin, _ := pdf.GetMargins()
	pageWidth, _ := pdf.GetPageSize()
	usableWidth := pageWidth - leftMargin - rightMargin

	//
	// 1. "Briefkopf für Sichtfenster"
	//
	pdf.SetFont("DejaVu", "", 8)
	pdf.SetXY(leftMargin, topMargin)
	pdf.CellFormat(usableWidth, 5, "Briefkopf für Sichtfenster", "", 1, "L", false, 0, "")
	pdf.Ln(3)

	//
	// 2. Sender (Left) and Recipient (Right)
	//
	pdf.SetFont("DejaVu", "L", 12)

	// Recipient block on the left
	recipientBlock := data.RecipientName + "\n" + data.RecipientAddress
	pdf.MultiCell(usableWidth, 5, recipientBlock, "", "L", false)
	pdf.Ln(3)

	// Sender block on the right
	senderBlock := data.SenderName + "\n" + data.SenderAddress
	pdf.MultiCell(usableWidth, 5, senderBlock, "", "R", false)

	pdf.Ln(25) // Space before the subject
	// ---------------------------
	// 4. Subject Block
	// ---------------------------
	pdf.SetFont("DejaVu", "B", 12)
	// Use MultiCell instead of CellFormat to enable line wrapping.
	pdf.MultiCell(usableWidth, 5, "Betreff: "+subject, "", "L", false)
	pdf.Ln(4)

	// ---------------------------
	// 5. Salutation
	// ---------------------------
	pdf.SetFont("DejaVu", "", 12)
	pdf.CellFormat(usableWidth, 5, data.Salutation, "", 1, "L", false, 0, "")
	pdf.Ln(4)

	// ---------------------------
	// 6. Body Text
	// ---------------------------
	pdf.MultiCell(usableWidth, 5, buffer.String(), "", "L", false)
	pdf.Ln(8)

	// ---------------------------
	// 7. Closing & Signature
	// ---------------------------
	pdf.CellFormat(usableWidth, 5, data.SenderName, "", 1, "L", false, 0, "")

	// Output the PDF to a buffer.
	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
