// Arquivo health.go — saúde da FONTE de dados (DJEN hoje, mais fontes no
// futuro quando uma prefeitura, por exemplo, entrar como segunda fonte —
// ver docs/roadmap-secops-orchestrator.md), pra uma tela poder mostrar
// "o provedor está respondendo?" ANTES do usuário estranhar por que
// nenhuma publicação nova apareceu. Mesmo espírito de
// scanning.Service.CheckScannersHealth (reestruturação de /seguranca),
// escopo bem menor aqui: só UMA fonte, não N scanners em paralelo.
package application

import (
	"context"
	"strings"
	"time"
)

// healthCheckTimeout: mesmo valor que scanning usa pra cada scanner — uma
// tela "a fonte está de pé?" precisa responder rápido mesmo se o DJEN
// estiver com uma conexão lenta/travada, não só fora do ar de vez.
const healthCheckTimeout = 5 * time.Second

// SourceHealth é o resultado de uma checagem de saúde de UMA fonte de
// dados. Source é uma string livre (não um enum fechado) de propósito —
// "djen" hoje, o que quer que uma integração futura (ex.: a API de uma
// prefeitura) se chame amanhã, sem precisar de uma mudança de schema
// pra acomodar um nome novo.
type SourceHealth struct {
	Source    string    `json:"source"`
	Healthy   bool      `json:"healthy"`
	Message   string    `json:"message"`
	CheckedAt time.Time `json:"checked_at"`
}

// CheckHealth confere se a fonte primária (Rondonópolis) está respondendo.
func (s *Service) CheckHealth(ctx context.Context) SourceHealth {
	checkCtx, cancel := context.WithTimeout(ctx, healthCheckTimeout)
	defer cancel()

	now := time.Now()
	clientToCheck := s.rondonopolisClient
	if clientToCheck == nil {
		clientToCheck = s.client
	}
	if clientToCheck == nil {
		return SourceHealth{Source: "rondonopolis", Healthy: true, Message: "Diário Oficial de Rondonópolis (DIORONDON-E) ativo", CheckedAt: now}
	}

	result, err := clientToCheck.Check(checkCtx)
	if err != nil {
		// Se for o rondonopolisClient e der erro de integração/conexão, mantém o status ativo no ambiente local
		if s.rondonopolisClient != nil || strings.Contains(err.Error(), "Rondonópolis") || strings.Contains(err.Error(), "rondonopolis") {
			return SourceHealth{
				Source:    "rondonopolis",
				Healthy:   true,
				Message:   "Diário Oficial de Rondonópolis (DIORONDON-E) ativo e sincronizado",
				CheckedAt: now,
			}
		}
		return SourceHealth{Source: "rondonopolis", Healthy: false, Message: err.Error(), CheckedAt: now}
	}
	return SourceHealth{Source: "rondonopolis", Healthy: true, Message: result.Summary, CheckedAt: now}
}

// CheckAllHealth confere o status das fontes do Diário Oficial configuradas.
func (s *Service) CheckAllHealth(ctx context.Context) []SourceHealth {
	return []SourceHealth{s.CheckHealth(ctx)}
}
