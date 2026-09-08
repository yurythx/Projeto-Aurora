import { act, render } from "@testing-library/react";
import { afterEach, describe, expect, it } from "vitest";

import { BrandingProvider } from "./BrandingContext";
import { resetBrandingStore, updateBrandingStore } from "./brandingStore";

// Achado de auditoria: branding.faviconUrl era um campo declarado desde
// sempre (tipo, default, persistido no cookie, campo no formulário de
// configurações) mas NADA o lia — o <link rel="icon"> nunca refletia a
// personalização. Este teste cobre o efeito que finalmente conecta os
// dois lados (ver o useEffect de favicon em BrandingContext.tsx).
//
// updateBrandingStore/resetBrandingStore vivem fora do React (cookie +
// módulo, não escopados por instância de <BrandingProvider>) — chamados
// direto aqui, sem precisar de um componente consumidor/clique de botão
// só pra disparar o mesmo efeito.
const MARKER_SELECTOR = 'link[rel="icon"][data-aurora-branding-favicon]';

describe("BrandingProvider — favicon dinâmico (white-label)", () => {
  afterEach(() => {
    act(() => resetBrandingStore());
    document.querySelectorAll(MARKER_SELECTOR).forEach((el) => el.remove());
  });

  it("cria um <link rel=icon> com a URL https:// configurada", () => {
    render(<BrandingProvider>{null}</BrandingProvider>);

    act(() => updateBrandingStore({ faviconUrl: "https://exemplo.gov.br/favicon.png" }));

    const link = document.querySelector<HTMLLinkElement>(MARKER_SELECTOR);
    expect(link).not.toBeNull();
    expect(link!.href).toBe("https://exemplo.gov.br/favicon.png");
  });

  it("rejeita esquemas perigosos (javascript:) — nenhum <link> é criado", () => {
    render(<BrandingProvider>{null}</BrandingProvider>);

    act(() => updateBrandingStore({ faviconUrl: "javascript:alert(1)" }));

    expect(document.querySelector(MARKER_SELECTOR)).toBeNull();
  });

  it("remove o <link> customizado quando a URL é limpa (volta ao favicon padrão do arquivo estático)", () => {
    render(<BrandingProvider>{null}</BrandingProvider>);

    act(() => updateBrandingStore({ faviconUrl: "https://exemplo.gov.br/favicon.png" }));
    expect(document.querySelector(MARKER_SELECTOR)).not.toBeNull();

    act(() => updateBrandingStore({ faviconUrl: "" }));
    expect(document.querySelector(MARKER_SELECTOR)).toBeNull();
  });
});
