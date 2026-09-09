// Mesa de atendimento: o analista lê o que o cliente respondeu e responde de
// volta.
//
// O layout de três colunas — fila, conversa, contexto do cliente — é o do
// projeto desenhado no Lovable, trazido para o Design System ARCOM e ligado ao
// backend real. O que a API ainda não serve não aparece inventado aqui: some,
// ou é dito na tela.
//
// A JANELA DE 24 HORAS É O CENTRO DA TELA
//
// Quando o cliente escreve, abre uma janela em que a resposta é texto livre e
// não é tarifada. Fora dela, só um template aprovado alcança a pessoa — e isso
// é disparo da régua, não conversa. Por isso a contagem aparece no topo da
// conversa, na cor do tempo que resta, e a caixa de resposta se fecha sozinha
// quando a janela vence: prometer envio que a Meta vai recusar é pior que
// mostrar o campo desabilitado.
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { keepPreviousData } from "@tanstack/react-query";
import { Clock, Search, Send, User } from "lucide-react";
import { useEffect, useMemo, useRef, useState } from "react";
import { Link } from "react-router-dom";
import { Cabecalho } from "../components/Shell";
import { Aviso, Botao, Carregando, Etiqueta, Vazio } from "../components/ui";
import type { TomEtiqueta } from "../components/ui";
import { api } from "../core/api";
import { cn } from "../core/cn";
import { dataHora } from "../core/formato";
import { notificar } from "../core/toast";
import type { ErroApi } from "../core/erro";
import type { Lista } from "../tipos/api";

type Conversa = {
  id: string;
  telefone: string;
  nome: string;
  nomePerfil: string | null;
  clienteId: string | null;
  janelaAberta: boolean;
  janelaExpiraEm: string | null;
  ultimaMensagemEm: string | null;
  naoLidas: number;
  previa: string | null;
  previaDoCliente: boolean;
};

type Mensagem = {
  id: string;
  direcao: "entrada" | "saida";
  tipo: string;
  texto: string | null;
  status: string | null;
  ocorridaEm: string;
};

// Frases que o analista usa o dia inteiro. São só texto inserido no campo —
// nada é enviado por clicar aqui, para o clique errado não falar com o cliente.
const FRASES_PRONTAS = [
  "Bom dia! Sou da equipe de cobrança da ARCOM.",
  "Consigo confirmar esse acordo hoje.",
  "Pode me enviar o comprovante do pagamento?",
  "Qual prazo você consegue para o retorno?",
];

const chaves = {
  fila: (pagina: number) => ["conversas", pagina] as const,
  thread: (id: string) => ["conversas", id, "mensagens"] as const,
};

export default function Mesa() {
  const [busca, setBusca] = useState("");
  const [soAguardando, setSoAguardando] = useState(false);
  const [selecionada, setSelecionada] = useState<string | null>(null);
  const queryClient = useQueryClient();

  const fila = useQuery({
    queryKey: chaves.fila(1),
    queryFn: async () =>
      (await api.get<Lista<Conversa>>("/conversas", { params: { pagina: 1, porPagina: 50 } }))
        .data,
    placeholderData: keepPreviousData,
    // A mesa é tela de plantão: o analista fica com ela aberta esperando
    // cliente responder. Sem refetch periódico ele só veria a resposta ao
    // recarregar a página na mão.
    refetchInterval: 20_000,
  });

  // useMemo e não `?? []` direto: um literal novo a cada render trocaria a
  // dependência do useMemo abaixo e refiltraria a fila sem nada ter mudado.
  const conversas = useMemo(() => fila.data?.itens ?? [], [fila.data]);

  const visiveis = useMemo(() => {
    const termo = busca.trim().toLowerCase();
    return conversas.filter((c) => {
      if (soAguardando && !c.previaDoCliente) return false;
      if (!termo) return true;
      return (
        c.nome.toLowerCase().includes(termo) ||
        c.telefone.toLowerCase().includes(termo)
      );
    });
  }, [conversas, busca, soAguardando]);

  // Seleciona a primeira conversa quando a fila chega, para a tela não abrir
  // com a coluna do meio vazia.
  useEffect(() => {
    const primeira = visiveis[0];
    if (selecionada === null && primeira) {
      setSelecionada(primeira.id);
    }
  }, [visiveis, selecionada]);

  const atual = conversas.find((c) => c.id === selecionada) ?? null;
  const aguardando = conversas.filter((c) => c.previaDoCliente).length;

  return (
    <>
      <Cabecalho
        titulo="Mesa de atendimento"
        descricao="Quem respondeu a cobrança, e o que a equipe respondeu de volta."
      />

      {fila.isLoading ? (
        <Carregando texto="Carregando a fila…" />
      ) : conversas.length === 0 ? (
        <Vazio
          titulo="Nenhuma conversa ainda"
          descricao="Assim que um cliente responder a uma mensagem da régua, a conversa aparece aqui."
        />
      ) : (
        <div className="grid min-h-[38rem] overflow-hidden rounded-lg border border-surface-border bg-white shadow-sm lg:grid-cols-[19rem_1fr] xl:grid-cols-[19rem_1fr_20rem]">
          <ColunaFila
            conversas={visiveis}
            total={conversas.length}
            aguardando={aguardando}
            busca={busca}
            onBusca={setBusca}
            soAguardando={soAguardando}
            onSoAguardando={setSoAguardando}
            selecionada={selecionada}
            onSelecionar={(id) => {
              setSelecionada(id);
              // Abrir a conversa zera as não lidas no servidor; invalidar a
              // fila é o que faz o contador sumir da lista sem F5.
              void queryClient.invalidateQueries({ queryKey: ["conversas"] });
            }}
          />
          {atual ? (
            <>
              <ColunaConversa conversa={atual} />
              <ColunaCliente conversa={atual} />
            </>
          ) : (
            <div className="border-l border-surface-border p-8">
              <Vazio
                titulo="Nada encontrado"
                descricao="Revise o nome ou o telefone digitado na busca."
              />
            </div>
          )}
        </div>
      )}
    </>
  );
}

// ─── coluna 1: a fila ────────────────────────────────────────────────────

function ColunaFila({
  conversas,
  total,
  aguardando,
  busca,
  onBusca,
  soAguardando,
  onSoAguardando,
  selecionada,
  onSelecionar,
}: {
  conversas: Conversa[];
  total: number;
  aguardando: number;
  busca: string;
  onBusca: (v: string) => void;
  soAguardando: boolean;
  onSoAguardando: (v: boolean) => void;
  selecionada: string | null;
  onSelecionar: (id: string) => void;
}) {
  return (
    <div className="flex min-w-0 flex-col">
      <div className="border-b border-surface-border p-3">
        <label className="flex items-center gap-2 rounded-md border border-surface-border bg-surface px-3 py-2 focus-within:border-verde-arcom">
          <Search className="h-4 w-4 flex-none text-arcom-gray" aria-hidden />
          <input
            className="min-w-0 flex-1 bg-transparent text-sm outline-none placeholder:text-arcom-gray"
            placeholder="Buscar por nome ou telefone"
            aria-label="Buscar por nome ou telefone"
            value={busca}
            onChange={(e) => onBusca(e.target.value)}
          />
        </label>
      </div>

      <div className="px-3 pt-3">
        <div className="flex items-center justify-center gap-2 rounded-md bg-verde-escuro px-3 py-2 text-xs font-bold text-white">
          Renegociação ARCOM
          <span className="opacity-75">· {total}</span>
        </div>
      </div>

      <div className="mt-3 flex border-b border-surface-border" role="tablist">
        <BotaoAba
          ativa={!soAguardando}
          onClick={() => onSoAguardando(false)}
          rotulo={`Todas · ${total}`}
        />
        <BotaoAba
          ativa={soAguardando}
          onClick={() => onSoAguardando(true)}
          rotulo={`Aguardando · ${aguardando}`}
        />
      </div>

      <ul className="flex-1 overflow-y-auto">
        {conversas.map((c) => (
          <li key={c.id}>
            <button
              type="button"
              aria-current={c.id === selecionada ? "true" : undefined}
              onClick={() => onSelecionar(c.id)}
              className={cn(
                "flex w-full gap-3 border-b border-l-[3px] border-surface-border px-3 py-3 text-left transition-colors duration-fast",
                c.id === selecionada
                  ? "border-l-verde-arcom bg-verde-arcom/10"
                  : "border-l-transparent hover:bg-surface",
              )}
            >
              <Iniciais nome={c.nome} />
              <span className="min-w-0 flex-1">
                <span className="flex items-baseline gap-2">
                  <b className="min-w-0 flex-1 truncate text-sm font-bold text-verde-escuro">
                    {c.nome}
                  </b>
                  <span className="flex flex-none items-center gap-1.5 text-xs text-arcom-gray">
                    <PontoDaJanela conversa={c} />
                    {horaCurta(c.ultimaMensagemEm)}
                  </span>
                </span>
                <span className="mt-0.5 flex items-center gap-2">
                  <span className="min-w-0 flex-1 truncate text-xs text-arcom-gray">
                    {c.previa ?? "Mensagem sem texto"}
                  </span>
                  {c.naoLidas > 0 ? (
                    <span className="flex-none rounded-full bg-verde-arcom px-1.5 py-0.5 text-[10px] font-bold text-white">
                      {c.naoLidas}
                    </span>
                  ) : null}
                </span>
              </span>
            </button>
          </li>
        ))}
        {conversas.length === 0 ? (
          <li className="px-4 py-8 text-center text-sm text-arcom-gray">
            Nenhuma conversa neste filtro.
          </li>
        ) : null}
      </ul>
    </div>
  );
}

function BotaoAba({
  ativa,
  onClick,
  rotulo,
}: {
  ativa: boolean;
  onClick: () => void;
  rotulo: string;
}) {
  return (
    <button
      type="button"
      role="tab"
      aria-selected={ativa}
      onClick={onClick}
      className={cn(
        "flex-1 border-b-2 px-2 py-2.5 text-xs font-bold transition-colors duration-fast",
        ativa
          ? "border-verde-arcom text-verde-arcom"
          : "border-transparent text-arcom-gray hover:text-verde-arcom",
      )}
    >
      {rotulo}
    </button>
  );
}

// ─── coluna 2: a conversa ────────────────────────────────────────────────

function ColunaConversa({ conversa }: { conversa: Conversa }) {
  const [texto, setTexto] = useState("");
  const campoRef = useRef<HTMLTextAreaElement>(null);
  const fimRef = useRef<HTMLDivElement>(null);
  const queryClient = useQueryClient();

  const thread = useQuery({
    queryKey: chaves.thread(conversa.id),
    queryFn: async () =>
      (await api.get<{ itens: Mensagem[] }>(`/conversas/${conversa.id}/mensagens`)).data,
    refetchInterval: 20_000,
  });

  const mensagens = thread.data?.itens ?? [];

  // Rola para o fim quando a thread muda: numa conversa, a mensagem que
  // importa é a última.
  useEffect(() => {
    fimRef.current?.scrollIntoView({ block: "end" });
  }, [mensagens.length, conversa.id]);

  // Troca de conversa limpa o rascunho — mandar para o cliente errado o texto
  // digitado para outro é o erro mais caro desta tela.
  useEffect(() => {
    setTexto("");
  }, [conversa.id]);

  const responder = useMutation({
    mutationFn: async (corpo: string) =>
      (
        await api.post<{ mensagem: Mensagem }>(
          `/conversas/${conversa.id}/mensagens`,
          { texto: corpo },
          { skipErroGlobal: true },
        )
      ).data,
    onSuccess: () => {
      setTexto("");
      void queryClient.invalidateQueries({ queryKey: chaves.thread(conversa.id) });
      void queryClient.invalidateQueries({ queryKey: ["conversas"] });
    },
    onError: (erro: ErroApi) => {
      // Tratado na mão, e não pelo interceptor, porque a janela fechada tem
      // resposta própria: a fila é o caminho, não "tente de novo".
      if (erro.codigo === "janela_fechada") {
        notificar.erro(
          "A janela de 24 horas fechou. Para falar com este cliente agora, use a régua de cobrança.",
        );
        void queryClient.invalidateQueries({ queryKey: ["conversas"] });
        return;
      }
      notificar.erro(erro.mensagem);
    },
  });

  const podeEnviar = conversa.janelaAberta && texto.trim().length > 0;

  function enviar() {
    if (!podeEnviar || responder.isPending) return;
    responder.mutate(texto.trim());
  }

  function inserir(frase: string) {
    setTexto((atual) => (atual ? `${atual} ${frase}` : frase));
    campoRef.current?.focus();
  }

  return (
    <div className="flex min-w-0 flex-col border-surface-border lg:border-l">
      <div className="flex items-center gap-3 border-b border-surface-border px-4 py-3">
        <Iniciais nome={conversa.nome} />
        <div className="min-w-0 flex-1">
          <b className="block truncate text-sm font-bold text-verde-escuro">
            {conversa.nome}
          </b>
          <span className="block truncate text-xs text-arcom-gray">
            {conversa.telefone}
            {conversa.nomePerfil && conversa.nomePerfil !== conversa.nome
              ? ` · perfil "${conversa.nomePerfil}"`
              : ""}
          </span>
        </div>
        <EtiquetaDaJanela conversa={conversa} />
      </div>

      <div className="flex-1 space-y-3 overflow-y-auto bg-surface p-4">
        {thread.isLoading ? (
          <Carregando texto="Carregando a conversa…" />
        ) : mensagens.length === 0 ? (
          <p className="py-8 text-center text-sm text-arcom-gray">
            Nenhuma mensagem registrada nesta conversa.
          </p>
        ) : (
          mensagens.map((m) => <Bolha key={m.id} mensagem={m} />)
        )}
        <div ref={fimRef} />
      </div>

      <div className="border-t border-surface-border p-3">
        {conversa.janelaAberta ? (
          <>
            <div className="mb-2 flex flex-wrap gap-1.5">
              {FRASES_PRONTAS.map((frase) => (
                <button
                  key={frase}
                  type="button"
                  onClick={() => inserir(frase)}
                  className="rounded-full border border-surface-border px-2.5 py-1 text-xs text-arcom-gray transition-colors duration-fast hover:border-verde-arcom hover:text-verde-arcom"
                >
                  {frase}
                </button>
              ))}
            </div>
            <div className="flex items-end gap-2">
              <textarea
                ref={campoRef}
                rows={2}
                value={texto}
                maxLength={4096}
                onChange={(e) => setTexto(e.target.value)}
                onKeyDown={(e) => {
                  if (e.key === "Enter" && !e.shiftKey) {
                    e.preventDefault();
                    enviar();
                  }
                }}
                aria-label="Escreva a resposta"
                placeholder="Escreva a resposta — Enter envia, Shift+Enter quebra linha"
                className="min-w-0 flex-1 resize-none rounded-md border border-surface-border px-3 py-2 text-sm outline-none focus:border-verde-arcom"
              />
              <Botao
                onClick={enviar}
                disabled={!podeEnviar || responder.isPending}
                carregando={responder.isPending}
              >
                <Send className="h-4 w-4" aria-hidden />
                Enviar
              </Botao>
            </div>
          </>
        ) : (
          <Aviso>
            <strong>A janela de 24 horas desta conversa fechou.</strong> O
            WhatsApp só aceita texto livre enquanto ela está aberta. Para falar
            com este cliente agora, use a{" "}
            <Link to="/disparos" className="font-bold underline">
              régua de cobrança
            </Link>
            , que manda uma mensagem aprovada e reabre a conversa quando ele
            responder.
          </Aviso>
        )}
      </div>
    </div>
  );
}

function Bolha({ mensagem }: { mensagem: Mensagem }) {
  const doCliente = mensagem.direcao === "entrada";
  // Template é o que a régua mandou sozinha. Marcar como automático evita o
  // analista achar que um colega falou com o cliente.
  const daRegua = !doCliente && mensagem.tipo === "template";

  return (
    <div className={cn("flex", doCliente ? "justify-start" : "justify-end")}>
      <div
        className={cn(
          "max-w-[80%] rounded-lg border px-3 py-2 text-sm",
          doCliente
            ? "border-surface-border bg-white text-verde-escuro"
            : daRegua
              ? "border-verde-lima bg-verde-lima/25 text-verde-escuro"
              : "border-verde-arcom bg-verde-arcom text-white",
        )}
      >
        <span
          className={cn(
            "mb-0.5 block text-[10px] font-bold tracking-wider uppercase",
            doCliente || daRegua ? "text-arcom-gray" : "text-white/70",
          )}
        >
          {doCliente ? "Cliente" : daRegua ? "Automático · régua" : "Equipe"}
        </span>
        <p className="break-words whitespace-pre-wrap">
          {mensagem.texto ?? `Mensagem do tipo ${mensagem.tipo}, sem texto.`}
        </p>
        <span
          className={cn(
            "mt-1 block text-[10px]",
            doCliente || daRegua ? "text-arcom-gray" : "text-white/70",
          )}
        >
          {dataHora(mensagem.ocorridaEm)}
          {mensagem.status ? ` · ${mensagem.status}` : ""}
        </span>
      </div>
    </div>
  );
}

// ─── coluna 3: contexto do cliente ───────────────────────────────────────

function ColunaCliente({ conversa }: { conversa: Conversa }) {
  return (
    <div className="hidden min-w-0 flex-col border-l border-surface-border xl:flex">
      <div className="flex items-center gap-3 border-b border-surface-border px-4 py-3">
        <Iniciais nome={conversa.nome} />
        <div className="min-w-0">
          <b className="block truncate text-sm font-bold text-verde-escuro">
            {conversa.nome}
          </b>
          <span className="block truncate text-xs text-arcom-gray">
            {conversa.telefone}
          </span>
        </div>
      </div>

      <div className="space-y-4 p-4">
        {conversa.clienteId ? (
          <div className="rounded-md border border-verde-arcom/25 bg-verde-arcom/10 p-3">
            <p className="flex items-center gap-2 text-xs font-bold text-verde-arcom">
              <User className="h-4 w-4" aria-hidden />
              Reconhecido na carteira
            </p>
            <p className="mt-1 text-xs text-arcom-gray">
              O número casa com um cliente cadastrado, então a dívida e a
              política aplicável vêm da carteira.
            </p>
            <Link
              to="/carteira"
              className="mt-2 inline-block text-xs font-bold text-verde-arcom underline"
            >
              Abrir na carteira
            </Link>
          </div>
        ) : (
          <div className="rounded-md border border-amber-200 bg-amber-50 p-3">
            <p className="text-xs font-bold text-amber-700">
              Número não reconhecido
            </p>
            <p className="mt-1 text-xs text-amber-700">
              Esse telefone não casou com nenhum cliente da carteira. Confirme
              quem é antes de tratar de valores — pode ser contato novo do
              cliente, ou pessoa errada.
            </p>
          </div>
        )}

        <div>
          <span className="text-[10px] font-bold tracking-wider text-arcom-gray uppercase">
            Janela de atendimento
          </span>
          <p className="mt-1 text-sm text-verde-escuro">
            {conversa.janelaAberta
              ? `Aberta até ${dataHora(conversa.janelaExpiraEm)}. Responder não tem custo.`
              : "Fechada. Só a régua alcança este cliente agora."}
          </p>
        </div>

        <div>
          <span className="text-[10px] font-bold tracking-wider text-arcom-gray uppercase">
            Última mensagem
          </span>
          <p className="mt-1 text-sm text-verde-escuro">
            {dataHora(conversa.ultimaMensagemEm)}
          </p>
        </div>

        {/* O painel de ofertas do desenho original (condição à vista, parcelado
            e a travada por alçada) depende de um cálculo que o backend ainda
            não expõe. Fica dito, e não simulado: número de acordo inventado na
            tela do analista vira proposta errada ao cliente. */}
        <div className="rounded-md border border-dashed border-surface-border p-3">
          <span className="text-[10px] font-bold tracking-wider text-arcom-gray uppercase">
            Próxima etapa
          </span>
          <p className="mt-1 text-xs text-arcom-gray">
            As condições calculadas — à vista, parcelado e o que exige aprovação
            — entram aqui quando o cálculo de oferta virar endpoint. Até então,
            simule pela tela da dívida antes de propor ao cliente.
          </p>
        </div>
      </div>
    </div>
  );
}

// ─── peças compartilhadas ────────────────────────────────────────────────

function Iniciais({ nome }: { nome: string }) {
  const letras = nome
    .split(/\s+/)
    .filter((p) => p.length > 0)
    .slice(0, 2)
    .map((p) => p[0]?.toUpperCase() ?? "")
    .join("");

  return (
    <span className="flex h-9 w-9 flex-none items-center justify-center rounded-full bg-verde-escuro text-xs font-bold text-verde-lima">
      {letras || "?"}
    </span>
  );
}

/** Minutos restantes da janela, ou null quando ela já fechou. */
function minutosRestantes(conversa: Conversa): number | null {
  if (!conversa.janelaAberta || !conversa.janelaExpiraEm) return null;
  const restante = new Date(conversa.janelaExpiraEm).getTime() - Date.now();
  return restante > 0 ? Math.floor(restante / 60_000) : null;
}

/** Tom pelo tempo que resta: o analista precisa ver de longe o que expira. */
function tomDaJanela(minutos: number | null): TomEtiqueta {
  if (minutos === null) return "neutro";
  if (minutos < 60) return "perigo";
  if (minutos < 360) return "alerta";
  return "sucesso";
}

function EtiquetaDaJanela({ conversa }: { conversa: Conversa }) {
  const minutos = minutosRestantes(conversa);

  return (
    <Etiqueta tom={tomDaJanela(minutos)}>
      <Clock className="mr-1 h-3 w-3" aria-hidden />
      {minutos === null ? "Janela fechada" : `Janela: ${duracao(minutos)}`}
    </Etiqueta>
  );
}

/** O ponto de cor na fila. Sempre acompanhado da hora ao lado e da etiqueta na
 *  conversa aberta — cor sozinha não informa quem não distingue verde de
 *  âmbar. */
function PontoDaJanela({ conversa }: { conversa: Conversa }) {
  const minutos = minutosRestantes(conversa);
  const cor =
    minutos === null
      ? "bg-arcom-gray"
      : minutos < 60
        ? "bg-danger"
        : minutos < 360
          ? "bg-amber-500"
          : "bg-verde-arcom";

  return (
    <span
      className={cn("h-2 w-2 flex-none rounded-full", cor)}
      title={minutos === null ? "Janela fechada" : `Janela: ${duracao(minutos)}`}
    />
  );
}

function duracao(minutos: number): string {
  const horas = Math.floor(minutos / 60);
  if (horas >= 1) return `${horas}h${String(minutos % 60).padStart(2, "0")}`;
  return `${minutos} min`;
}

/** Hora do dia para hoje, data curta para o resto — como a fila do WhatsApp. */
function horaCurta(iso: string | null): string {
  if (!iso) return "—";
  const quando = new Date(iso);
  const hoje = new Date();
  const mesmoDia =
    quando.getDate() === hoje.getDate() &&
    quando.getMonth() === hoje.getMonth() &&
    quando.getFullYear() === hoje.getFullYear();

  return quando.toLocaleString("pt-BR", {
    ...(mesmoDia
      ? { hour: "2-digit", minute: "2-digit" }
      : { day: "2-digit", month: "2-digit" }),
  });
}
