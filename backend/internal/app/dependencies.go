// Package app conecta entre si toda dependência de plataforma e de módulo
// (banco de dados, mensageria, autenticação, hub de WebSocket, serviços de
// módulo) e expõe o router HTTP já montado usado pelo cmd/api e o runner
// de worker já montado usado pelo cmd/worker. É o único lugar autorizado a
// conhecer todo módulo de uma vez — nenhum módulo importa outro
// diretamente, só internal/app os conecta.
package app

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/yurythx/projeto-aurora/internal/domain/events"
	"github.com/yurythx/projeto-aurora/internal/platform/audit"
	"github.com/yurythx/projeto-aurora/internal/platform/auth"
	"github.com/yurythx/projeto-aurora/internal/platform/config"
	"github.com/yurythx/projeto-aurora/internal/platform/configflags"
	"github.com/yurythx/projeto-aurora/internal/platform/database"
	"github.com/yurythx/projeto-aurora/internal/platform/httpserver"
	"github.com/yurythx/projeto-aurora/internal/platform/idempotency"
	"github.com/yurythx/projeto-aurora/internal/platform/keycloakconfig"
	"github.com/yurythx/projeto-aurora/internal/platform/lgpd"
	"github.com/yurythx/projeto-aurora/internal/platform/logging"
	"github.com/yurythx/projeto-aurora/internal/platform/messaging"
	"github.com/yurythx/projeto-aurora/internal/platform/metrics"
	"github.com/yurythx/projeto-aurora/internal/platform/outbox"
	"github.com/yurythx/projeto-aurora/internal/platform/ratelimit"
	"github.com/yurythx/projeto-aurora/internal/platform/secretcrypto"
	"github.com/yurythx/projeto-aurora/internal/platform/storage"
	"github.com/yurythx/projeto-aurora/internal/platform/telemetry"
	"github.com/yurythx/projeto-aurora/internal/platform/transparency"
	"github.com/yurythx/projeto-aurora/internal/platform/ws"
)

// RateLimiters guarda todo rate limiter distribuído (baseado em Postgres —
// rate limiting distribuído) que a API usa. Construído uma única vez por
// processo e compartilhado por toda requisição, para que cada réplica da
// API leia/escreva as mesmas linhas de rate_limit_buckets em vez de cada
// uma manter sua própria contagem independente em memória (e portanto
// N×-generosa demais).
type RateLimiters struct {
	WSTicket   httpserver.Limiter // POST /api/v1/ws/ticket
	LocalLogin httpserver.Limiter // POST /api/v1/auth/login — chave por IP, não por usuário (§ Sistema de Login Local), já que quem chama ainda não está autenticado
	// APIGlobal é aplicado a TODO o grupo /api/v1, com chave = subject do
	// usuário autenticado (fallback: IP). Antes só login e /ws/ticket
	// tinham limite e qualquer token válido marretava as demais rotas
	// sem teto (gap G-01 da auditoria de conformidade).
	APIGlobal httpserver.Limiter
	// AnonConsent limita POST /api/v1/lgpd/accept-anon por IP — rota
	// pública de consentimento de visitante não autenticado (gap G-11).
	AnonConsent httpserver.Limiter
	// PublicRead limita as rotas públicas de transparência ativa por IP
	// (F3.5).
	PublicRead httpserver.Limiter
}

// OutboxSource identifica este backend como o Source carimbado em todo
// envelope de evento gravado no outbox.
const OutboxSource = "projeto-aurora.platform"

// Dependencies guarda todo recurso de plataforma compartilhado.
// Dependências específicas de módulo (repositórios, casos de uso) são
// adicionadas a esta struct conforme cada módulo é conectado; nada aqui
// deve carregar regra de negócio.
type Dependencies struct {
	Config       *config.Config
	Logger       *slog.Logger
	DB           *pgxpool.Pool
	Verifier     *auth.Verifier
	LocalSigner  *auth.LocalSigner
	Messaging    *messaging.Connection
	Publisher    events.EventPublisher
	Outbox       *outbox.Writer
	OutboxStats  *outbox.Stats
	Storage      storage.Provider
	Hub          *ws.Hub
	Tickets      *ws.TicketStore
	Modules      *Modules
	RateLimiters *RateLimiters
	Idempotency  idempotency.Store
	Flags        configflags.Store
	KeycloakCfg  keycloakconfig.Store
	LGPDSvc      *lgpd.Service
	Transparency *transparency.Service
	AuditExp     *audit.Exporter

	telemetryShutdown telemetry.Shutdown
}

// NewDependencies constrói e valida toda dependência de plataforma para um
// processo. component distingue "api" de "worker" em logs e traces (os
// dois compartilham este mesmo bootstrap). Retorna um erro imediatamente
// se qualquer dependência obrigatória (banco de dados, RabbitMQ, discovery
// OIDC) não puder ser alcançada, para que o processo falhe rápido em vez
// de servir tráfego num estado parcialmente inicializado.
func NewDependencies(ctx context.Context, component string) (*Dependencies, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("app: load config: %w", err)
	}

	serviceName := cfg.App.Name + "-" + component
	logger := logging.New(logging.Options{
		Level:       cfg.App.LogLevel,
		Format:      cfg.App.LogFormat,
		Service:     serviceName,
		Environment: cfg.App.Env,
	})

	telemetryShutdown, err := telemetry.Setup(ctx, serviceName, cfg.App.Env, cfg.OTELExporterOTLPURL, logger)
	if err != nil {
		return nil, fmt.Errorf("app: setup telemetry: %w", err)
	}

	pool, err := database.New(ctx, cfg.Database)
	if err != nil {
		return nil, fmt.Errorf("app: connect database: %w", err)
	}
	metrics.RegisterPostgresPoolMetrics(pool)

	localSigner, err := auth.NewLocalSigner(cfg.LocalAuth)
	if err != nil {
		pool.Close()
		return nil, fmt.Errorf("app: initialize local auth signer: %w", err)
	}

	verifier, err := auth.NewVerifier(ctx, cfg.Keycloak, localSigner)
	if err != nil {
		pool.Close()
		return nil, fmt.Errorf("app: initialize OIDC verifier: %w", err)
	}

	configCipher, err := secretcrypto.NewFromBase64Key(cfg.Security.ConfigEncryptionKey)
	if err != nil {
		pool.Close()
		return nil, fmt.Errorf("app: initialize config encryption cipher: %w", err)
	}
	keycloakCfgStore := keycloakconfig.NewPostgresStore(pool, configCipher)

	// Se um admin já salvou uma configuração do Keycloak pelo menu
	// Configurações > Keycloak (ver internal/platform/keycloakconfig) ANTES
	// deste boot, ela precisa valer imediatamente — sem isto, um restart do
	// processo (deploy, crash, `docker compose restart`) voltaria
	// silenciosamente a usar só as variáveis de ambiente (cfg.Keycloak),
	// desfazendo a configuração salva até alguém reabrir a tela e salvar de
	// novo. Erro aqui não impede o boot (o processo já subiu válido com
	// cfg.Keycloak via NewVerifier acima) — só fica registrado em log,
	// porque um Postgres consultável no boot mas com uma configuração
	// salva que não faz mais discovery (ex.: issuer desativado depois de
	// salvo) não deveria travar toda a API.
	if savedKeycloak, err := keycloakCfgStore.Get(ctx); err != nil {
		logger.Error("app: falha ao ler configuração do Keycloak persistida no Postgres — seguindo só com variáveis de ambiente",
			slog.String("erro", err.Error()))
	} else if savedKeycloak.Configured {
		reloadCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
		reloadErr := verifier.Reload(reloadCtx, savedKeycloak.ToKeycloakConfig())
		cancel()
		if reloadErr != nil {
			logger.Error("app: configuração do Keycloak persistida no Postgres não passou no discovery no boot — seguindo com variáveis de ambiente",
				slog.String("erro", reloadErr.Error()))
		}
	}

	mqConn, err := messaging.Connect(ctx, cfg.RabbitMQ.URL, logger)
	if err != nil {
		pool.Close()
		return nil, fmt.Errorf("app: connect to rabbitmq: %w", err)
	}

	// Declara a topologia (exchange, filas, DLQs, bindings) aqui no
	// bootstrap — tanto o cmd/api quanto o cmd/worker chamam
	// NewDependencies, então a topologia existe garantidamente antes de
	// qualquer um dos dois tentar publicar ou consumir, não importa qual
	// suba primeiro.
	topologyCh, err := mqConn.Channel()
	if err != nil {
		pool.Close()
		_ = mqConn.Close()
		return nil, fmt.Errorf("app: open channel to declare topology: %w", err)
	}
	if err := messaging.DeclareTopology(topologyCh, messaging.AllQueues()); err != nil {
		pool.Close()
		_ = mqConn.Close()
		return nil, fmt.Errorf("app: declare rabbitmq topology: %w", err)
	}
	_ = topologyCh.Close()

	publisher := messaging.NewPublisher(mqConn)

	minioProvider, err := storage.NewMinioProviderWithPresign(
		cfg.MinIO.Endpoint, cfg.MinIO.PublicEndpoint,
		cfg.MinIO.AccessKey, cfg.MinIO.SecretKey,
		cfg.MinIO.UseSSL, cfg.MinIO.PublicUseSSL)
	if err != nil {
		pool.Close()
		_ = mqConn.Close()
		return nil, fmt.Errorf("app: connect to minio: %w", err)
	}

	err = minioProvider.EnsureBucket(ctx, cfg.MinIO.Bucket)
	if err != nil {
		logger.Warn("não foi possível garantir a existência do bucket do minio na inicialização", slog.String("erro", err.Error()))
	}

	// Proxies reversos confiáveis (gap G-04): valida os CIDRs no boot —
	// uma faixa mal digitada é falha de configuração de segurança, o
	// processo deve recusar subir em vez de confiar num XFF que não
	// deveria.
	trustedProxies, err := httpserver.ParseTrustedProxies(cfg.Security.TrustedProxies)
	if err != nil {
		pool.Close()
		_ = mqConn.Close()
		return nil, fmt.Errorf("app: parse TRUSTED_PROXIES: %w", err)
	}

	deps := &Dependencies{
		Config:      cfg,
		Logger:      logger,
		DB:          pool,
		Verifier:    verifier,
		LocalSigner: localSigner,
		Messaging:   mqConn,
		Publisher:   publisher,
		Outbox:      outbox.NewWriter(OutboxSource),
		OutboxStats: outbox.NewStats(pool),
		Storage:     minioProvider,
		Hub:         ws.NewHub(logger),
		Tickets:     ws.NewTicketStore(ws.TicketTTL),
		RateLimiters: &RateLimiters{
			// O último argumento (bucket) namespaceia cada limiter dentro
			// de rate_limit_buckets — ver o comentário de
			// ratelimit.PostgresLimiter: sem ele, dois limiters
			// diferentes recebendo o MESMO key (o subject do usuário
			// autenticado, comum entre rotas de módulos diferentes)
			// escreviam na mesma linha e se anulavam.
			//
			// Equivalente a 1 req/s, burst 5: até 5 requisições a cada 5s.
			WSTicket: ratelimit.NewPostgresLimiter(pool, 5, 5, "ws_ticket"),
			// Mais apertado de propósito — até 5 tentativas de login a
			// cada 60s por IP, para desacelerar força bruta de senha sem
			// travar um usuário legítimo que só errou a senha uma ou
			// duas vezes.
			LocalLogin: ratelimit.NewPostgresLimiter(pool, 60, 5, "local_login"),
			// Teto largo por identidade autenticada — não estorva o uso
			// normal do painel, só barra abuso grosseiro. Parametrizável
			// por API_RATE_LIMIT_WINDOW_SECONDS / API_RATE_LIMIT_MAX.
			APIGlobal: ratelimit.NewPostgresLimiter(pool, cfg.APIRateLimit.WindowSeconds, cfg.APIRateLimit.MaxRequests, "api_global"),
			// Consentimento anônimo: até 10 por IP a cada 60s — um
			// visitante legítimo aceita uma vez.
			AnonConsent: ratelimit.NewPostgresLimiter(pool, 60, 10, "anon_consent"),
			// Transparência ativa (leitura pública): 60 por IP a cada 60s.
			PublicRead: ratelimit.NewPostgresLimiter(pool, 60, 60, "public_read"),
		},
		Idempotency:  idempotency.NewPostgresStore(pool),
		Flags:        configflags.NewPostgresStore(pool),
		KeycloakCfg:  keycloakCfgStore,
		LGPDSvc:      lgpd.NewService(pool, logger, trustedProxies),
		Transparency: transparency.NewService(pool, logger),
		AuditExp:     audit.NewExporter(pool, logger),

		telemetryShutdown: telemetryShutdown,
	}
	deps.Modules = buildModules(deps)

	return deps, nil
}

// Close libera todo recurso aberto por NewDependencies. Seguro de chamar
// uma vez durante o graceful shutdown.
func (d *Dependencies) Close() {
	if d.Tickets != nil {
		d.Tickets.Close()
	}
	if d.Messaging != nil {
		_ = d.Messaging.Close()
	}
	if d.DB != nil {
		d.DB.Close()
	}
	if d.telemetryShutdown != nil {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = d.telemetryShutdown(shutdownCtx)
	}
}
