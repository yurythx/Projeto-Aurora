import { notFound } from "next/navigation";

import { ErrorState } from "@/components/ui/ErrorState";
import { IntegrationCard } from "@/components/integrations/IntegrationCard";
import { integrationRegistry } from "@/lib/integrations/registry";
import { ApiError } from "@/lib/api/client";
import { serverApiGet } from "@/lib/api/server";
import type { Integration } from "@/types/api";

export default async function IntegrationDetailPage({
  params,
}: {
  params: Promise<{ key: string }>;
}) {
  const { key } = await params;

  let integration: Integration | undefined;
  let errorMessage: string | null = null;
  try {
    const { data } = await serverApiGet<Integration[]>("v1/integrations");
    integration = data.find((i) => i.key === key);
  } catch (err) {
    errorMessage = err instanceof ApiError ? err.message : "Falha ao carregar";
  }

  if (!errorMessage && !integration) {
    notFound();
  }

  const registryEntry = integrationRegistry[key];

  return (
    <div className="flex flex-col gap-6">
      {errorMessage && <ErrorState message={errorMessage} />}

      {integration && (
        <>
          <div>
            <p className="dateline">Integração</p>
            <h1 className="mt-2 text-2xl font-semibold">{integration.name}</h1>
            <p className="mt-1 text-sm text-muted">
              {registryEntry?.description ?? "Configuração e teste de conectividade desta integração."}
            </p>
          </div>

          <IntegrationCard integration={integration} testPath={registryEntry?.testPath} />
        </>
      )}
    </div>
  );
}
