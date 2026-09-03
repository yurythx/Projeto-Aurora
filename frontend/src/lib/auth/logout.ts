import { signOut } from "next-auth/react";

// Logout completo (RP-Initiated Logout — §30): 1) busca a URL de logout do
// Keycloak ENQUANTO a sessão local ainda existe (precisa do id_token_hint,
// lido server-side em /api/auth/keycloak-logout-url); 2) só então limpa a
// sessão local (signOut, sem redirecionar ainda); 3) navega o navegador
// até o Keycloak para encerrar a sessão lá também. Chamar apenas signOut()
// deixaria a sessão viva no provedor de identidade — outra aba, ou um
// login silencioso, reautenticaria sem pedir credenciais de novo. Sessões
// do login local não têm id_token (ver next-auth.d.ts) — a chamada
// funciona nesse caso mesmo assim, resolvendo para logoutUrl = "/".
//
// Compartilhado por UserMenu (dashboard) e PublicShell (páginas públicas).
export async function fullSignOut() {
  let logoutUrl = "/";
  try {
    const res = await fetch("/api/auth/keycloak-logout-url");
    const data: { url: string } = await res.json();
    logoutUrl = data.url;
  } catch {
    // Se a chamada falhar, ainda completamos o logout local abaixo — melhor
    // encerrar só a sessão local do que travar o usuário logado.
  }
  await signOut({ redirect: false });
  window.location.href = logoutUrl;
}
