package app

import (
	exampleApp "github.com/yurythx/projeto-aurora/internal/modules/example/application"
	exampleInfra "github.com/yurythx/projeto-aurora/internal/modules/example/infrastructure"
	exampleTransport "github.com/yurythx/projeto-aurora/internal/modules/example/transport"

	integrationsApp "github.com/yurythx/projeto-aurora/internal/modules/integrations/application"
	integrationsInfra "github.com/yurythx/projeto-aurora/internal/modules/integrations/infrastructure"
	integrationsTransport "github.com/yurythx/projeto-aurora/internal/modules/integrations/transport"

	usersApp "github.com/yurythx/projeto-aurora/internal/modules/users/application"
	usersInfra "github.com/yurythx/projeto-aurora/internal/modules/users/infrastructure"
	usersTransport "github.com/yurythx/projeto-aurora/internal/modules/users/transport"

	"github.com/yurythx/projeto-aurora/internal/platform/audit"
	"github.com/yurythx/projeto-aurora/internal/platform/configflags"
	"github.com/yurythx/projeto-aurora/internal/platform/keycloakconfig"
	"github.com/yurythx/projeto-aurora/internal/platform/localauth"
)

// Modules guarda o serviço de aplicação e os handlers HTTP de todo módulo
// de negócio do Projeto Aurora. Servindo como o "ponto de montagem" central
// que conecta domain/application/infrastructure/transport de cada módulo.
type Modules struct {
	Users struct {
		Handlers *usersTransport.Handlers
	}
	Integrations struct {
		Service  *integrationsApp.Service
		Handlers *integrationsTransport.Handlers
	}
	ConfigFlags struct {
		Handlers *configflags.Handlers
	}
	KeycloakConfig struct {
		Handlers *keycloakconfig.Handlers
	}
	LocalAuth struct {
		Handlers *localauth.Handlers
	}
	Example struct {
		Service  *exampleApp.Service
		Handlers *exampleTransport.Handlers
	}
}

// buildModules constrói cada módulo de negócio do Projeto Aurora.
func buildModules(deps *Dependencies) *Modules {
	auditWriter := audit.NewWriter(deps.DB)

	m := &Modules{}

	// Módulo de Usuários
	usersRepo := usersInfra.NewPostgresRepository(deps.DB)
	usersSvc := usersApp.NewService(usersRepo, auditWriter)
	m.Users.Handlers = usersTransport.NewHandlers(usersSvc, deps.Logger, deps.Config.MaxPageSize)

	// Módulo de Integrações
	integrationsRepo := integrationsInfra.NewPostgresRepository(deps.DB)
	integrationsSvc := integrationsApp.NewService(integrationsRepo)
	m.Integrations.Service = integrationsSvc
	m.Integrations.Handlers = integrationsTransport.NewHandlers(integrationsSvc, deps.Logger)

	// Feature Flags e Autenticação Local
	m.ConfigFlags.Handlers = configflags.NewHandlers(deps.Flags, auditWriter, deps.Logger)

	// Configuração dinâmica do Keycloak (menu Configurações > Keycloak)
	m.KeycloakConfig.Handlers = keycloakconfig.NewHandlers(deps.KeycloakCfg, deps.Verifier, deps.Config.Keycloak, auditWriter, deps.Logger)

	localAuthStore := localauth.NewPostgresStore(deps.DB)
	m.LocalAuth.Handlers = localauth.NewHandlers(localAuthStore, deps.LocalSigner, auditWriter, deps.Logger)

	// Módulo Exemplo (Template genérico para novos módulos)
	exampleRepo := exampleInfra.NewPostgresRepository(deps.DB)
	exampleSvc := exampleApp.NewService(deps.DB, exampleRepo, deps.Outbox, auditWriter, deps.Logger)
	m.Example.Service = exampleSvc
	m.Example.Handlers = exampleTransport.NewHandlers(exampleSvc, deps.Logger)

	return m
}
