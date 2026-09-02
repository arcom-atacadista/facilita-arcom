// Tipos do contrato com o backend (ver system-design/padroes/09-contrato-api.md).
// Um lugar só: se o backend mudar um campo, o TypeScript aponta todas as telas
// afetadas em vez de a tela quebrar em runtime.

export type Papel = "aprendiz" | "analista" | "coordenacao" | "gerencia";
export type Equipe = "interno" | "filiais";

export type Usuario = {
  id: string;
  nome: string;
  email: string;
  papel: Papel;
  equipe: Equipe;
  ativo: boolean;
  alcadaMaxima: number;
  codigoCobranca: string | null;
  criadoEm: string;
};

export type Faixa = "3-30" | "31-60" | "61-90" | "fora";

export type StatusDivida = "aberto" | "negociado" | "quitado" | "cancelado";

export type Cliente = {
  id: string;
  nome: string;
  documento: string;
  telefone: string | null;
  email: string | null;
};

export type Divida = {
  id: string;
  contrato: string;
  valorOriginal: number;
  /** ISO (AAAA-MM-DD) — o backend nunca manda data formatada. */
  vencimento: string;
  diasAtraso: number;
  faixa: Faixa;
  status: StatusDivida;
  filial: string | null;
  responsavelCobranca: string | null;
  cliente: Cliente;
  temLinkAtivo: boolean;
};

export type Oferta = {
  descontoPct: number;
  valorTotal: number;
  economia: number;
  maxParcelas: number;
  entrada: number;
};

export type Ofertas = {
  avista: Oferta;
  /** Nulo quando a política da faixa só permite à vista. */
  parcelado: Oferta | null;
};

export type Parcela = {
  numero: number;
  valor: number;
  vencimento: string;
  pago: boolean;
};

export type Acordo = {
  id: string;
  dividaId: string;
  tipoPagamento: "pix" | "boleto";
  descontoPct: number;
  entrada: number;
  parcelas: number;
  valorTotal: number;
  status: "ativo" | "quitado" | "rompido" | "cancelado";
  origem: "cliente" | "operador";
  criadoEm: string;
  lista: Parcela[];
};

export type Politica = {
  id: string;
  nome: string;
  faixaMin: number;
  faixaMax: number;
  descontoAvista: number;
  descontoParcelado: number;
  maxParcelas: number;
  entradaMinimaPct: number;
};

export type StatusDisparo = "na_fila" | "enviado" | "erro" | "cancelado";

export type Disparo = {
  id: string;
  dividaId: string;
  /** Já vem mascarado pelo backend — a tela nunca recebe o número completo. */
  telefone: string;
  canal: string;
  status: StatusDisparo;
  tentativas: number;
  erroDetalhe: string | null;
  agendadoPara: string;
  enviadoEm: string | null;
  criadoEm: string;
};

export type ResumoDisparos = {
  porStatus: Partial<Record<StatusDisparo, number>>;
  canal: string;
  /** Falso enquanto não houver canal de envio configurado. */
  ativo: boolean;
};

/** A proposta que o cliente devedor vê na tela pública. */
export type Proposta = {
  cliente: string;
  contrato: string;
  valorOriginal: number;
  vencimento: string;
  diasAtraso: number;
  ofertas: Ofertas;
  acordo: Acordo | null;
};

export type Paginacao = {
  pagina: number;
  porPagina: number;
  total: number;
  totalPaginas: number;
};

export type Lista<T> = {
  itens: T[];
  paginacao: Paginacao;
};
