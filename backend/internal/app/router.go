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
	"github.com/yurythx/projeto-aurora/internal/platform/localauth"
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
	})

	checks := []httpserver.Check{
		{Name: "postgres", Fn: database.Ping(deps.DB)},
		{Name: "rabbitmq", Fn: deps.Messaging.Ping},
	}
	r.Get("/ready", httpserver.ReadyHandler(checks, 3*time.Second))

	// WebSocket autenticado por ticket
	r.Get("/ws", ws.UpgradeHandler(deps.Hub, deps.Tickets, deps.Config.FrontendURL, deps.Logger))

	// Rota de login local (pública)
	localauth.RegisterRoutes(r, deps.Modules.LocalAuth.Handlers, deps.Logger, deps.RateLimiters.LocalLogin)

	// Rotas protegidas (/api/v1)
	r.Route("/api/v1", func(api chi.Router) {
		api.Use(auth.RequireAuthentication(deps.Verifier, deps.Logger))
		api.Use(idempotency.Middleware(deps.Idempotency, deps.Logger))

		api.With(httpserver.RateLimit(deps.Logger, deps.RateLimiters.WSTicket, wsTicketRateLimitKey)).
			Post("/ws/ticket", ws.TicketHandler(deps.Tickets, deps.Logger))

		usersTransport.RegisterRoutes(api, deps.Modules.Users.Handlers, deps.Logger)
		integrationsTransport.RegisterRoutes(api, deps.Modules.Integrations.Handlers)
		configflags.RegisterRoutes(api, deps.Modules.ConfigFlags.Handlers, deps.Logger)
		exampleTransport.RegisterRoutes(api, deps.Modules.Example.Handlers)
	})

	return r
}

func wsTicketRateLimitKey(r *http.Request) string {
	if identity, ok := auth.IdentityFromContext(r.Context()); ok && identity.Subject != "" {
		return identity.Subject
	}
	return httpserver.ClientIPKey(r)
}
