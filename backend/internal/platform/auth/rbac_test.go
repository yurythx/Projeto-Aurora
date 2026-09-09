package auth

import "testing"

func TestHasPermission_AdminHasEverything(t *testing.T) {
	admin := Identity{Roles: []string{RoleAdmin}}

	for _, p := range []Permission{
		PermUsersRead, PermUsersManage, PermIntegrationsRead, PermIntegrationsTest,
		PermIntegrationsManage, PermAuditRead, PermFeatureFlagsManage, PermKeycloakManage,
	} {
		if !HasPermission(admin, p) {
			t.Errorf("expected admin to have permission %q", p)
		}
	}
}

// TestHasPermission_KeycloakManageIsAdminOnly cobre o mesmo raciocínio de
// PermFeatureFlagsManage: alterar a configuração do Keycloak em runtime
// (ver internal/platform/keycloakconfig) afeta a autenticação de TODA a
// plataforma, então nenhum role além de aurora-admin pode receber essa
// permissão — nem o aurora-integration-manager, que já gerencia
// integrações externas.
func TestHasPermission_KeycloakManageIsAdminOnly(t *testing.T) {
	for _, role := range []RoleName{RoleUser, RoleIntegrationManager, RoleAuditor} {
		identity := Identity{Roles: []string{role}}
		if HasPermission(identity, PermKeycloakManage) {
			t.Errorf("role %q não deveria ter keycloak:manage", role)
		}
	}
}

func TestHasPermission_RoleGrantsOnlyItsPermissions(t *testing.T) {
	manager := Identity{Roles: []string{RoleIntegrationManager}}

	if !HasPermission(manager, PermIntegrationsTest) {
		t.Error("expected integration manager to have integrations:test")
	}
	if HasPermission(manager, PermUsersManage) {
		t.Error("expected integration manager NOT to have users:manage")
	}
}

func TestHasPermission_NoRolesDeniesEverything(t *testing.T) {
	anon := Identity{}
	if HasPermission(anon, PermUsersRead) {
		t.Error("expected an identity with no roles to have no permissions")
	}
}

// TestHasPermission_UserRoleIsSelfServiceOnly trava o gap G-10: o papel
// base aurora-user não pode ter NENHUMA permissão de diretório — ele
// enxerga só a si mesmo via GET /api/v1/me (que exige apenas
// autenticação). Listar/consultar terceiros expõe PII (e-mail) e passou a
// exigir aurora-auditor ou aurora-admin.
func TestHasPermission_UserRoleIsSelfServiceOnly(t *testing.T) {
	user := Identity{Roles: []string{RoleUser}}
	for _, p := range []Permission{
		PermUsersRead, PermUsersManage, PermIntegrationsRead, PermIntegrationsManage,
		PermIntegrationsTest, PermAuditRead, PermFeatureFlagsManage, PermKeycloakManage,
	} {
		if HasPermission(user, p) {
			t.Errorf("aurora-user não deveria ter %q", p)
		}
	}
}

// TestHasPermission_AuditorCanReadUsers garante que a leitura do
// diretório de usuários continua disponível para quem tem função de
// controle (aurora-auditor) depois do estreitamento do gap G-10.
func TestHasPermission_AuditorCanReadUsers(t *testing.T) {
	auditor := Identity{Roles: []string{RoleAuditor}}
	if !HasPermission(auditor, PermUsersRead) {
		t.Error("aurora-auditor deveria manter users:read")
	}
	if HasPermission(auditor, PermUsersManage) {
		t.Error("aurora-auditor não deveria ter users:manage (só leitura)")
	}
}
