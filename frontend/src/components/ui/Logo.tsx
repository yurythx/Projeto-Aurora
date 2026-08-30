// Marca do Projeto Nova — um SELO/CARIMBO institucional, não um monograma
// de app. Anéis concêntricos (o carimbo oficial), o "N" de Nova recortado
// em branco no centro e um traço em tinta de carimbo (#8a1c1c) na base: o
// mesmo motivo de "documento oficial" que atravessa o resto da interface.
// Azul institucional (mesma faixa da Topbar/paleta), nada de roxo — a
// versão anterior deste arquivo era, literalmente, a marca de outro
// produto.
//
// Sem <defs>/gradientes de propósito: quando dois <Logo> coexistem no DOM
// (ex.: variante desktop + variante mobile na tela de login, uma delas com
// display:none), IDs de gradiente repetidos fazem o navegador resolver o
// paint pelo primeiro <svg> — que, oculto, não pinta — e a marca visível
// aparecia só como o tracinho vermelho. Fills sólidos eliminam isso.
//
// JSX inline (não um <img src="/icon.svg">) pra não pagar uma requisição
// de rede por um asset tão pequeno e poder redimensionar via prop. É o
// mesmo desenho de app/icon.svg (favicon) — mantidos em sincronia à mão.
export function Logo({ size = 32, className = "" }: { size?: number; className?: string }) {
  return (
    <svg
      width={size}
      height={size}
      viewBox="0 0 28 28"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
      role="img"
      aria-label="Projeto Nova"
      className={`shrink-0 ${className}`}
    >
      <rect x="1" y="1" width="26" height="26" rx="8" fill="#16305a" />

      {/* Anéis do carimbo */}
      <circle cx="14" cy="14" r="10" fill="none" stroke="#F8FAFC" strokeOpacity="0.9" strokeWidth="1" />
      <circle cx="14" cy="14" r="7.6" fill="none" stroke="#F8FAFC" strokeOpacity="0.32" strokeWidth="0.75" />

      {/* "N" de Nova */}
      <rect x="9.6" y="9" width="2.3" height="10" fill="#F8FAFC" />
      <rect x="16.1" y="9" width="2.3" height="10" fill="#F8FAFC" />
      <polygon points="11.9,9 11.9,12.9 16.1,19 16.1,15.1" fill="#F8FAFC" />

      {/* Traço em tinta de carimbo */}
      <rect x="9.6" y="20.4" width="8.8" height="1.3" fill="#8a1c1c" />

      <rect
        x="1.5"
        y="1.5"
        width="25"
        height="25"
        rx="7.5"
        fill="none"
        stroke="rgba(255,255,255,0.18)"
        strokeWidth="0.75"
      />
    </svg>
  );
}
