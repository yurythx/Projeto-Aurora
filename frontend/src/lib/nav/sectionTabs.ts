import type { SectionTab } from "@/components/layout/SectionTabs";

// Sub-navegação por seção — uma fonte de verdade, consumida tanto pelas
// páginas (que renderizam <SectionTabs>) quanto pela Sidebar (para saber
// que a seção "Contratos" cobre /contratos e /contratos/cadastro).

export const CONTRATOS_TABS: SectionTab[] = [
  { href: "/contratos", label: "Liquidação" },
  { href: "/contratos/cadastro", label: "Cadastro" },
];

export const DIARIO_TABS: SectionTab[] = [
  { href: "/diario-oficial", label: "Portal" },
  { href: "/diario", label: "Busca" },
  { href: "/pessoal", label: "Atos de Pessoal" },
  { href: "/diario/revisao", label: "Revisão" },
];
