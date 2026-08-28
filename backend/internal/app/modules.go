package app

import (
	contratosApp "github.com/yurythx/projeto-nova/internal/modules/contratos/application"
	contratosInfra "github.com/yurythx/projeto-nova/internal/modules/contratos/infrastructure"
	contratosTransport "github.com/yurythx/projeto-nova/internal/modules/contratos/transport"

	diarioApp "github.com/yurythx/projeto-nova/internal/modules/diario_oficial/application"
	diarioInfra "github.com/yurythx/projeto-nova/internal/modules/diario_oficial/infrastructure"
	diarioTransport "github.com/yurythx/projeto-nova/internal/modules/diario_oficial/transport"

	integrationsApp "github.com/yurythx/projeto-nova/internal/modules/integrations/application"
	integrationsInfra "github.com/yurythx/projeto-nova/internal/modules/integrations/infrastructure"
	integrationsTransport "github.com/yurythx/projeto-nova/internal/modules/integrations/transport"

	usersApp "github.com/yurythx/projeto-nova/internal/modules/users/application"
	usersInfra "github.com/yurythx/projeto-nova/internal/modules/users/infrastructure"
	usersTransport "github.com/yurythx/projeto-nova/internal/modules/users/transport"

	"github.com/yurythx/projeto-nova/internal/platform/audit"
	"github.com/yurythx/projeto-nova/internal/platform/configflags"
	"github.com/yurythx/projeto-nova/internal/platform/jobs"
	"github.com/yurythx/projeto-nova/internal/platform/localauth"

	demandsApp "github.com/yurythx/projeto-nova/internal/modules/demands/application"
	demandsInfra "github.com/yurythx/projeto-nova/internal/modules/demands/infrastructure"
	demandsTransport "github.com/yurythx/projeto-nova/internal/modules/demands/transport"
)

// Modules guarda o serviço de aplicação e os handlers HTTP de todo módulo
// de negócio. Construído uma vez por processo (cmd/api e cmd/worker cada
// um recebe o seu via NewDependencies) — este é o único arquivo autorizado
// a importar todo módulo de uma vez (§76), servindo como o "ponto de
// montagem" central que conecta domain/application/infrastructure/
// transport de cada módulo entre si e com a plataforma.
type Modules struct {
	Users struct {
		Handlers *usersTransport.Handlers
	}
	Integrations struct {
		Service  *integrationsApp.Service
		Handlers *integrationsTransport.Handlers
	}
	DiarioOficial struct {
		Service  *diarioApp.Service
		Handlers *diarioTransport.Handlers
	}
	Contratos struct {
		Service  *contratosApp.Service
		Handlers *contratosTransport.Handlers
	}
	Demands struct {
		Service  *demandsApp.Service
		Handlers *demandsTransport.Handlers
	}
	ConfigFlags struct {
		Handlers *configflags.Handlers
	}
	LocalAuth struct {
		Handlers *localauth.Handlers
	}
}

// buildModules constrói cada módulo de negócio na ordem certa: primeiro
// as dependências compartilhadas (repositório de jobs, writer de
// auditoria), depois users e integrations (que não dependem de outros
// módulos), e por fim diario_oficial (que depende do integrationsSvc
// já construído para registrar o resultado de seus testes — ver
// internal/modules/integrations/application.Service.RecordCheckResult).
func buildModules(deps *Dependencies) *Modules {
	jobsRepo := jobs.NewRepository(deps.DB)
	auditWriter := audit.NewWriter(deps.DB)

	m := &Modules{}

	usersRepo := usersInfra.NewPostgresRepository(deps.DB)
	usersSvc := usersApp.NewService(usersRepo, auditWriter)
	m.Users.Handlers = usersTransport.NewHandlers(usersSvc, deps.Logger, deps.Config.MaxPageSize)

	integrationsRepo := integrationsInfra.NewPostgresRepository(deps.DB)
	integrationsSvc := integrationsApp.NewService(integrationsRepo)
	m.Integrations.Service = integrationsSvc
	m.Integrations.Handlers = integrationsTransport.NewHandlers(integrationsSvc, deps.Logger)

	diarioClient := diarioInfra.NewHTTPClient(deps.Config.DiarioOficial.BaseURL, deps.Config.DiarioOficial.Timeout, deps.Logger)
	rondonopolisClient := diarioInfra.NewRondonopolisClient(deps.Config.DiarioOficial.RondonopolisBaseURL, deps.Config.DiarioOficial.RondonopolisToken, deps.Config.DiarioOficial.Timeout, deps.Logger)
	diarioRepo := diarioInfra.NewPostgresRepository(deps.DB)
	editionRepo := diarioInfra.NewPostgresEditionRepository(deps.DB)
	diarioSvc := diarioApp.NewService(deps.DB, jobsRepo, deps.Outbox, diarioClient, diarioRepo, integrationsSvc, auditWriter, deps.Flags, deps.Logger).WithRondonopolisClient(rondonopolisClient)
	diarioSvc.SetEditionRepository(editionRepo)
	m.DiarioOficial.Service = diarioSvc
	m.DiarioOficial.Handlers = diarioTransport.NewHandlers(diarioSvc, deps.Logger)

	m.ConfigFlags.Handlers = configflags.NewHandlers(deps.Flags, auditWriter, deps.Logger)

	localAuthStore := localauth.NewPostgresStore(deps.DB)
	m.LocalAuth.Handlers = localauth.NewHandlers(localAuthStore, deps.LocalSigner, auditWriter, deps.Logger)

	// Módulo Contratos Municipais (Fase 4 — Projeto-Nova)
	contratosRepo := contratosInfra.NewPostgresRepository(deps.DB)
	contratosSvc := contratosApp.NewService(contratosRepo, deps.Logger)
	m.Contratos.Service = contratosSvc
	m.Contratos.Handlers = contratosTransport.NewHandlers(contratosSvc, deps.Logger)

	// Módulo Demandas Mensais (Kanban)
	demandsRepo := demandsInfra.NewPostgresRepository(deps.DB)
	demandsSvc := demandsApp.NewService(deps.DB, demandsRepo, deps.Outbox, deps.Storage, auditWriter, deps.Config.MinIO.Bucket, deps.Logger)
	m.Demands.Service = demandsSvc
	m.Demands.Handlers = demandsTransport.NewHandlers(demandsSvc, deps.Logger)

	return m
}
