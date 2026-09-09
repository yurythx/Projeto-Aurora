package app

import (
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	exampleTransport "github.com/yurythx/projeto-aurora/internal/modules/example/transport"
	integrationsTransport "github.com/yurythx/projeto-aurora/internal/modules/integrations/transport"
	usersTransport "github.com/yurythx/projeto-aurora/internal/modules/users/transport"

	"github.com/yurythx/projeto-aurora/internal/platform/auth"
	"github.com/yurythx/projeto-aurora/internal/platform/configflags"
	"github.com/yurythx/projeto-aurora/internal/platform/database"
	"github.com/yurythx/projeto-aurora/internal/platform/httpserver"
	"github.com/yurythx/projeto-aurora/internal/platform/idempotency"
	"github.com/yurythx/projeto-aurora/internal/platform/keycloakconfig"
	"github.com/yurythx/projeto-aurora/internal/platform/lgpd"
	"github.com/yurythx/projeto-aurora/internal/platform/localauth"
	"github.com/yurythx/projeto-aurora/internal/platform/outbox"
	"github.com/yurythx/projeto-aurora/internal/platform/transparency"
	"github.com/yurythx/projeto-aurora/internal/platform/ws"
)

// NewRouter monta o router HTTP completo para o Projeto Aurora: a base da plataforma
// (health, metrics, request id, recovery, CORS, security headers), o /ready
// conectado a toda dependência essencial, e as rotas de negócio versionadas em /api/v1 e /ws.
func NewRouter(deps *Dependencies) chi.Router {
	r := httpserver.New(httpserver.Options{
		Logger:         deps.Logger,
		AllowedOrigins: []string{deps.Config.FrontendURL},
		RequestTimeout: 30 * time.Second,
		MetricsToken:   deps.Config.Security.MetricsToken,
	})

	checks := []httpserver.Check{
		{Name: "postgres", Fn: database.Ping(deps.DB)},
		{Name: "rabbitmq", Fn: deps.Messaging.Ping},
		// MinIO — achado de auditoria: o card correspondente no painel de
		// Monitoramento sempre mostrava "Desconhecido", porque nada nunca
		// perguntava ao MinIO se ele estava de pé (só postgres/rabbitmq
		// eram checados aqui). deps.Storage.Ping reusa o client MinIO já
		// autenticado — sem ele, esta checagem só existiria se o MinIO
		// estivesse fora do ar E algo tentasse fazer upload/download na
		// hora, tarde demais para um painel de monitoramento.
		{Name: "minio", Fn: deps.Storage.Ping},
	}
	readiness := httpserver.ReadyHandler(checks, 3*time.Second)
	r.Get("/ready", readiness)
	r.Get("/readyz", readiness) // alias canônico k8s (gap G-14)

	// WebSocket autenticado por ticket
	r.Get("/ws", ws.UpgradeHandler(deps.Hub, deps.Tickets, deps.Config.FrontendURL, deps.Logger))

	// Rota pública de documentação de API (OpenAPI 3.0 / Swagger UI - e-PING)
	r.Get("/openapi.json", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "docs/openapi.json")
	})
	// Assets do Swagger UI vendorados (F2.1): swagger-ui-dist@5.17.14 +
	// swagger-initializer.js próprio — sem CDN, então a CSP de /docs pode
	// ficar em 'self' (só style-src precisa de 'unsafe-inline', porque o
	// Swagger UI injeta <style> em runtime).
	r.Get("/docs/assets/*", func(w http.ResponseWriter, r *http.Request) {
		asset := chi.URLParam(r, "*")
		if strings.Contains(asset, "..") { // defesa contra path traversal
			http.NotFound(w, r)
			return
		}
		http.ServeFile(w, r, "docs/swagger-ui/"+asset)
	})
	r.Get("/docs", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Content-Security-Policy",
			"default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; "+
				"img-src 'self' data:; connect-src 'self'; frame-ancestors 'none'; base-uri 'self'")
		w.Write([]byte(`<!DOCTYPE html>
<html lang="pt-BR">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Documentação da API — Projeto Aurora (OpenAPI 3.0)</title>
  <link rel="stylesheet" href="/docs/assets/swagger-ui.css" />
</head>
<body>
  <div id="swagger-ui"></div>
  <script src="/docs/assets/swagger-ui-bundle.js"></script>
  <script src="/docs/assets/swagger-initializer.js"></script>
</body>
</html>`))
	})

	// Rota de login local (pública)
	localauth.RegisterRoutes(r, deps.Modules.LocalAuth.Handlers, deps.Logger, deps.RateLimiters.LocalLogin)

	// Consentimento LGPD de visitante NÃO autenticado (gap G-11) — rota
	// pública, rate-limited por IP.
	lgpd.RegisterPublicRoutes(r, deps.LGPDSvc, deps.RateLimiters.AnonConsent)

	// Transparência ativa / dados abertos (LAI art. 8º, LC 131 — F3.5) —
	// rotas públicas em /api/v1/transparencia/*, rate-limited por IP.
	transparency.RegisterRoutes(r, deps.Transparency, deps.RateLimiters.PublicRead)

	// Rotas protegidas (/api/v1)
	r.Route("/api/v1", func(api chi.Router) {
		api.Use(auth.RequireAuthentication(deps.Verifier, deps.Logger))
		// Rate limit por identidade autenticada (fallback: IP) em TODO o
		// grupo /api/v1 — gap G-01: antes só login e /ws/ticket tinham
		// teto, qualquer token válido podia marretar as demais rotas.
		api.Use(httpserver.RateLimit(deps.Logger, deps.RateLimiters.APIGlobal, apiRateLimitKey))
		api.Use(idempotency.Middleware(deps.Idempotency, deps.Logger))

		api.With(httpserver.RateLimit(deps.Logger, deps.RateLimiters.WSTicket, wsTicketRateLimitKey)).
			Post("/ws/ticket", ws.TicketHandler(deps.Tickets, deps.Logger))

		// POST /api/v1/auth/logout — precisa de identidade, então mora
		// aqui dentro (o login fica fora, em localauth.RegisterRoutes).
		localauth.RegisterAuthedRoutes(api, deps.Modules.LocalAuth.Handlers)

		usersTransport.RegisterRoutes(api, deps.Modules.Users.Handlers, deps.Logger)
		integrationsTransport.RegisterRoutes(api, deps.Modules.Integrations.Handlers, deps.Logger)
		configflags.RegisterRoutes(api, deps.Modules.ConfigFlags.Handlers, deps.Logger)
		keycloakconfig.RegisterRoutes(api, deps.Modules.KeycloakConfig.Handlers, deps.Logger)
		outbox.RegisterStatsRoutes(api, deps.Modules.OutboxStats.Handlers, deps.Logger)
		exampleTransport.RegisterRoutes(api, deps.Modules.Example.Handlers)
		deps.LGPDSvc.RegisterRoutes(api)
		deps.LGPDSvc.RegisterDSRRoutes(api) // direitos do titular — LGPD art. 18 (F3.2)
		deps.AuditExp.RegisterRoutes(api)
	})

	return r
}

func wsTicketRateLimitKey(r *http.Request) string {
	if identity, ok := auth.IdentityFromContext(r.Context()); ok && identity.Subject != "" {
		return identity.Subject
	}
	return httpserver.ClientIPKey(r)
}

// apiRateLimitKey identifica o chamador do rate limiter global de
// /api/v1: o subject do token autenticado quando disponível (o normal,
// já que o grupo está atrás de RequireAuthentication), caindo para o IP
// só em caminhos de borda. Mesma forma de wsTicketRateLimitKey — nome
// próprio só para documentar a intenção no ponto de uso.
func apiRateLimitKey(r *http.Request) string {
	if identity, ok := auth.IdentityFromContext(r.Context()); ok && identity.Subject != "" {
		return identity.Subject
	}
	return httpserver.ClientIPKey(r)
}
