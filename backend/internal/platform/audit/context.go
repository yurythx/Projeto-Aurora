package audit

import (
	"net/http"

	"github.com/google/uuid"

	"github.com/yurythx/projeto-aurora/internal/platform/logging"
)

// FromRequest devolve uma Entry já preenchida com o IP de origem e o
// correlation_id extraídos da requisição HTTP — o chamador só completa
// UserID/Action/ResourceType/ResourceID/Metadata.
//
// Motivação (achado de auditoria de conformidade, gap G-07): as entradas
// de auditoria das mutações administrativas (feature flags via PATCH,
// configuração do Keycloak via PUT) eram construídas à mão preenchendo só
// Action/ResourceType/ResourceID/Metadata — sem ip_address nem
// correlation_id. §49 exige "identificador do agente, ação, dados
// alterados, timestamp UTC e IP de origem" em toda operação de escrita, e
// eram justamente as operações mais sensíveis (desligar um módulo em
// produção, trocar o issuer OIDC de toda a plataforma) que ficavam sem o
// IP. Este helper centraliza o preenchimento para que nenhuma chamada
// futura repita o esquecimento.
//
//   - IPAddress recebe r.RemoteAddr cru; Writer.Record chama sanitizeIP,
//     que já sabe tirar a porta de um "host:porta" e devolver "" (grava
//     NULL) para um valor que não seja um IP válido — um valor inválido
//     nunca deve derrubar o INSERT de auditoria.
//   - CorrelationID vem de logging.CorrelationID (populado pelo middleware
//     RequestID a partir do X-Request-ID). Se não for um UUID parseável
//     (um cliente pode mandar um X-Request-ID em qualquer formato), o
//     campo fica nil em vez de abortar — degradação graciosa.
func FromRequest(r *http.Request) Entry {
	e := Entry{IPAddress: r.RemoteAddr}
	if cid := logging.CorrelationID(r.Context()); cid != "" {
		if id, err := uuid.Parse(cid); err == nil {
			e.CorrelationID = &id
		}
	}
	return e
}
