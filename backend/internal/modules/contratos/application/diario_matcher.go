package application

import (
	"context"
	"fmt"
	"log/slog"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/yurythx/projeto-nova/internal/modules/contratos/domain"
	"github.com/yurythx/projeto-nova/internal/platform/database"
	"github.com/yurythx/projeto-nova/internal/platform/outbox"
)

// DiarioFinding é a projeção mínima de um ato do Diário Oficial que o
// casador automático precisa. Preenchida por DiarioMatchSource (adaptada
// em internal/app sobre o repositório de findings do módulo diario_oficial
// — os módulos não se importam diretamente).
type DiarioFinding struct {
	EditionNumber  string
	EditionDate    *time.Time
	ActType        string
	ServidorNome   string
	EmpresaNome    string
	CNPJ           string
	PortariaNumber string
	RawContent     string
	DocURL         string // já com âncora #page=N quando conhecida
}

// DiarioMatchSource busca no Diário Oficial os atos que mencionam um
// contrato (pelo número, no texto ou na portaria) ou casam pelo CNPJ.
type DiarioMatchSource interface {
	FindForContract(ctx context.Context, numero, cnpjDigits string) ([]DiarioFinding, error)
}

// WithDiarioMatching liga o casador automático Contrato <-> Diário Oficial.
// db + outbox são exigidos para o padrão Transactional Outbox: cada contrato
// é processado numa única transação onde as linhas de contrato_diario_refs /
// contrato_diario_alertas e os eventos de outbox que as anunciam nascem
// juntos ou não nascem. runInTx / writeEvent são seams sobrescritos em teste.
func (s *Service) WithDiarioMatching(src DiarioMatchSource, db *pgxpool.Pool, ob *outbox.Writer) *Service {
	s.diario = src
	s.db = db
	s.outbox = ob
	s.runInTx = func(ctx context.Context, fn func(pgx.Tx) error) error {
		return database.WithTx(ctx, db, func(ctx context.Context, tx pgx.Tx) error { return fn(tx) })
	}
	s.writeEvent = func(ctx context.Context, tx pgx.Tx, eventType, aggregateID string, payload any) error {
		return ob.Write(ctx, tx, eventType, "contrato", aggregateID, uuid.New(), payload)
	}
	return s
}

var nonDigit = regexp.MustCompile(`\D`)

// statusesElegiveis: contratos que já podem ter publicação no Diário.
var statusesElegiveis = []domain.Status{
	domain.StatusEmAnalise, domain.StatusAprovado, domain.StatusVigente, domain.StatusEncerrado,
}

const (
	AlertKindMovimentacaoPessoal = "MOVIMENTACAO_PESSOAL"
	AlertKindSemVinculoDiario    = "SEM_VINCULO_DIARIO"

	EventDiarioRefLinked = "contrato.diario_ref.linked"
	EventFiscalAlert     = "contrato.fiscal_alert"
)

// atosDeSaida: publicar um destes para alguém ligado ao contrato pode
// significar que o fiscal/preposto mudou.
var atosDeSaida = map[string]bool{
	"EXONERACAO": true, "RESCISAO": true, "RELOTACAO": true,
}

// RunDiarioMatch varre os contratos elegíveis, vincula as publicações do
// Diário Oficial que os citam (idempotente) e abre alertas de fiscalização.
// Chamado por um loop do worker. Devolve (refs novas vinculadas, alertas
// novos emitidos).
func (s *Service) RunDiarioMatch(ctx context.Context) (linked int, alerts int, err error) {
	if s.diario == nil {
		return 0, 0, nil
	}
	byStatus, err := s.repo.ListByStatus(ctx)
	if err != nil {
		return 0, 0, fmt.Errorf("contratos: diario match: list by status: %w", err)
	}

	for _, st := range statusesElegiveis {
		for _, c := range byStatus[st] {
			l, a := s.matchOne(ctx, c, st)
			linked += l
			alerts += a
		}
	}
	if linked > 0 || alerts > 0 {
		s.logger.Info("contratos: casamento com o Diário Oficial",
			slog.Int("refs_vinculadas", linked), slog.Int("alertas", alerts))
	}
	return linked, alerts, nil
}

func fiscalAlertPayload(c domain.Contrato, kind, message string, f DiarioFinding) map[string]any {
	return map[string]any{
		"contrato_id":     c.ID.String(),
		"contrato_numero": c.Numero,
		"kind":            kind,
		"message":         message,
		"servidor":        f.ServidorNome,
		"act_type":        f.ActType,
		"edition_number":  f.EditionNumber,
		"doc_url":         f.DocURL,
	}
}

// matchOne processa UM contrato numa única transação: os INSERTs de
// contrato_diario_refs / contrato_diario_alertas e os eventos de outbox que
// os anunciam são commitados juntos (Transactional Outbox — nada de dual
// write). Qualquer erro faz rollback de tudo e o contrato é reprocessado no
// próximo ciclo (o casador é idempotente).
func (s *Service) matchOne(ctx context.Context, c domain.Contrato, st domain.Status) (linked int, alerts int) {
	numero := strings.TrimSpace(c.Numero)
	if len([]rune(numero)) < 4 {
		numero = "" // curto demais: casaria com qualquer coisa
	}
	cnpjDigits := nonDigit.ReplaceAllString(c.CNPJ, "")
	if len(cnpjDigits) < 14 {
		cnpjDigits = ""
	}
	if numero == "" && cnpjDigits == "" {
		return 0, 0
	}

	findings, err := s.diario.FindForContract(ctx, numero, cnpjDigits)
	if err != nil {
		s.logger.Warn("contratos: busca no Diário falhou", slog.String("contrato", c.Numero), slog.Any("erro", err))
		return 0, 0
	}

	txErr := s.runInTx(ctx, func(tx pgx.Tx) error {
		linked, alerts = 0, 0 // idempotente sob eventual retry da tx
		seenEdition := map[string]bool{}

		for _, f := range findings {
			if f.EditionNumber != "" && !seenEdition[f.EditionNumber] {
				seenEdition[f.EditionNumber] = true
				ref := domain.DiarioRef{
					ContratoID:    c.ID,
					EditionNumber: f.EditionNumber,
					TipoEvento:    tipoEventoFromAct(f.ActType),
					PublicadoEm:   f.EditionDate,
					Contexto:      snippet(f.RawContent, numero),
					DocURL:        f.DocURL,
				}
				inserted, e := s.repo.LinkDiarioRefTx(ctx, tx, ref)
				if e != nil {
					return e
				}
				if inserted {
					linked++
				}
			}

			if st == domain.StatusVigente && f.ServidorNome != "" && atosDeSaida[strings.ToUpper(f.ActType)] {
				refKey := f.EditionNumber + "|" + f.ServidorNome + "|" + strings.ToUpper(f.ActType)
				first, e := s.repo.RecordAlertOnceTx(ctx, tx, c.ID, AlertKindMovimentacaoPessoal, refKey)
				if e != nil {
					return e
				}
				if first {
					msg := fmt.Sprintf("Publicação de %s de %s (edição %s) pode afetar a fiscalização do contrato %s.",
						strings.ToLower(strings.ReplaceAll(f.ActType, "_", " ")), f.ServidorNome, f.EditionNumber, c.Numero)
					if e := s.writeEvent(ctx, tx, EventFiscalAlert, c.ID.String(),
						fiscalAlertPayload(c, AlertKindMovimentacaoPessoal, msg, f)); e != nil {
						return e
					}
					alerts++
				}
			}
		}

		if linked > 0 {
			if e := s.writeEvent(ctx, tx, EventDiarioRefLinked, c.ID.String(), map[string]any{
				"contrato_id":     c.ID.String(),
				"contrato_numero": c.Numero,
				"refs_vinculadas": linked,
			}); e != nil {
				return e
			}
		}

		// Contrato vigente sem nenhuma publicação vinculada (inclusive as
		// recém-inseridas nesta mesma tx): pode estar sem portaria de fiscal.
		if st == domain.StatusVigente {
			refs, e := s.repo.ListDiarioRefsTx(ctx, tx, c.ID)
			if e != nil {
				return e
			}
			if len(refs) == 0 {
				first, e := s.repo.RecordAlertOnceTx(ctx, tx, c.ID, AlertKindSemVinculoDiario, "-")
				if e != nil {
					return e
				}
				if first {
					msg := fmt.Sprintf("Contrato %s está vigente mas não tem nenhuma publicação do Diário Oficial vinculada — verifique a portaria de designação do fiscal.", c.Numero)
					if e := s.writeEvent(ctx, tx, EventFiscalAlert, c.ID.String(),
						fiscalAlertPayload(c, AlertKindSemVinculoDiario, msg, DiarioFinding{})); e != nil {
						return e
					}
					alerts++
				}
			}
		}
		return nil
	})

	if txErr != nil {
		s.logger.Warn("contratos: casamento do contrato falhou (rollback, tenta no próximo ciclo)",
			slog.String("contrato", c.Numero), slog.Any("erro", txErr))
		return 0, 0
	}
	return linked, alerts
}

func tipoEventoFromAct(act string) string {
	switch strings.ToUpper(act) {
	case "CONTRATO", "CONTRATACAO_TEMPORARIA":
		return "CONTRATO"
	case "EXONERACAO", "RESCISAO":
		return "RESCISAO"
	case "RELOTACAO":
		return "RELOTACAO"
	case "DESIGNACAO_FUNCAO":
		return "FISCALIZACAO"
	case "NOMEACAO_EFETIVO", "NOMEACAO_COMISSIONADO":
		return "NOMEACAO"
	default:
		return "MENCAO"
	}
}

// snippet devolve um trecho de ~280 caracteres do texto centrado na
// primeira ocorrência de `termo` (o número do contrato), para dar contexto
// na UI. Opera sobre runes, não bytes: um corte no meio de um caractere
// acentuado geraria UTF-8 inválido e o INSERT de contrato_diario_refs
// falharia (SQLSTATE 22021).
func snippet(raw, termo string) string {
	rs := []rune(strings.Join(strings.Fields(raw), " "))
	const window = 280
	const lead = 80

	startRune := 0
	if termo != "" {
		if i := strings.Index(strings.ToLower(string(rs)), strings.ToLower(termo)); i >= 0 {
			startRune = len([]rune(string(rs)[:i])) - lead
			if startRune < 0 {
				startRune = 0
			}
		}
	}
	endRune := startRune + window
	if endRune > len(rs) {
		endRune = len(rs)
	}
	return strings.TrimSpace(string(rs[startRune:endRune]))
}
