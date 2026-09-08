// Package logging configura o logger estruturado da aplicação (log/slog).
// Ele nunca deve logar segredos: quem chama não pode passar tokens, senhas
// ou client secrets como atributos de log.
package logging

import (
	"context"
	"log/slog"
	"os"
	"strings"
)

type ctxKey string

const (
	requestIDKey     ctxKey = "request_id"
	correlationIDKey ctxKey = "correlation_id"
	userIDKey        ctxKey = "user_id"
)

// Options configura o logger base.
type Options struct {
	Level       string // debug | info | warn | error
	Format      string // json | text
	Service     string
	Environment string
}

// New constrói um *slog.Logger escrevendo em stdout no formato e nível
// pedidos, com os atributos service/environment presos em todo registro —
// assim toda linha de log já vem identificada com "de qual processo" e
// "em qual ambiente" ela veio, sem cada chamador precisar repetir isso.
func New(opts Options) *slog.Logger {
	level := parseLevel(opts.Level)

	handlerOpts := &slog.HandlerOptions{
		Level: level,
	}

	var handler slog.Handler
	if strings.EqualFold(opts.Format, "text") {
		// Formato texto é mais legível rodando localmente no terminal;
		// json é o que se espera em produção, onde os logs são coletados
		// por um agregador (ver APP_LOG_FORMAT no .env.example).
		handler = slog.NewTextHandler(os.Stdout, handlerOpts)
	} else {
		handler = slog.NewJSONHandler(os.Stdout, handlerOpts)
	}

	logger := slog.New(handler).With(
		slog.String("service", opts.Service),
		slog.String("environment", opts.Environment),
	)

	return logger
}

func parseLevel(level string) slog.Level {
	switch strings.ToLower(level) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// WithRequestID retorna um context carregando o id da requisição HTTP
// informado (normalmente o valor do header X-Request-ID, gerado pelo
// middleware RequestID se o cliente não mandar um).
func WithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, requestIDKey, id)
}

// RequestID extrai o id de requisição previamente guardado via WithRequestID.
func RequestID(ctx context.Context) string {
	v, _ := ctx.Value(requestIDKey).(string)
	return v
}

// WithCorrelationID retorna um context carregando o correlation id
// informado — usado para rastrear um único fluxo de negócio (ex.: "rodar
// o teste de uma integração") por HTTP, Postgres, RabbitMQ, o worker e as
// notificações via WebSocket, mesmo cruzando processos (§50).
func WithCorrelationID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, correlationIDKey, id)
}

// CorrelationID extrai o correlation id previamente guardado via
// WithCorrelationID.
func CorrelationID(ctx context.Context) string {
	v, _ := ctx.Value(correlationIDKey).(string)
	return v
}

// WithUserID retorna um context carregando o id do usuário autenticado.
func WithUserID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, userIDKey, id)
}

// UserID extrai o id de usuário previamente guardado via WithUserID.
func UserID(ctx context.Context) string {
	v, _ := ctx.Value(userIDKey).(string)
	return v
}

// FromContext retorna um logger enriquecido com os atributos
// request_id/correlation_id/user_id extraídos de ctx, quando presentes.
// Handlers e casos de uso devem chamar esta função em vez de logar
// diretamente com o logger base, para que toda linha de log possa ser
// correlacionada a uma única requisição/fluxo de negócio.
func FromContext(ctx context.Context, base *slog.Logger) *slog.Logger {
	l := base
	if id := RequestID(ctx); id != "" {
		l = l.With(slog.String("request_id", id))
	}
	if id := CorrelationID(ctx); id != "" {
		l = l.With(slog.String("correlation_id", id))
	}
	if id := UserID(ctx); id != "" {
		l = l.With(slog.String("user_id", id))
	}
	return l
}

// -----------------------------------------------------------------------------
// Higienização e Mascaramento de PII (LGPD - Lei 13.709/2018)
// -----------------------------------------------------------------------------

// SecretString oculta completamente o valor quando passado para o slog.
type SecretString string

func (s SecretString) LogValue() slog.Value {
	return slog.StringValue("[REDACTED]")
}

// PIICPF aplica mascaramento em CPF (ex.: "123.***.***-45").
type PIICPF string

func (c PIICPF) LogValue() slog.Value {
	return slog.StringValue(MaskCPF(string(c)))
}

// PIIEmail aplica mascaramento em e-mail (ex.: "u***r@domain.com").
type PIIEmail string

func (e PIIEmail) LogValue() slog.Value {
	return slog.StringValue(MaskEmail(string(e)))
}

// PIIPhone aplica mascaramento em telefone (ex.: "(66) 9****-1234").
type PIIPhone string

func (p PIIPhone) LogValue() slog.Value {
	return slog.StringValue(MaskPhone(string(p)))
}

// MaskCPF oculta dígitos centrais do CPF mantendo os primeiros e últimos dígitos visíveis para depuração.
func MaskCPF(cpf string) string {
	clean := strings.Map(func(r rune) rune {
		if r >= '0' && r <= '9' {
			return r
		}
		return -1
	}, cpf)
	if len(clean) != 11 {
		return "***.***.***-**"
	}
	return clean[:3] + ".***.***-" + clean[9:]
}

// MaskEmail oculta o corpo do e-mail mantendo a primeira e última letra do usuário e o domínio.
func MaskEmail(email string) string {
	parts := strings.Split(email, "@")
	if len(parts) != 2 || len(parts[0]) <= 2 {
		return "***@***"
	}
	user := parts[0]
	domain := parts[1]
	maskedUser := string(user[0]) + strings.Repeat("*", len(user)-2) + string(user[len(user)-1])
	return maskedUser + "@" + domain
}

// MaskPhone oculta os dígitos intermediários do número de telefone.
func MaskPhone(phone string) string {
	if len(phone) < 8 {
		return "*****"
	}
	return phone[:3] + "****" + phone[len(phone)-2:]
}
