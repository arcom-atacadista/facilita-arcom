// Formatação para exibição. O backend manda número e data ISO em UTC; a
// conversão para o jeito brasileiro acontece só aqui, na hora de mostrar
// (ver system-design/padroes/09-contrato-api.md).
import dayjs from "dayjs";
import utc from "dayjs/plugin/utc";
import timezone from "dayjs/plugin/timezone";

dayjs.extend(utc);
dayjs.extend(timezone);

const FUSO = "America/Sao_Paulo";

const formatadorMoeda = new Intl.NumberFormat("pt-BR", {
  style: "currency",
  currency: "BRL",
});

export function moeda(valor: number): string {
  return formatadorMoeda.format(valor ?? 0);
}

export function porcentagem(valor: number): string {
  // Sem casas decimais quando é inteiro: "10%" lê melhor que "10,0%".
  return Number.isInteger(valor)
    ? `${valor}%`
    : `${valor.toFixed(1).replace(".", ",")}%`;
}

/** Data sem hora (vencimento) — o backend manda AAAA-MM-DD. */
export function data(iso: string | null | undefined): string {
  if (!iso) return "—";
  return dayjs.utc(iso.slice(0, 10)).format("DD/MM/YYYY");
}

/** Data com hora — converte do UTC do backend para o fuso de Brasília. */
export function dataHora(iso: string | null | undefined): string {
  if (!iso) return "—";
  return dayjs.utc(iso).tz(FUSO).format("DD/MM/YYYY [às] HH:mm");
}

export function documento(doc: string | null | undefined): string {
  const so = (doc ?? "").replace(/\D/g, "");
  if (so.length === 11)
    return so.replace(/(\d{3})(\d{3})(\d{3})(\d{2})/, "$1.$2.$3-$4");
  if (so.length === 14)
    return so.replace(/(\d{2})(\d{3})(\d{3})(\d{4})(\d{2})/, "$1.$2.$3/$4-$5");
  return doc ?? "—";
}

export function telefone(tel: string | null | undefined): string {
  const so = (tel ?? "").replace(/\D/g, "");
  if (so.length === 11)
    return so.replace(/(\d{2})(\d{5})(\d{4})/, "($1) $2-$3");
  if (so.length === 10)
    return so.replace(/(\d{2})(\d{4})(\d{4})/, "($1) $2-$3");
  return tel ?? "—";
}

export const ROTULO_FAIXA: Record<string, string> = {
  "3-30": "3 a 30 dias",
  "31-60": "31 a 60 dias",
  "61-90": "61 a 90 dias",
  fora: "Fora da régua",
};

export const ROTULO_PAPEL: Record<string, string> = {
  aprendiz: "Aprendiz",
  analista: "Analista",
  coordenacao: "Coordenação",
  gerencia: "Gerência",
};

export const ROTULO_STATUS_DIVIDA: Record<string, string> = {
  aberto: "Em aberto",
  negociado: "Negociado",
  quitado: "Quitado",
  cancelado: "Cancelado",
};

export const ROTULO_STATUS_DISPARO: Record<string, string> = {
  na_fila: "Na fila",
  enviado: "Enviado",
  erro: "Falhou",
  cancelado: "Cancelado",
};
