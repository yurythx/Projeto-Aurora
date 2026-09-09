"use client";

import { useEffect } from "react";
import { usePathname, useRouter, useSearchParams } from "next/navigation";

import { useToast } from "@/components/notifications/ToastProvider";

// Mensagens amigáveis de autenticação (achado de auditoria: logout e
// login terminavam em silêncio total — nenhuma confirmação visual de que
// a ação funcionou). Cada fluxo carimba um parâmetro de busca no
// redirecionamento final; este componente só existe pra ler esse
// parâmetro, mostrar o toast certo, e então LIMPAR a URL (router.replace
// sem o parâmetro) — sem isso, um F5 ou "voltar" do navegador repetiria o
// mesmo toast indefinidamente.
//
// Montado uma vez em cada shell que pode ser o destino de um desses
// redirecionamentos: PublicShell (?logout=success, na home) e
// DashboardShell (?welcome=1, depois do login). Não renderiza nada.
export function AuthFlashToast() {
  const router = useRouter();
  const pathname = usePathname();
  const searchParams = useSearchParams();
  const { showToast } = useToast();

  // searchParams.toString() como dependência (não o objeto em si, que é
  // uma instância nova a cada render) — dispara de novo só quando a
  // query string muda de verdade.
  const query = searchParams.toString();

  useEffect(() => {
    const params = new URLSearchParams(query);
    const logout = params.get("logout");
    const welcome = params.get("welcome");

    if (logout === "success") {
      showToast({
        title: "Você saiu com segurança",
        description: "Sua sessão foi encerrada.",
        tone: "success",
      });
    } else if (welcome === "1") {
      showToast({
        title: "Bem-vindo(a) de volta!",
        description: "Login realizado com sucesso.",
        tone: "success",
      });
    } else {
      return;
    }

    params.delete("logout");
    params.delete("welcome");
    const rest = params.toString();
    router.replace(rest ? `${pathname}?${rest}` : pathname, { scroll: false });
    // eslint-disable-next-line react-hooks/exhaustive-deps -- só reage à query; router/pathname/showToast são estáveis o bastante aqui
  }, [query]);

  return null;
}
