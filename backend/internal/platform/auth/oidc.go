package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"

	"github.com/yurythx/projeto-nova/internal/platform/config"
)

// discoveryTimeout limita quanto tempo NewVerifier espera pelo documento
// de discovery OIDC. Sem um teto, um issuer configurado errado (DNS que
// nunca resolve, uma rota que faz o pacote cair no buraco negro) faria o
// processo travar indefinidamente no startup, em vez de "falhar rápido"
// como o restante deste construtor promete.
const discoveryTimeout = 10 * time.Second

// Verifier valida access tokens contra o realm do Keycloak configurado
// e, opcionalmente, contra tokens locais assinados pelo próprio backend
// (§ Sistema de Login Local — sempre um caminho ADICIONAL, nunca um
// substituto do Keycloak). Faz o discovery OIDC uma única vez no startup
// e mantém o JWKS em cache no próprio processo (só é atualizado quando
// aparece um key id desconhecido, conforme a implementação de remote key
// set do go-oidc) — sem nenhuma chamada ao Keycloak por requisição (§29).
type Verifier struct {
	idTokenVerifier *oidc.IDTokenVerifier
	clientID        string
	localSigner     *LocalSigner
}

// NewVerifier faz o discovery OIDC contra cfg.IssuerURL. Falha rápido
// (retorna um erro) se o issuer estiver inalcançável ou malformado, para
// que uma configuração errada apareça no startup em vez de na primeira
// requisição que precisar validar um token. localSigner pode ser nil —
// nesse caso Verify nunca tenta o caminho local, só Keycloak, exatamente
// como antes deste recurso existir. Note que localSigner é *outra* chave,
// independente de qualquer coisa relacionada a cfg (Keycloak): os dois
// caminhos de autenticação nunca compartilham material criptográfico.
func NewVerifier(ctx context.Context, cfg config.KeycloakConfig, localSigner *LocalSigner) (*Verifier, error) {
	if cfg.IssuerURL == "" {
		if localSigner == nil {
			return nil, fmt.Errorf("auth: Keycloak IssuerURL is empty and LocalAuth is disabled")
		}
		return &Verifier{
			idTokenVerifier: nil,
			clientID:        cfg.ClientID,
			localSigner:     localSigner,
		}, nil
	}

	discoveryCtx, cancel := context.WithTimeout(ctx, discoveryTimeout)
	defer cancel()

	provider, err := oidc.NewProvider(discoveryCtx, cfg.IssuerURL)
	if err != nil {
		return nil, fmt.Errorf("auth: OIDC discovery failed for issuer %q: %w", cfg.IssuerURL, err)
	}

	verifier := provider.VerifierContext(ctx, &oidc.Config{
		ClientID:             cfg.Audience,
		SupportedSigningAlgs: []string{oidc.RS256, oidc.RS384, oidc.RS512, oidc.ES256},
	})

	return &Verifier{
		idTokenVerifier: verifier,
		clientID:        cfg.ClientID,
		localSigner:     localSigner,
	}, nil
}

// Verify valida a assinatura, o issuer, a audiência, a expiração e o
// algoritmo de rawToken, e então extrai a Identity da plataforma a partir
// das suas claims. Nunca chama o Keycloak — a verificação é inteiramente
// local, contra o JWKS em cache.
//
// Tenta primeiro o Keycloak (o caminho principal e sempre ativo); se isso
// falhar e o login local estiver habilitado, tenta a verificação RS256
// local (com a chave própria de LocalSigner) antes de desistir. Um token
// de verdade do Keycloak nunca passa na verificação local por acidente, e
// vice-versa — as duas chaves RSA são inteiramente independentes (uma
// pertence ao realm do Keycloak, a outra só existe neste processo), então
// não há ambiguidade possível sobre qual token "pertence" a qual caminho.
func (v *Verifier) Verify(ctx context.Context, rawToken string) (Identity, error) {
	var keycloakErr error
	if v.idTokenVerifier != nil {
		token, err := v.idTokenVerifier.Verify(ctx, rawToken)
		if err == nil {
			var claims accessTokenClaims
			if err := token.Claims(&claims); err != nil {
				return Identity{}, fmt.Errorf("auth: decode token claims: %w", err)
			}
			return claims.toIdentity(v.clientID), nil
		}
		keycloakErr = err
	}

	if v.localSigner == nil {
		if keycloakErr != nil {
			return Identity{}, fmt.Errorf("auth: token verification failed: %w", keycloakErr)
		}
		return Identity{}, fmt.Errorf("auth: no token verifiers available")
	}

	identity, localErr := v.localSigner.verifyToken(rawToken)
	if localErr != nil {
		if keycloakErr != nil {
			return Identity{}, fmt.Errorf("auth: token verification failed (keycloak: %v, local: %w)", keycloakErr, localErr)
		}
		return Identity{}, fmt.Errorf("auth: local token verification failed: %w", localErr)
	}
	return identity, nil
}
