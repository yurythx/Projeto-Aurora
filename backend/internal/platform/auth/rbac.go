package auth

// Roles conhecidos pela plataforma. O Keycloak é a fonte da verdade
// para a atribuição de roles a usuários — estas constantes existem só para
// que as checagens de autorização no código não espalhem strings literais
// (evitando erro de digitação silencioso).
const (
	RoleUser               RoleName = "aurora-user"
	RoleAdmin              RoleName = "aurora-admin"
	RoleIntegrationManager RoleName = "aurora-integration-manager"
	RoleAuditor            RoleName = "aurora-auditor"
)

type RoleName = string

// Permission é uma capacidade granular verificada por RequirePermission,
// para handlers em que "o chamador tem um destes roles" não é específico
// o bastante. O mapeamento abaixo é deliberadamente simples — uma única
// tabela estática role -> permissões — e é o ponto de extensão caso a
// plataforma precise no futuro de regras por recurso ou baseadas em
// atributos (attribute-based access control).
type Permission string

const (
	PermUsersRead          Permission = "users:read"
	PermUsersManage        Permission = "users:manage"
	PermIntegrationsRead   Permission = "integrations:read"
	PermIntegrationsTest   Permission = "integrations:test"
	PermIntegrationsManage Permission = "integrations:manage"
	PermAuditRead          Permission = "audit:read"
	// PermFeatureFlagsManage não é concedida a nenhum role em
	// rolePermissions abaixo — só o aurora-admin a possui, através do
	// atalho em HasPermission. Alternar feature flags em produção afeta todo
	// mundo imediatamente, então é deliberadamente restrito ao papel mais
	// privilegiado, sem meio-termo por role.
	PermFeatureFlagsManage Permission = "feature_flags:manage"
	// PermKeycloakManage, pelo mesmo motivo de PermFeatureFlagsManage
	// acima, também não é concedida a nenhum role em rolePermissions — só
	// o aurora-admin (via HasPermission). Ver a configuração dinâmica do
	// Keycloak em internal/platform/keycloakconfig — errar aqui derruba
	// a autenticação de TODA a plataforma, então esta permissão é
	// deliberadamente mais restrita do que PermIntegrationsManage.
	PermKeycloakManage Permission = "keycloak:manage"
)

// rolePermissions concede ao aurora-admin toda permissão implicitamente
// (verificado à parte em HasPermission) e dá aos demais roles o conjunto
// mínimo implicado pelo próprio nome — ex.: um "aurora-auditor" só pode ler
// (audit, users, integrations), nunca escrever.
var rolePermissions = map[RoleName][]Permission{
	RoleUser: {
		PermUsersRead,
	},
	RoleIntegrationManager: {
		PermIntegrationsRead,
		PermIntegrationsTest,
		PermIntegrationsManage,
	},
	RoleAuditor: {
		PermAuditRead,
		PermUsersRead,
		PermIntegrationsRead,
	},
}

// HasPermission reporta se os roles de identity concedem permission.
// aurora-admin sempre tem toda permissão, independente do mapa acima.
func HasPermission(identity Identity, permission Permission) bool {
	if identity.HasRole(RoleAdmin) {
		return true
	}
	for _, role := range identity.Roles {
		for _, p := range rolePermissions[role] {
			if p == permission {
				return true
			}
		}
	}
	return false
}
