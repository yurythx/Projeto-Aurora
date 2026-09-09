package keycloakconfig

import (
	"testing"

	"github.com/yurythx/projeto-aurora/internal/platform/config"
)

func TestSettings_ToKeycloakConfig(t *testing.T) {
	s := Settings{
		IssuerURL:        "https://sso.orgao.gov.br/realms/aurora",
		Realm:            "aurora",
		ClientID:         "aurora-backend",
		ClientSecret:     "segredo",
		Audience:         "aurora-backend",
		FrontendClientID: "aurora-frontend", // não faz parte de config.KeycloakConfig — deve ser ignorado
	}

	want := config.KeycloakConfig{
		IssuerURL:    "https://sso.orgao.gov.br/realms/aurora",
		Realm:        "aurora",
		ClientID:     "aurora-backend",
		ClientSecret: "segredo",
		Audience:     "aurora-backend",
	}

	if got := s.ToKeycloakConfig(); got != want {
		t.Errorf("ToKeycloakConfig() = %+v, want %+v", got, want)
	}
}
