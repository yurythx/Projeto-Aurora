package application

import (
	"testing"
	"time"
)

func TestExtractHREvents_Exoneracao(t *testing.T) {
	content := `SECRETARIA MUNICIPAL DE GOVERNO
PORTARIA Nº 41.735, DE 22 DE JULHO DE 2026.
RESOLVE: Art. 1º Exonerar a pedido, VANETE BARBOSA DO REGO, do cargo em comissão de Agente Administrativo da Familia.`

	pubDate := time.Now()
	events := ExtractHREvents(content, "6262", "http://example.com/pdf.pdf", pubDate)

	if len(events) == 0 {
		t.Fatalf("Expected HR events, got 0")
	}

	foundExoneracao := false
	for _, ev := range events {
		if ev.Type == HREventExoneracao {
			foundExoneracao = true
			if ev.Servidor != "Vanete Barbosa do Rego" {
				t.Errorf("Servidor = %q, want Vanete Barbosa do Rego", ev.Servidor)
			}
			if ev.PortariaNumber != "41.735" {
				t.Errorf("PortariaNumber = %q, want 41.735", ev.PortariaNumber)
			}
		}
	}

	if !foundExoneracao {
		t.Errorf("Did not extract Exoneração event")
	}
}

func TestExtractHREvents_Nomeacao(t *testing.T) {
	content := `PORTARIA Nº 42.100
RESOLVE: Art. 1º Nomear CARLOS EDUARDO ALMEIDA, para o cargo em comissão de Assessor Técnico.`

	pubDate := time.Now()
	events := ExtractHREvents(content, "6263", "http://example.com/pdf.pdf", pubDate)

	foundNomeacao := false
	for _, ev := range events {
		if ev.Type == HREventNomeacao {
			foundNomeacao = true
			if ev.Servidor != "Carlos Eduardo Almeida" {
				t.Errorf("Servidor = %q, want Carlos Eduardo Almeida", ev.Servidor)
			}
		}
	}

	if !foundNomeacao {
		t.Errorf("Did not extract Nomeação event")
	}
}
