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
