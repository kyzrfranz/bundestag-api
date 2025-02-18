package pdf

import (
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

func TestGenerate(t *testing.T) {

	data := LetterData{
		SenderName:       "Christian Olearius",
		SenderAddress:    "Mustermannstraße 123\n12345 Musterstadt",
		RecipientName:    "Olaf Schulz",
		RecipientAddress: "Deutscher Bundestag\nPlatz der Republik 1\n11011 Berlin",
		Salutation:       "Sehr geehrter Herr Schulz,",
		Party:            "SPD",
	}

	pdf, err := Generate(data, "../../static")
	assert.NoError(t, err)

	os.WriteFile("test.pdf", pdf, 0644)
}
