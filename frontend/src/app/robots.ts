import type { MetadataRoute } from "next";

import { APP_URL } from "@/lib/env";

// Disallow explícito das seções autenticadas — um crawler nunca passa da
// tela de login de qualquer forma (proxy.ts redireciona), mas listar
// isso aqui evita tentativa de rastreio/ruído de log por um bot bem-
// comportado, e deixa a intenção (só / e /sobre são públicas) explícita
// em vez de implícita.
const baseUrl = APP_URL;

export default function robots(): MetadataRoute.Robots {
  return {
    rules: {
      userAgent: "*",
      allow: ["/", "/sobre", "/padroes", "/acessibilidade", "/privacidade"],
      disallow: ["/dashboard", "/integracoes", "/configuracao", "/seguranca", "/login"],
    },
    sitemap: `${baseUrl}/sitemap.xml`,
  };
}
