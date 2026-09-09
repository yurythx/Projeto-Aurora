package transport

// CreateItemRequest — as tags validate:"..." são conferidas por
// httputil.Validate no handler (blueprint: todo módulo valida a entrada
// por struct-tag, nunca só confia no domínio).
type CreateItemRequest struct {
	Title       string `json:"title" validate:"required,max=200"`
	Description string `json:"description" validate:"max=2000"`
}

type ItemResponse struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Status      string `json:"status"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}
