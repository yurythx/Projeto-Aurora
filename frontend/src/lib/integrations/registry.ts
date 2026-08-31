// Registro de integrações do Projeto Aurora.

export interface IntegrationRegistryEntry {
  description: string;
  testPath: string;
}

export const integrationRegistry: Record<string, IntegrationRegistryEntry> = {
  "example-service": {
    description:
      "Executa uma verificação assíncrona de conectividade com o serviço de exemplo da plataforma.",
    testPath: "v1/integrations/example-service/test",
  },
};
