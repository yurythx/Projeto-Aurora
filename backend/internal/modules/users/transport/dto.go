// Package transport implementa os handlers HTTP do módulo users. Não
// contém nenhuma regra de negócio — só parsing/validação de requisição,
// chamadas para application.Service, e formatação de resposta (§24).
package transport

import (
	"time"

	"github.com/yurythx/projeto-aurora/internal/modules/users/domain"
)

// UserResponse é o formato público de um usuário retornado pela API —
// deliberadamente não inclui KeycloakSubject, que é um detalhe interno de
// sincronização com o Keycloak, não algo que o cliente precisa ver.
type UserResponse struct {
	ID          string     `json:"id"`
	Username    string     `json:"username"`
	Email       string     `json:"email"`
	DisplayName string     `json:"display_name"`
	Active      bool       `json:"active"`
	CreatedAt   time.Time  `json:"created_at"`
	LastSeenAt  *time.Time `json:"last_seen_at,omitempty"`
}

// toUserResponse é o formato completo (com e-mail) — usado só em
// GET /api/v1/users/{id} e GET /api/v1/me.
func toUserResponse(u *domain.User) UserResponse {
	return UserResponse{
		ID:          u.ID.String(),
		Username:    u.Username,
		Email:       u.Email,
		DisplayName: u.DisplayName,
		Active:      u.Active,
		CreatedAt:   u.CreatedAt,
		LastSeenAt:  u.LastSeenAt,
	}
}

// UserListItem é a projeção de GET /api/v1/users (lista) — ver toUserResponse
// acima para o formato completo. Gap G-10 da
// auditoria de conformidade: deliberadamente SEM e-mail e sem
// last_seen_at — esses são dados pessoais de outro titular (LGPD art. 6º
// III, minimização) e não são necessários para uma listagem. O e-mail
// completo só aparece em GET /api/v1/users/{id} (aurora-auditor/admin) e
// em GET /api/v1/me (o próprio titular).
type UserListItem struct {
	ID          string    `json:"id"`
	Username    string    `json:"username"`
	DisplayName string    `json:"display_name"`
	Active      bool      `json:"active"`
	CreatedAt   time.Time `json:"created_at"`
}

func toUserListItem(u *domain.User) UserListItem {
	return UserListItem{
		ID:          u.ID.String(),
		Username:    u.Username,
		DisplayName: u.DisplayName,
		Active:      u.Active,
		CreatedAt:   u.CreatedAt,
	}
}

func toUserListItems(users []*domain.User) []UserListItem {
	out := make([]UserListItem, 0, len(users))
	for _, u := range users {
		out = append(out, toUserListItem(u))
	}
	return out
}
