package app

import (
	"net/http"
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
	"github.com/yurythx/projeto-aurora/internal/platform/localauth"
	"github.com/yurythx/projeto-aurora/internal/platform/outbox"
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
	r.Get("/docs", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		// Sobrescreve a CSP restritiva de SecurityHeaders (gap G-03) só
		// nesta página HTML — o Swagger UI é carregado do CDN oficial e
		// usa um <script> inline de bootstrap. TODO (F2.1): servir o
		// Swagger UI localmente e voltar à CSP padrão.
		w.Header().Set("Content-Security-Policy",
			"default-src 'self'; script-src 'self' 'unsafe-inline' https://cdn.jsdelivr.net; "+
				"style-src 'self' 'unsafe-inline' https://cdn.jsdelivr.net; img-src 'self' data:; "+
				"connect-src 'self'; frame-ancestors 'none'; base-uri 'self'")
		w.Write([]byte(`<!DOCTYPE html>
<html lang="pt-BR">
<head>
  <meta charset="UTF-8">
  <title>Documentação da API — Projeto Aurora (OpenAPI 3.0)</title>
  <link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/swagger-ui-dist@5/swagger-ui.css" />
</head>
<body>
  <div id="swagger-ui"></div>
  <script src="https://cdn.jsdelivr.net/npm/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
  <script>
    window.onload = () => {
      SwaggerUIBundle({
        url: '/openapi.json',
        dom_id: '#swagger-ui',
      });
    };
  </script>
</body>
</html>`))
	})

	// Rota de login local (pública)
	localauth.RegisterRoutes(r, deps.Modules.LocalAuth.Handlers, deps.Logger, deps.RateLimiters.LocalLogin)

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
