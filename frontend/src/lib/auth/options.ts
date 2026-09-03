import type { NextAuthOptions, Session } from "next-auth";
import type { JWT } from "next-auth/jwt";
import CredentialsProvider from "next-auth/providers/credentials";
import KeycloakProvider from "next-auth/providers/keycloak";

import { BACKEND_INTERNAL_URL } from "@/lib/env";

// Env somente de servidor — nunca com prefixo NEXT_PUBLIC_, nunca enviada
// ao navegador (§30: segredos nunca em NEXT_PUBLIC_*).
const issuer = process.env.KEYCLOAK_ISSUER_URL;
const clientId = process.env.KEYCLOAK_FRONTEND_CLIENT_ID;
const clientSecret = process.env.KEYCLOAK_FRONTEND_CLIENT_SECRET;

if (!issuer || !clientId || !clientSecret) {
  console.warn(
    "Missing Keycloak frontend OIDC configuration: KEYCLOAK_ISSUER_URL, " +
      "KEYCLOAK_FRONTEND_CLIENT_ID and KEYCLOAK_FRONTEND_CLIENT_SECRET are not all present. Keycloak login is disabled.",
  );
}

// O mesmo endereço interno que o proxy BFF usa (ver
// app/api/backend/[...path]/route.ts) — o login local também é uma
// chamada server-to-server ao backend Go, nunca exposta ao navegador.
// Parametrizado em lib/env.ts (S-04).
const backendInternalURL = BACKEND_INTERNAL_URL;


interface KeycloakTokenResponse {
  access_token: string;
  refresh_token?: string;
  id_token?: string;
  expires_in: number;
  error?: string;
}

/** Renova um access token expirado através do endpoint de token do
 * Keycloak, usando o refresh_token guardado na sessão. Só se aplica a
 * sessões que vieram do Keycloak — o login local (ver
 * localLoginAuthorize) não tem conceito de refresh token, uma sessão
 * local expirada simplesmente exige logar de novo. */
async function refreshAccessToken(token: JWT): Promise<JWT> {
  try {
    const response = await fetch(`${issuer}/protocol/openid-connect/token`, {
      method: "POST",
      headers: { "Content-Type": "application/x-www-form-urlencoded" },
      body: new URLSearchParams({
        client_id: clientId!,
        client_secret: clientSecret!,
        grant_type: "refresh_token",
        refresh_token: token.refreshToken as string,
      }),
      // Sem timeout, um Keycloak lento/pendurado trava o callback jwt (que
      // roda em toda requisição que toca a sessão).
      signal: AbortSignal.timeout(5000),
    });

    const refreshed: KeycloakTokenResponse = await response.json();
    if (!response.ok || refreshed.error || !refreshed.access_token) {
      throw refreshed;
    }
    // expires_in ausente/NaN viraria accessTokenExpires=NaN e, como
    // `Date.now() < NaN` é sempre false, o token entraria em loop de
    // refresh a cada requisição, martelando o Keycloak.
    const ttlSeconds = Number(refreshed.expires_in);
    if (!Number.isFinite(ttlSeconds) || ttlSeconds <= 0) {
      throw new Error(`expires_in inválido: ${refreshed.expires_in}`);
    }

    return {
      ...token,
      accessToken: refreshed.access_token,
      accessTokenExpires: Date.now() + ttlSeconds * 1000,
      refreshToken: refreshed.refresh_token ?? token.refreshToken,
      error: undefined,
    };
  } catch {
    // Marca o erro na sessão em vez de lançar — o chamador (callback jwt)
    // segue com um token expirado + error="RefreshAccessTokenError", e é
    // esse campo que o middleware/proxy.ts usa para decidir redirecionar
    // para /login. NÃO logamos `err` (pode conter eco do refresh_token).
    console.error("Falha ao renovar o access token do Keycloak");
    return { ...token, error: "RefreshAccessTokenError" };
  }
}

interface LocalLoginResponse {
  data: {
    access_token: string;
    expires_at: string;
    user: { id: string; username: string; email: string };
  } | null;
  error: { code: string; message: string } | null;
}

export const authOptions: NextAuthOptions = {
  providers: [
    ...(issuer && clientId && clientSecret
      ? [
          KeycloakProvider({
            issuer,
            clientId,
            clientSecret,
            // As checks padrão do KeycloakProvider já incluem "pkce" e "state"
            // — Authorization Code + PKCE conforme §30, sem configuração extra
            // necessária.
          }),
        ]
      : []),

    // Login local (§ Sistema de Login Local) — um caminho ADICIONAL ao
    // Keycloak, nunca um substituto. Chama o mesmo endpoint que qualquer
    // outro cliente HTTP chamaria (POST /api/v1/auth/login no backend
    // Go); este provider só faz a ponte entre o formulário de
    // usuário/senha e a sessão do NextAuth. Se LOCAL_AUTH_ENABLED=false
    // no backend, o endpoint responde 404 e authorize() retorna null —
    // o botão de login local continua aparecendo no frontend, mas
    // qualquer tentativa falha com a mensagem genérica de credenciais
    // inválidas (não vaza se o recurso está desligado ou se a senha
    // estava errada).
    CredentialsProvider({
      id: "local",
      name: "Local",
      credentials: {
        username: { label: "Usuário", type: "text" },
        password: { label: "Senha", type: "password" },
      },
      async authorize(credentials) {
        if (!credentials?.username || !credentials?.password) {
          return null;
        }

        // authorize() SÓ pode retornar um usuário quando o backend confirma
        // a credencial. Qualquer outro desfecho — backend fora do ar, JSON
        // ilegível, 401/403/404/500 — é `null` (o NextAuth converte em
        // "credenciais inválidas", sem vazar o motivo). Nunca há "sessão
        // demo": código que decide autenticação não pode ter modo de
        // conveniência (era um bypass — senha errada logava como demo-user).
        let res: Response;
        try {
          res = await fetch(`${backendInternalURL}/api/v1/auth/login`, {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({
              username: credentials.username,
              password: credentials.password,
            }),
            signal: AbortSignal.timeout(5000),
          });
        } catch (err) {
          console.error("Local login: backend inacessível — negando", err);
          return null;
        }

        let body: LocalLoginResponse;
        try {
          body = await res.json();
        } catch {
          return null;
        }

        if (!res.ok || !body.data) {
          return null;
        }

        return {
          id: body.data.user.id,
          name: body.data.user.username,
          email: body.data.user.email,
          accessToken: body.data.access_token,
          accessTokenExpires: new Date(body.data.expires_at).getTime(),
        };
      },
    }),
  ],

  // Estratégia de sessão em JWT: a sessão é um cookie criptografado,
  // HttpOnly, SameSite=Lax, marcado Secure automaticamente sempre que
  // NEXTAUTH_URL é https:// (padrão do next-auth — §30) — nunca
  // localStorage ou sessionStorage. O access token bruto vive só dentro
  // deste token criptografado no lado do servidor, nunca no objeto
  // `session` exposto a Client Components — ver o callback session abaixo.
  //
  // Usando deliberadamente o nome/opções de cookie padrão do next-auth
  // aqui, em vez de um nome customizado: toda chamada de getToken() do
  // lado do servidor (proxy.ts, o proxy BFF do backend) precisa concordar
  // exatamente sobre qual nome de cookie ler, e um nome customizado é mais
  // um lugar onde as coisas podem silenciosamente sair de sincronia e
  // quebrar a autenticação.
  session: { strategy: "jwt" },

  callbacks: {
    async jwt({ token, account, user }) {
      if (account?.provider === "keycloak") {
        token.accessToken = account.access_token;
        token.accessTokenExpires = account.expires_at ? account.expires_at * 1000 : undefined;
        token.refreshToken = account.refresh_token;
        token.idToken = account.id_token;
      } else if (account?.provider === "local" && user) {
        token.accessToken = user.accessToken;
        token.accessTokenExpires = user.accessTokenExpires;
        token.refreshToken = undefined;
        token.idToken = undefined;
      }

      if (token.accessToken && typeof token.accessToken === "string") {
        try {
          const parts = (token.accessToken as string).split(".");
          if (parts.length === 3 && parts[1]) {
            const payloadStr = Buffer.from(parts[1], "base64url").toString("utf-8");
            const payload = JSON.parse(payloadStr);
            const rolesSet = new Set<string>();

            if (Array.isArray(payload.roles)) {
              payload.roles.forEach((r: string) => rolesSet.add(r));
            }
            if (Array.isArray(payload.realm_access?.roles)) {
              payload.realm_access.roles.forEach((r: string) => rolesSet.add(r));
            }
            if (clientId && payload.resource_access?.[clientId]?.roles) {
              payload.resource_access[clientId].roles.forEach((r: string) => rolesSet.add(r));
            }

            token.roles = Array.from(rolesSet);
          }
        } catch (err) {
          console.error("Failed to decode token roles", err);
        }
      }

      if (token.accessTokenExpires && Date.now() < (token.accessTokenExpires as number)) {
        return token;
      }

      if (token.refreshToken) {
        return refreshAccessToken(token);
      }

      return { ...token, error: "RefreshAccessTokenError" };
    },

    async session({ session, token }): Promise<Session> {
      session.user = session.user ?? {};
      session.user.roles = token.roles ?? [];
      session.error = token.error as string | undefined;
      return session;
    },
  },

  pages: {
    signIn: "/login",
  },
};
