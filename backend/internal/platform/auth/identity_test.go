package auth

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

func TestActorUUID(t *testing.T) {
	t.Run("sem identidade no contexto", func(t *testing.T) {
		id, subject, ok := ActorUUID(context.Background())
		if id != nil || subject != "" || ok {
			t.Errorf("esperava (nil, \"\", false), got (%v, %q, %v)", id, subject, ok)
		}
	})

	t.Run("subject UUID válido", func(t *testing.T) {
		want := uuid.New()
		ctx := WithIdentity(context.Background(), Identity{Subject: want.String()})
		id, subject, ok := ActorUUID(ctx)
		if !ok || id == nil || *id != want || subject != want.String() {
			t.Errorf("esperava (%v, ok), got (%v, %q, %v)", want, id, subject, ok)
		}
	})

	t.Run("subject não-UUID devolve o bruto e ok=false (autoria a logar)", func(t *testing.T) {
		ctx := WithIdentity(context.Background(), Identity{Subject: "keycloak:abc-123"})
		id, subject, ok := ActorUUID(ctx)
		if ok || id != nil {
			t.Errorf("subject não-UUID não deveria converter: id=%v ok=%v", id, ok)
		}
		if subject != "keycloak:abc-123" {
			t.Errorf("subject bruto deveria ser propagado para o log, got %q", subject)
		}
	})
}
