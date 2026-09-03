package auth

// accessTokenClaims espelha o subconjunto das claims de um access token do
// Keycloak que a plataforma se importa. O Keycloak coloca os roles de
// realm (globais) em "realm_access.roles" e os roles por client em
// "resource_access.<client_id>.roles" — os dois são mesclados em
// Identity.Roles, já que o código de autorização (rbac.go) não precisa
// distinguir a origem do role, só se o usuário o possui.
import "strings"

type accessTokenClaims struct {
	Subject              string                   `json:"sub"`
	PreferredUsername    string                   `json:"preferred_username"`
	Email                string                   `json:"email"`
	GovBRConfiabilidade  string                   `json:"govbr_confiabilidade"`
	RealmAccess          roleContainer            `json:"realm_access"`
	ResourceAccess       map[string]roleContainer `json:"resource_access"`
}

type roleContainer struct {
	Roles []string `json:"roles"`
}

// parseGovBRLevel normaliza a string do claim de confiabilidade do Gov.br.
func parseGovBRLevel(levelStr string) GovBRLevel {
	switch strings.ToUpper(strings.TrimSpace(levelStr)) {
	case "BRONZE", "1":
		return GovBRLevelBronze
	case "PRATA", "2":
		return GovBRLevelPrata
	case "OURO", "3":
		return GovBRLevelOuro
	default:
		return GovBRLevelUnknown
	}
}

// toIdentity mescla os roles de realm com os roles concedidos para
// clientID numa única Identity, sem duplicatas.
func (c accessTokenClaims) toIdentity(clientID string) Identity {
	seen := make(map[string]struct{}, len(c.RealmAccess.Roles))
	roles := make([]string, 0, len(c.RealmAccess.Roles))

	add := func(rs []string) {
		for _, r := range rs {
			if _, ok := seen[r]; ok {
				continue
			}
			seen[r] = struct{}{}
			roles = append(roles, r)
		}
	}

	add(c.RealmAccess.Roles)
	if clientRoles, ok := c.ResourceAccess[clientID]; ok {
		add(clientRoles.Roles)
	}

	return Identity{
		Subject:    c.Subject,
		Username:   c.PreferredUsername,
		Email:      c.Email,
		Roles:      roles,
		GovBRLevel: parseGovBRLevel(c.GovBRConfiabilidade),
		Source:     SourceKeycloak,
	}
}
