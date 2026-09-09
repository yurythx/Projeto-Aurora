// Registro de integrações do Projeto Aurora — o ponto de extensão onde um
// NOVO módulo de negócio real regista a descrição e (se tiver um endpoint
// de teste de verdade no backend) o testPath da sua própria integração,
// pra IntegrationCard (app/(protected)/integracoes/[key]/page.tsx) achar
// aqui em vez de inventar algo. Sem uma entrada aqui, a tela ainda
// funciona — description cai pro texto genérico e o botão "Testar
// conexão" nem aparece (ver o `{testPath && (...)}` em IntegrationCard).
//
// Achado de auditoria: havia uma entrada "example-service" apontando pra
// v1/integrations/example-service/test — uma rota que NUNCA existiu no
// backend (o endpoint genérico de teste de integração foi avaliado e
// deliberadamente não construído, por ser especulativo demais sem um
// provedor real por trás — ver o rate limiter TestJob removido de
// internal/app/dependencies.go pelo mesmo motivo). Como a tabela
// `integrations` nunca teve uma linha com essa key, o botão nunca chegou
// a aparecer pra ninguém clicar — mas ficava como uma armadilha pronta
// pro dia em que alguém seedasse uma integração de verdade com esse
// nome. Removida; registre aqui só quando o backend tiver o endpoint de
// teste correspondente implementado e real.
export interface IntegrationRegistryEntry {
  description: string;
  testPath: string;
}

export const integrationRegistry: Record<string, IntegrationRegistryEntry> = {};
