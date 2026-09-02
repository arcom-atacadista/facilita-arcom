// Normaliza a resposta de erro do backend (application/problem+json, ver
// padroes/09-contrato-api.md) pro tipo que o resto do app usa. Nenhuma tela
// deve importar AxiosError diretamente para tratar erro de API — só este
// tipo.
export type Problema = {
  status: number;
  titulo: string;
  /** Texto que pode ir pro usuário (toast, campo de formulário). */
  mensagem: string;
  /** Contrato de máquina — decida comportamento por isto, nunca por `mensagem`. */
  codigo: string;
  campos?: Record<string, string>;
};

/**
 * Um `Problema` que também é um `Error` de verdade (stack trace, funciona com
 * `instanceof`) — é isto que `core/api.ts` rejeita nas promises, nunca um
 * objeto solto.
 */
export class ErroApi extends Error implements Problema {
  status: number;
  titulo: string;
  mensagem: string;
  codigo: string;
  campos?: Record<string, string>;

  constructor(problema: Problema) {
    super(problema.mensagem);
    this.name = "ErroApi";
    this.status = problema.status;
    this.titulo = problema.titulo;
    this.mensagem = problema.mensagem;
    this.codigo = problema.codigo;
    this.campos = problema.campos;
  }
}

type CorpoProblemaJSON = {
  title?: unknown;
  status?: unknown;
  detail?: unknown;
  codigo?: unknown;
  campos?: unknown;
};

function ehCorpoDeProblema(dado: unknown): dado is CorpoProblemaJSON {
  if (!dado || typeof dado !== "object") return false;
  const candidato = dado as Record<string, unknown>;
  // Exige `status` numérico pra diferenciar payload real da API de erro do
  // próprio axios (ex.: "Network Error", timeout), que não tem esse formato.
  return typeof candidato.status === "number";
}

/**
 * Converte o que veio de um erro do axios (`error.response?.data`, ou
 * `undefined` quando nem chegou a ter resposta — erro de rede) num
 * `ErroApi` sempre preenchido. Nunca lança.
 */
export function normalizarErro(
  status: number | undefined,
  corpo: unknown,
): ErroApi {
  if (status === undefined) {
    return new ErroApi({
      status: 0,
      titulo: "Erro de conexão",
      mensagem:
        "Não foi possível falar com o servidor. Verifique sua internet.",
      codigo: "erro_de_rede",
    });
  }

  if (ehCorpoDeProblema(corpo)) {
    return new ErroApi({
      status: typeof corpo.status === "number" ? corpo.status : status,
      titulo: typeof corpo.title === "string" ? corpo.title : "Erro",
      mensagem:
        typeof corpo.detail === "string"
          ? corpo.detail
          : "Ocorreu um erro. Tente novamente.",
      codigo: typeof corpo.codigo === "string" ? corpo.codigo : "erro_interno",
      campos:
        corpo.campos && typeof corpo.campos === "object"
          ? (corpo.campos as Record<string, string>)
          : undefined,
    });
  }

  return new ErroApi({
    status,
    titulo: "Erro",
    mensagem: "Ocorreu um erro. Tente novamente.",
    codigo: "erro_interno",
  });
}
