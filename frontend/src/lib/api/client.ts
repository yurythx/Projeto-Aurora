// Cliente tipado para Client Components — sempre chama o proxy BFF na
// mesma origem (/api/backend/*), nunca a API Go diretamente, para que o
// bearer token nunca precise chegar ao JavaScript executado no navegador
// (§30).

import type { ZodType } from "zod";

import { APP_URL } from "@/lib/env";

interface ErrorBody {
  code: string;
  message: string;
}

interface Envelope<T> {
  data: T | null;
  error: ErrorBody | null;
  meta?: unknown;
}

export class ApiError extends Error {
  readonly status: number;
  readonly code: string;

  constructor(status: number, code: string, message: string) {
    super(message);
    this.name = "ApiError";
    this.status = status;
    this.code = code;
  }
}

async function request<T>(
  path: string,
  init?: RequestInit,
  schema?: ZodType<T>,
): Promise<{ data: T; meta?: unknown }> {
  let cleanPath = path.startsWith("/") ? path.slice(1) : path;
  if (cleanPath.startsWith("api/")) {
    cleanPath = cleanPath.slice(4);
  }
  if (cleanPath.startsWith("backend/")) {
    cleanPath = cleanPath.slice(8);
  }
  const isFormData = typeof FormData !== "undefined" && init?.body instanceof FormData;
  // No browser é same-origin (""); em SSR precisa da origem absoluta da app
  // (parametrizada em lib/env.ts, default acompanhando a porta atual).
  const baseUrl = typeof window !== "undefined" ? "" : APP_URL;

  const res = await fetch(`${baseUrl}/api/backend/${cleanPath}`, {
    ...init,
    headers: {
      ...(isFormData ? {} : { "Content-Type": "application/json" }),
      ...(init?.headers ?? {}),
    },
  });

  // 401 do proxy BFF = sessão morta/expirada (ver lib/auth/tokenState.ts).
  // Sem isto, uma tela com polling SWR só acumulava erros silenciosos e o
  // usuário nunca era levado de volta ao login. RBAC negado usa 403, não
  // 401, então redirecionar aqui é seguro. Navegação DURA de propósito
  // (location.replace, não router.push): descarta todo o estado de cliente
  // da tela expirada e reexecuta o middleware; replace() não deixa a tela
  // morta no histórico do back.
  if (res.status === 401 && typeof window !== "undefined" && window.location.pathname !== "/login") {
    const target = new URL("/login", window.location.origin);
    target.searchParams.set("callbackUrl", window.location.pathname + window.location.search);
    // Este 401 só acontece pra quem já tinha uma sessão (o proxy BFF exige
    // token pra sequer tentar a chamada) — sempre "sessão expirou", nunca
    // "nunca logou" (ver o mesmo aviso em LoginCard.tsx).
    target.searchParams.set("reason", "session_expired");
    window.location.replace(target.toString());
  }

  let json: Envelope<T>;
  try {
    json = await res.json();
  } catch {
    throw new ApiError(
      res.status,
      "INVALID_RESPONSE",
      "O servidor retornou uma resposta ilegível.",
    );
  }

  if (!res.ok || json.error) {
    throw new ApiError(
      res.status,
      json.error?.code ?? "UNKNOWN_ERROR",
      json.error?.message ?? "Algo deu errado. Tente novamente.",
    );
  }

  // Validação de contrato (S-07): idêntica à de lib/api/server.ts.
  if (schema) {
    const parsed = schema.safeParse(json.data);
    if (!parsed.success) {
      throw new ApiError(
        502,
        "INVALID_RESPONSE",
        "A resposta da API não corresponde ao formato esperado.",
      );
    }
    return { data: parsed.data, meta: json.meta };
  }

  return { data: json.data as T, meta: json.meta };
}

export const apiClient = {
  get: <T>(path: string, schema?: ZodType<T>) => request<T>(path, { method: "GET" }, schema),
  post: <T>(path: string, body?: unknown) =>
    request<T>(path, {
      method: "POST",
      body: body !== undefined ? JSON.stringify(body) : undefined,
    }),
  // postForm: caminho separado de post() pra um corpo multipart/form-data
  // (ex.: upload de arquivo) — nunca passa por JSON.stringify, e nunca
  // define Content-Type manualmente: o browser precisa gerar esse header
  // sozinho a partir do FormData, incluindo o boundary multipart, que
  // request() (Content-Type: application/json fixo) sobrescreveria e
  // quebraria a requisição inteira.
  postForm: <T>(path: string, form: FormData) =>
    request<T>(path, { method: "POST", body: form, headers: {} }),
  patch: <T>(path: string, body?: unknown) =>
    request<T>(path, {
      method: "PATCH",
      body: body !== undefined ? JSON.stringify(body) : undefined,
    }),
  // put/delete: nenhum módulo atual os usa ainda — mantidos por
  // completude do cliente HTTP genérico, mesmo formato de corpo/erro que
  // post/patch já têm.
  put: <T>(path: string, body?: unknown) =>
    request<T>(path, {
      method: "PUT",
      body: body !== undefined ? JSON.stringify(body) : undefined,
    }),
  delete: <T>(path: string) => request<T>(path, { method: "DELETE" }),
};
