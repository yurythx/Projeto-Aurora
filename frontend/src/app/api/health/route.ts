import { NextResponse } from "next/server";

import { BACKEND_INTERNAL_URL } from "@/lib/env";

// GET /api/health — ponte server-to-server pro /ready do backend Go.
//
// Rota PRÓPRIA, fora do proxy BFF genérico (app/api/backend/[...path]/
// route.ts), de propósito: aquele proxy sempre monta o alvo como
// /api/{path} (rotas de negócio versionadas, ex. /api/v1/integrations) e
// sempre injeta um Authorization: Bearer — nenhuma das duas coisas se
// aplica aqui. /ready não vive sob /api/v1 (é o mesmo endpoint que o
// HEALTHCHECK do Docker Compose chama direto, sem token nenhum, e
// continua público de propósito: um orquestrador de containers nunca tem
// uma sessão de usuário pra apresentar). Tentar buscá-lo através do proxy
// genérico só resultava em 404 (bug real: GET /api/backend/health, que o
// proxy nunca conseguiria mapear pra nada que exista no backend).
//
// Reformata o {postgres, rabbitmq} bruto do /ready no formato que o
// painel de Monitoramento (PlatformMonitoringDashboard) espera.
export async function GET() {
  let backendResponse: Response;
  try {
    backendResponse = await fetch(`${BACKEND_INTERNAL_URL}/ready`, {
      cache: "no-store",
      signal: AbortSignal.timeout(5000),
    });
  } catch {
    return NextResponse.json(
      {
        data: { status: "unhealthy", timestamp: new Date().toISOString(), services: {} },
        error: { code: "DEPENDENCY_UNAVAILABLE", message: "A API não respondeu ao /ready." },
      },
      { status: 503 },
    );
  }

  let checks: Record<string, string> = {};
  try {
    const body: { data: Record<string, string> | null } = await backendResponse.json();
    checks = body.data ?? {};
  } catch {
    // Resposta ilegível — segue com checks vazio; o status geral abaixo
    // ainda reflete o backendResponse.ok/status recebido.
  }

  const services = Object.fromEntries(
    Object.entries(checks).map(([name, status]) => [name, { status }]),
  );
  const allOk = Object.values(checks).every((status) => status === "ok");

  return NextResponse.json(
    {
      data: {
        status: backendResponse.ok && allOk ? "ok" : "degraded",
        timestamp: new Date().toISOString(),
        services,
      },
      error: null,
    },
    { status: backendResponse.status },
  );
}
