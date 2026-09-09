// Package transparency expõe dados de interesse público em formato aberto
// (JSON, CSV, XML), SEM autenticação — transparência ativa (LAI, Lei
// 12.527/2011, art. 8º; LC 131/2009). Rotas montadas em
// /api/v1/transparencia/*, rate-limited por IP.
//
// F3.5 do roadmap de conformidade — SCAFFOLD: o órgão define quais
// datasets de interesse público de fato publicar. Os datasets aqui são
// exemplos deliberadamente seguros (agregados, nunca dado pessoal):
//   - GET /transparencia/datasets        → catálogo + dicionário de dados
//   - GET /transparencia/plataforma      → métricas agregadas da plataforma
//   - GET /transparencia/integracoes     → status dos serviços externos
//   - GET /transparencia/auditoria/acoes → catálogo de ações auditadas + volume
//
// Todos aceitam ?format=json|csv|xml (default: json) ou o header Accept.
package transparency

import (
	"encoding/csv"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	apperrors "github.com/yurythx/projeto-aurora/internal/domain/errors"
	"github.com/yurythx/projeto-aurora/internal/platform/httpserver"
	"github.com/yurythx/projeto-aurora/pkg/httputil"
)

type Service struct {
	db     *pgxpool.Pool
	logger *slog.Logger
}

func NewService(db *pgxpool.Pool, logger *slog.Logger) *Service {
	return &Service{db: db, logger: logger}
}

// RegisterRoutes monta as rotas públicas de transparência. r deve ser um
// router SEM auth.RequireAuthentication; limiter aplica rate limit por IP.
func RegisterRoutes(r chi.Router, s *Service, limiter httpserver.Limiter) {
	r.Route("/api/v1/transparencia", func(t chi.Router) {
		t.Use(httpserver.RateLimit(s.logger, limiter, httpserver.ClientIPKey))
		t.Get("/datasets", s.handleCatalog)
		t.Get("/plataforma", s.handlePlatform)
		t.Get("/integracoes", s.handleIntegrations)
		t.Get("/auditoria/acoes", s.handleAuditActions)
	})
}

// ---- catálogo / dicionário de dados -------------------------------------

func (s *Service) handleCatalog(w http.ResponseWriter, r *http.Request) {
	catalog := []map[string]any{
		{
			"id": "plataforma", "titulo": "Métricas agregadas da plataforma",
			"endpoint": "/api/v1/transparencia/plataforma",
			"campos": map[string]string{
				"usuarios_ativos":      "inteiro — contagem de contas ativas (sem identificação)",
				"integracoes_total":    "inteiro — total de integrações externas cadastradas",
				"termos_lgpd_vigentes": "texto — versão vigente dos Termos/Política de Privacidade",
				"gerado_em":            "data-hora UTC (RFC3339)",
			},
			"periodicidade": "tempo real", "formatos": []string{"json", "csv", "xml"},
		},
		{
			"id": "integracoes", "titulo": "Serviços externos e disponibilidade",
			"endpoint": "/api/v1/transparencia/integracoes",
			"campos": map[string]string{
				"nome": "texto", "tipo": "texto", "status": "texto (online|offline|degraded|unknown|disabled)",
				"ultima_verificacao": "data-hora UTC ou vazio",
			},
			"periodicidade": "tempo real", "formatos": []string{"json", "csv", "xml"},
		},
		{
			"id": "auditoria_acoes", "titulo": "Ações registradas na trilha de auditoria (agregado)",
			"endpoint": "/api/v1/transparencia/auditoria/acoes?dias=30",
			"campos": map[string]string{
				"acao":         "texto — identificador da ação (ex.: login, user.created)",
				"ocorrencias":  "inteiro — quantas vezes no período",
				"periodo_dias": "inteiro — janela consultada (default 30, máx 365)",
			},
			"periodicidade": "tempo real", "formatos": []string{"json", "csv", "xml"},
		},
	}
	s.write(w, r, "catalogo_transparencia", map[string]any{
		"nota": "Transparência ativa (LAI art. 8º). Datasets de exemplo, agregados e sem dado pessoal — " +
			"o órgão define os datasets de interesse público a publicar.",
	}, catalog)
}

// ---- plataforma -------------------------------------------------------

func (s *Service) handlePlatform(w http.ResponseWriter, r *http.Request) {
	var usuariosAtivos, integracoesTotal int
	_ = s.db.QueryRow(r.Context(), `SELECT count(*) FROM users WHERE active`).Scan(&usuariosAtivos)
	_ = s.db.QueryRow(r.Context(), `SELECT count(*) FROM integrations`).Scan(&integracoesTotal)

	row := map[string]any{
		"usuarios_ativos":      usuariosAtivos,
		"integracoes_total":    integracoesTotal,
		"termos_lgpd_vigentes": "v1.0.0-2026",
		"gerado_em":            time.Now().UTC().Format(time.RFC3339),
	}
	s.write(w, r, "plataforma", nil, []map[string]any{row})
}

// ---- integrações ----------------------------------------------------

func (s *Service) handleIntegrations(w http.ResponseWriter, r *http.Request) {
	rows, err := s.db.Query(r.Context(), `
		SELECT name, type, status, COALESCE(last_check_at::text, '')
		FROM integrations ORDER BY name`)
	if err != nil {
		httputil.WriteError(w, r, s.logger, apperrors.Internal(fmt.Errorf("transparency integrations: %w", err)))
		return
	}
	defer rows.Close()
	out := []map[string]any{}
	for rows.Next() {
		var name, typ, status, lastCheck string
		if err := rows.Scan(&name, &typ, &status, &lastCheck); err != nil {
			continue
		}
		out = append(out, map[string]any{"nome": name, "tipo": typ, "status": status, "ultima_verificacao": lastCheck})
	}
	s.write(w, r, "integracoes", nil, out)
}

// ---- ações auditadas (agregado) -----------------------------------

func (s *Service) handleAuditActions(w http.ResponseWriter, r *http.Request) {
	dias := 30
	if v := r.URL.Query().Get("dias"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 365 {
			dias = n
		}
	}
	since := time.Now().UTC().AddDate(0, 0, -dias)
	rows, err := s.db.Query(r.Context(), `
		SELECT action, count(*) FROM audit_logs
		WHERE created_at >= $1
		GROUP BY action ORDER BY count(*) DESC`, since)
	if err != nil {
		httputil.WriteError(w, r, s.logger, apperrors.Internal(fmt.Errorf("transparency audit actions: %w", err)))
		return
	}
	defer rows.Close()
	out := []map[string]any{}
	for rows.Next() {
		var action string
		var n int
		if err := rows.Scan(&action, &n); err != nil {
			continue
		}
		out = append(out, map[string]any{"acao": action, "ocorrencias": n, "periodo_dias": dias})
	}
	s.write(w, r, "auditoria_acoes", map[string]any{"periodo_dias": dias, "desde": since.Format(time.RFC3339)}, out)
}

// ---- serialização multi-formato --------------------------------------

func (s *Service) write(w http.ResponseWriter, r *http.Request, name string, meta map[string]any, rows []map[string]any) {
	format := chooseFormat(r)
	w.Header().Set("X-Dataset-Name", name)
	w.Header().Set("X-Generated-At", time.Now().UTC().Format(time.RFC3339))
	w.Header().Set("Cache-Control", "public, max-age=60")

	switch format {
	case "csv":
		w.Header().Set("Content-Type", "text/csv; charset=utf-8")
		cw := csv.NewWriter(w)
		if len(rows) > 0 {
			cols := sortedKeys(rows[0])
			_ = cw.Write(cols)
			for _, row := range rows {
				rec := make([]string, len(cols))
				for i, c := range cols {
					rec[i] = fmt.Sprintf("%v", row[c])
				}
				_ = cw.Write(rec)
			}
		}
		cw.Flush()
	case "xml":
		w.Header().Set("Content-Type", "application/xml; charset=utf-8")
		_, _ = w.Write([]byte(xml.Header))
		_, _ = w.Write([]byte("<dataset name=\"" + name + "\">\n"))
		for _, row := range rows {
			_, _ = w.Write([]byte("  <registro>\n"))
			for _, c := range sortedKeys(row) {
				_, _ = fmt.Fprintf(w, "    <%s>%v</%s>\n", c, row[c], c)
			}
			_, _ = w.Write([]byte("  </registro>\n"))
		}
		_, _ = w.Write([]byte("</dataset>\n"))
	default: // json
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		body := map[string]any{"dataset": name, "registros": rows}
		if meta != nil {
			body["meta"] = meta
		}
		_ = json.NewEncoder(w).Encode(body)
	}
}

func sortedKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	// ordenação simples por inserção — poucos campos por dataset
	for i := 1; i < len(keys); i++ {
		for j := i; j > 0 && keys[j-1] > keys[j]; j-- {
			keys[j-1], keys[j] = keys[j], keys[j-1]
		}
	}
	return keys
}

func chooseFormat(r *http.Request) string {
	switch strings.ToLower(strings.TrimSpace(r.URL.Query().Get("format"))) {
	case "csv":
		return "csv"
	case "xml":
		return "xml"
	case "json":
		return "json"
	}
	accept := r.Header.Get("Accept")
	switch {
	case strings.Contains(accept, "text/csv"):
		return "csv"
	case strings.Contains(accept, "application/xml"), strings.Contains(accept, "text/xml"):
		return "xml"
	default:
		return "json"
	}
}
