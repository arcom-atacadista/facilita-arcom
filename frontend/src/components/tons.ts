// Mapa de estado -> tom da etiqueta. Fica fora de ui.tsx porque aquele
// arquivo só exporta componentes: misturar função e componente no mesmo
// módulo quebra o fast refresh do Vite.
import type { TomEtiqueta } from "./ui";

/** Cor da faixa de atraso — quanto mais velho, mais grave. */
export function tomDaFaixa(faixa: string): TomEtiqueta {
  switch (faixa) {
    case "3-30":
      return "destaque";
    case "31-60":
      return "alerta";
    case "61-90":
      return "perigo";
    default:
      return "neutro";
  }
}

export function tomDoStatusDisparo(status: string): TomEtiqueta {
  switch (status) {
    case "enviado":
      return "sucesso";
    case "erro":
      return "perigo";
    case "cancelado":
      return "neutro";
    default:
      return "alerta";
  }
}
