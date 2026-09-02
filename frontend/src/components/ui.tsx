// Primitivos visuais do Facilita ARCOM. Todos usam só os tokens do
// tailwind.config.js — nenhuma cor, fonte ou raio solto
// (system-design/padroes/08-design-system.md).
import type {
  ButtonHTMLAttributes,
  InputHTMLAttributes,
  ReactNode,
  SelectHTMLAttributes,
} from "react";
import { cn } from "../core/cn";

// ─── Botão ───────────────────────────────────────────────────────────────

type VarianteBotao = "primario" | "secundario" | "fantasma" | "perigo";

const estilosBotao: Record<VarianteBotao, string> = {
  primario: "bg-verde-arcom text-white hover:bg-verde-escuro",
  secundario:
    "bg-white text-verde-escuro border border-surface-border hover:border-verde-arcom hover:text-verde-arcom",
  fantasma:
    "bg-transparent text-arcom-gray hover:bg-surface hover:text-verde-escuro",
  perigo: "bg-danger text-white hover:opacity-90",
};

type PropsBotao = ButtonHTMLAttributes<HTMLButtonElement> & {
  variante?: VarianteBotao;
  carregando?: boolean;
};

export function Botao({
  variante = "primario",
  carregando,
  className,
  children,
  disabled,
  ...resto
}: PropsBotao) {
  return (
    <button
      {...resto}
      // Botão de submit que dispara mutação fica desabilitado enquanto ela
      // corre: é o que impede o duplo clique virar dois acordos.
      disabled={disabled || carregando}
      className={cn(
        "inline-flex items-center justify-center gap-2 rounded-md px-4 py-2 text-sm font-bold",
        "transition-colors duration-fast",
        "focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-verde-arcom",
        "disabled:cursor-not-allowed disabled:opacity-50",
        estilosBotao[variante],
        className,
      )}
    >
      {carregando ? "Aguarde…" : children}
    </button>
  );
}

// ─── Campo de formulário ─────────────────────────────────────────────────

type PropsCampo = InputHTMLAttributes<HTMLInputElement> & {
  rotulo: string;
  erro?: string;
  dica?: string;
};

export function Campo({
  rotulo,
  erro,
  dica,
  id,
  className,
  ...resto
}: PropsCampo) {
  const idCampo = id ?? resto.name;
  const idErro = erro ? `${idCampo}-erro` : undefined;
  const idDica = dica ? `${idCampo}-dica` : undefined;

  return (
    <div className="flex flex-col gap-1.5">
      <label
        htmlFor={idCampo}
        className="text-xs font-bold uppercase tracking-wider text-arcom-gray"
      >
        {rotulo}
      </label>
      <input
        {...resto}
        id={idCampo}
        aria-invalid={erro ? true : undefined}
        // Liga a mensagem ao campo pra leitor de tela anunciar o erro junto.
        aria-describedby={cn(idErro, idDica) || undefined}
        className={cn(
          "rounded-md border bg-white px-3 py-2 text-sm text-verde-escuro",
          "focus:outline focus:outline-2 focus:outline-offset-1 focus:outline-verde-arcom",
          erro ? "border-danger" : "border-surface-border",
          className,
        )}
      />
      {dica && !erro ? (
        <span id={idDica} className="text-xs text-arcom-gray">
          {dica}
        </span>
      ) : null}
      {erro ? (
        <span id={idErro} className="text-xs font-medium text-danger">
          {erro}
        </span>
      ) : null}
    </div>
  );
}

type PropsSelecao = SelectHTMLAttributes<HTMLSelectElement> & {
  rotulo: string;
  erro?: string;
};

export function Selecao({
  rotulo,
  erro,
  id,
  className,
  children,
  ...resto
}: PropsSelecao) {
  const idCampo = id ?? resto.name;
  return (
    <div className="flex flex-col gap-1.5">
      <label
        htmlFor={idCampo}
        className="text-xs font-bold uppercase tracking-wider text-arcom-gray"
      >
        {rotulo}
      </label>
      <select
        {...resto}
        id={idCampo}
        aria-invalid={erro ? true : undefined}
        className={cn(
          "rounded-md border bg-white px-3 py-2 text-sm text-verde-escuro",
          "focus:outline focus:outline-2 focus:outline-offset-1 focus:outline-verde-arcom",
          erro ? "border-danger" : "border-surface-border",
          className,
        )}
      >
        {children}
      </select>
      {erro ? (
        <span className="text-xs font-medium text-danger">{erro}</span>
      ) : null}
    </div>
  );
}

// ─── Cartão ──────────────────────────────────────────────────────────────

export function Cartao({
  children,
  className,
}: {
  children: ReactNode;
  className?: string;
}) {
  return (
    <div
      className={cn(
        "rounded-lg border border-surface-border bg-white p-5 shadow-sm",
        className,
      )}
    >
      {children}
    </div>
  );
}

export function TituloDeSecao({
  children,
  acao,
}: {
  children: ReactNode;
  acao?: ReactNode;
}) {
  return (
    <div className="mb-4 flex items-center justify-between gap-4">
      <h2 className="text-base font-bold text-verde-escuro">{children}</h2>
      {acao}
    </div>
  );
}

// ─── Etiqueta de estado ──────────────────────────────────────────────────

export type TomEtiqueta =
  | "neutro"
  | "sucesso"
  | "alerta"
  | "perigo"
  | "destaque";

const estilosEtiqueta: Record<TomEtiqueta, string> = {
  neutro: "bg-surface text-arcom-gray border-surface-border",
  sucesso: "bg-verde-arcom/10 text-verde-arcom border-verde-arcom/25",
  alerta: "bg-amber-50 text-amber-700 border-amber-200",
  perigo: "bg-danger/10 text-danger border-danger/25",
  destaque: "bg-verde-lima/25 text-verde-escuro border-verde-lima",
};

export function Etiqueta({
  tom = "neutro",
  children,
}: {
  tom?: TomEtiqueta;
  children: ReactNode;
}) {
  return (
    <span
      className={cn(
        "inline-flex items-center rounded-full border px-2.5 py-0.5 text-xs font-bold whitespace-nowrap",
        estilosEtiqueta[tom],
      )}
    >
      {children}
    </span>
  );
}

// ─── Tabela ──────────────────────────────────────────────────────────────

export function Tabela({
  children,
  cabecalho,
}: {
  children: ReactNode;
  cabecalho: ReactNode;
}) {
  return (
    // overflow-x-auto: numa tela estreita a tabela rola dentro do próprio
    // container em vez de esticar a página inteira.
    <div className="overflow-x-auto rounded-lg border border-surface-border bg-white">
      <table className="w-full border-collapse text-sm">
        <thead className="border-b border-surface-border bg-surface/60">
          <tr className="text-left text-xs font-bold uppercase tracking-wider text-arcom-gray">
            {cabecalho}
          </tr>
        </thead>
        <tbody className="divide-y divide-surface-border">{children}</tbody>
      </table>
    </div>
  );
}

// children opcional: a coluna de ação tem cabeçalho vazio, e um <th> vazio é
// o correto ali — omitir a coluna desalinharia o cabeçalho das linhas.
export function Th({
  children,
  className,
}: {
  children?: ReactNode;
  className?: string;
}) {
  return (
    <th className={cn("whitespace-nowrap px-4 py-3 font-bold", className)}>
      {children}
    </th>
  );
}

export function Td({
  children,
  className,
}: {
  children: ReactNode;
  className?: string;
}) {
  return (
    <td className={cn("px-4 py-3 align-middle text-verde-escuro", className)}>
      {children}
    </td>
  );
}

/** Números em coluna alinham à direita e com largura fixa de dígito. */
export function TdNumero({ children }: { children: ReactNode }) {
  return (
    <Td className="text-right tabular-nums whitespace-nowrap">{children}</Td>
  );
}

// ─── Estados de lista ────────────────────────────────────────────────────

export function Vazio({
  titulo,
  descricao,
}: {
  titulo: string;
  descricao?: string;
}) {
  return (
    <div className="rounded-lg border border-dashed border-surface-border bg-white px-6 py-12 text-center">
      <p className="text-sm font-bold text-verde-escuro">{titulo}</p>
      {descricao ? (
        <p className="mt-1 text-sm text-arcom-gray">{descricao}</p>
      ) : null}
    </div>
  );
}

export function Carregando({ texto = "Carregando…" }: { texto?: string }) {
  return (
    <div
      className="px-6 py-12 text-center text-sm text-arcom-gray"
      role="status"
      aria-live="polite"
    >
      {texto}
    </div>
  );
}

/** Aviso persistente — usado onde uma condição do sistema precisa ficar à vista. */
export function Aviso({
  tom = "alerta",
  children,
}: {
  tom?: "alerta" | "perigo";
  children: ReactNode;
}) {
  return (
    <div
      className={cn(
        "rounded-md border px-4 py-3 text-sm",
        tom === "perigo"
          ? "border-danger/30 bg-danger/5 text-danger"
          : "border-amber-200 bg-amber-50 text-amber-800",
      )}
    >
      {children}
    </div>
  );
}

// ─── Paginação ───────────────────────────────────────────────────────────

export function Paginador({
  pagina,
  totalPaginas,
  total,
  aoMudar,
}: {
  pagina: number;
  totalPaginas: number;
  total: number;
  aoMudar: (pagina: number) => void;
}) {
  if (totalPaginas <= 1) {
    return <p className="text-xs text-arcom-gray">{total} registro(s)</p>;
  }
  return (
    <div className="flex items-center justify-between gap-4">
      <p className="text-xs text-arcom-gray">
        Página {pagina} de {totalPaginas} · {total} registro(s)
      </p>
      <div className="flex gap-2">
        <Botao
          variante="secundario"
          onClick={() => aoMudar(pagina - 1)}
          disabled={pagina <= 1}
        >
          Anterior
        </Botao>
        <Botao
          variante="secundario"
          onClick={() => aoMudar(pagina + 1)}
          disabled={pagina >= totalPaginas}
        >
          Próxima
        </Botao>
      </div>
    </div>
  );
}
