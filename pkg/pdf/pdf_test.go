package pdf

import (
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

func TestGenerate(t *testing.T) {

	data := LetterData{
		SenderName: "Christian Olearius",
		SenderAddress: Address{
			Street:  "Musterstraße 123",
			ZipCode: 12345,
			City:    "Musterstadt",
			Number:  1,
		},
		RecipientName: "Olaf Schulz",
		RecipientAddress: Address{
			Street:  "Platz der Republik",
			Label:   "Deutscher Bundestag",
			ZipCode: 11011,
			City:    "Berlin",
			Number:  1,
		},
		Salutation: "Sehr geehrter Herr Schulz,",
		Party:      "SPD",
	}

	pdf, err := Generate(data, "../../static")
	assert.NoError(t, err)

	os.WriteFile("test.pdf", pdf, 0644)
}
