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
	// RoleFiscal é o fiscal de contrato: opera o Kanban de liquidação
	// (mover card de etapa, anexar comprovantes de compliance, gerar os
	// documentos oficiais) e lê contratos — nada além disso. Antes dele, a
	// persona do fiscal só cabia em nova-admin ou no integration-manager,
	// ambos privilegiados demais.
	RoleFiscal RoleName = "nova-fiscal"
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

	// PermDiarioOficialRead/PermDiarioOficialManage controlam o
	// monitoramento real do Diário Oficial (MVP via DJEN): ler termos
	// monitorados/publicações casadas vs. cadastrar/remover um termo.
	PermDiarioOficialRead   Permission = "diario_oficial:read"
	PermDiarioOficialManage Permission = "diario_oficial:manage"
	// PermContratosRead/PermContratosManage controlam o módulo de
	// contratos municipais: ler contratos/kanban vs. criar/atualizar
	// contratos e vincular publicações do Diário Oficial.
	PermContratosRead   Permission = "contratos:read"
	PermContratosManage Permission = "contratos:manage"
	// PermDemandsRead/PermDemandsManage controlam o Kanban de demandas
	// mensais (IN SCL 01/2019): ver o quadro/checklist/histórico vs. mover
	// um card de etapa e anexar documentos de compliance (ações fiscais).
	PermDemandsRead   Permission = "demands:read"
	PermDemandsManage Permission = "demands:manage"
	// PermFeatureFlagsManage não é concedida a nenhum role em
	// rolePermissions abaixo — só o nix-admin a possui, através do atalho
	// em HasPermission. Alternar feature flags em produção afeta todo
	// mundo imediatamente, então é deliberadamente restrito ao papel mais
	// privilegiado, sem meio-termo por role.
	PermFeatureFlagsManage Permission = "feature_flags:manage"
)

// rolePermissions concede ao nova-admin toda permissão implicitamente
// (verificado à parte em HasPermission) e dá aos demais roles o conjunto
// mínimo implicado pelo próprio nome — ex.: um
// "nova-auditor" só pode ler (audit, users, integrations, contratos), nunca escrever.
var rolePermissions = map[RoleName][]Permission{
	RoleUser: {
		PermUsersRead,
	},
	RoleIntegrationManager: {
		PermIntegrationsRead,
		PermIntegrationsTest,
		PermIntegrationsManage,
		PermDiarioOficialRead,
		PermDiarioOficialManage,
		PermContratosRead,
		PermContratosManage,
		PermDemandsRead,
		PermDemandsManage,
	},
	RoleAuditor: {
		PermAuditRead,
		PermUsersRead,
		PermIntegrationsRead,
		PermDiarioOficialRead,
		PermContratosRead,
		PermDemandsRead,
	},
	// O fiscal opera a liquidação e consulta contratos; não gerencia
	// contratos, integrações, usuários nem termos do Diário.
	RoleFiscal: {
		PermDemandsRead,
		PermDemandsManage,
		PermContratosRead,
	},
}

// Nota (restrição por etapa): hoje quem tem demands:manage move o card
// entre quaisquer etapas. Um controle mais fino — o fiscal executa as
// ações fiscais (etapas 1/3/4/5) mas NÃO confirma sozinho os handoffs de
// contabilidade (entrada/saída das etapas 2 e 6) — dependeria de uma
// política que ainda não foi decidida e de passar a identidade do ator
// para application.ValidateTransition (que já recebe a demanda e o
// horário). É o ponto de extensão natural quando essa regra existir.

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
