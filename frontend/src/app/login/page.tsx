import { ShieldCheck } from "lucide-react";
import { connection } from "next/server";
import { cookies } from "next/headers";
import Link from "next/link";
import { Suspense, type CSSProperties } from "react";

import { LoginCard } from "@/components/auth/LoginCard";
import { Logo } from "@/components/ui/Logo";
import { Seal } from "@/components/ui/Seal";
import { ThemeToggle } from "@/components/ui/ThemeToggle";

// O painel de marca descreve exatamente o que a plataforma faz para um
// fiscal de contrato — sem linguagem de vendedor ("enterprise", "a
// tecnologia mais avançada do mercado"), que a versão anterior tinha
// herdado de um template.
const capabilities = [
  "As 6 etapas da IN SCL 01/2019 em um quadro só, do ofício ao arquivamento.",
  "Ofício, Ordem de Serviço e Relatório de Fiscalização gerados prontos para assinar.",
  "Aviso de SLA estourado e de certidão vencida antes de travar o processo.",
  "Diário Oficial de Rondonópolis lido e cruzado com os seus contratos.",
];

export default async function LoginPage() {
  // Força renderização dinâmica — necessário para que o
  // Content-Security-Policy com nonce (proxy.ts) seja aplicado
  // corretamente; veja o comentário equivalente em app/page.tsx.
  await connection();

  // Mesmo cookie "nova-theme" que o dashboard lê — o login também tem um
  // ThemeToggle (canto superior direito), então precisa do mesmo
  // tratamento sem-flash.
  const cookieStore = await cookies();
  const themeCookie = cookieStore.get("nova-theme")?.value;
  const initialTheme = themeCookie === "dark" || themeCookie === "light" ? themeCookie : undefined;

  return (
    <div className="flex min-h-screen">
      {/* Painel de marca — só no desktop (lg+). Abaixo disso a tela vira
          só o formulário, centralizado. O painel é sempre escuro
          (bg-brand-panel) em qualquer tema, então o texto é fixado em
          branco (não text-primary-foreground, que no tema escuro fica
          quase preto) e a tinta de carimbo do selo/marcadores no tom
          claro (--seal). */}
      <div
        className="relative hidden overflow-hidden bg-brand-panel text-white lg:flex lg:w-[44%] lg:flex-col lg:justify-between lg:p-12"
        style={{ "--seal": "#d1524a" } as CSSProperties}
      >
        {/* Assinatura: o selo como marca d'água, sangrando pelo canto. */}
        <Seal
          size={520}
          decorative
          className="pointer-events-none absolute -bottom-40 -right-40 text-white/[0.06]"
        />

        <div className="relative flex items-center gap-2.5 text-lg font-semibold">
          <Logo size={30} />
          Projeto Nova
        </div>

        <div className="relative flex flex-col gap-6">
          <p className="dateline text-white/60">Prefeitura Municipal de Rondonópolis</p>
          <h1 className="max-w-md text-3xl font-semibold leading-tight text-white">
            Fiscalização de contratos, do ofício ao arquivamento.
          </h1>
          <p className="max-w-sm text-sm text-white/75">
            Um painel para acompanhar cada demanda mensal pelas seis etapas da IN SCL 01/2019 — com
            os documentos, os prazos e o Diário Oficial no mesmo lugar.
          </p>
          <ul className="mt-1 flex flex-col gap-3 text-sm text-white/90">
            {capabilities.map((item) => (
              <li key={item} className="flex gap-3">
                <span aria-hidden="true" className="mt-2 h-1.5 w-1.5 shrink-0 rounded-full bg-seal" />
                {item}
              </li>
            ))}
          </ul>
        </div>

        <p className="relative flex items-center gap-2 text-xs text-white/55">
          <ShieldCheck size={14} aria-hidden="true" />
          Entrada por senha ou SSO corporativo · auditoria de toda ação sensível
        </p>
      </div>

      {/* Painel do formulário */}
      <div className="relative flex flex-1 flex-col items-center justify-center p-6">
        <div className="absolute right-4 top-4">
          <ThemeToggle initialTheme={initialTheme} />
        </div>

        <Link href="/" className="mb-10 flex items-center gap-2 text-lg font-semibold lg:hidden">
          <Logo size={30} />
          Projeto Nova
        </Link>

        <Suspense fallback={null}>
          <LoginCard />
        </Suspense>

        <Link href="/" className="mt-10 text-xs text-muted hover:text-foreground">
          ← Voltar para a página inicial
        </Link>
      </div>
    </div>
  );
}
