package auth

// Roles conhecidos pela plataforma. O Keycloak é a fonte da verdade
// para a atribuição de roles a usuários — estas constantes existem só para
// que as checagens de autorização no código não espalhem strings literais
// (evitando erro de digitação silencioso).
const (
	RoleUser               RoleName = "nova-user"
	RoleAdmin              RoleName = "nova-admin"
	RoleIntegrationManager RoleName = "projeto-nova-integration-manager"
	RoleAuditor            RoleName = "nova-auditor"
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
	// rolePermissions abaixo — só o nova-admin a possui, através do atalho
	// em HasPermission. Alternar feature flags em produção afeta todo
	// mundo imediatamente, então é deliberadamente restrito ao papel mais
	// privilegiado, sem meio-termo por role.
	PermFeatureFlagsManage Permission = "feature_flags:manage"
)

// rolePermissions concede ao nova-admin toda permissão implicitamente
// (verificado à parte em HasPermission) e dá aos demais roles o conjunto
// mínimo implicado pelo próprio nome — ex.: um "nova-auditor" só pode ler
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
// nova-admin sempre tem toda permissão, independente do mapa acima.
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
