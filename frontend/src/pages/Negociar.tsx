// Tela pública que o cliente devedor abre pelo link recebido na mensagem.
// Sem login: quem autoriza é a posse do token na URL.
//
// Nada de menu, nada de dado da empresa, e o backend só manda primeiro nome,
// contrato e valor — um link repassado não vira consulta à ficha do cliente.
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useState } from "react";
import { useParams } from "react-router-dom";
import { Aviso, Botao, Carregando, Cartao, Selecao } from "../components/ui";
import { api } from "../core/api";
import type { ErroApi } from "../core/erro";
import { data, moeda, porcentagem } from "../core/formato";
import type { Acordo, Proposta } from "../tipos/api";

export default function Negociar() {
  const { token = "" } = useParams();
  const queryClient = useQueryClient();

  const [tipo, setTipo] = useState<"avista" | "parcelado">("avista");
  const [parcelas, setParcelas] = useState("2");

  const consulta = useQuery({
    queryKey: ["negociar", token],
    queryFn: async () =>
      (await api.get<Proposta>(`/negociar/${token}`, { skipErroGlobal: true }))
        .data,
    retry: false,
  });

  const aceitar = useMutation({
    mutationFn: async () => {
      const { data: acordo } = await api.post<Acordo>(
        `/negociar/${token}/aceitar`,
        { tipo, parcelas: tipo === "avista" ? 1 : Number(parcelas) },
        { skipErroGlobal: true },
      );
      return acordo;
    },
    onSuccess: () =>
      void queryClient.invalidateQueries({ queryKey: ["negociar", token] }),
  });

  if (consulta.isLoading)
    return (
      <Moldura>
        <Carregando texto="Buscando sua proposta…" />
      </Moldura>
    );

  if (consulta.error) {
    const erro = consulta.error as ErroApi;
    return (
      <Moldura>
        <Cartao>
          <h1 className="mb-2 text-lg font-bold text-verde-escuro">
            Não conseguimos abrir sua proposta
          </h1>
          <p className="text-sm text-arcom-gray">{erro.mensagem}</p>
        </Cartao>
      </Moldura>
    );
  }

  const proposta = consulta.data;
  if (!proposta) return null;

  // Acordo já fechado (agora ou numa visita anterior): a tela vira o
  // comprovante, não oferece negociar de novo.
  const acordo = aceitar.data ?? proposta.acordo;
  if (acordo) {
    return (
      <Moldura>
        <Cartao>
          <p className="text-xs font-bold uppercase tracking-wider text-verde-arcom">
            Acordo confirmado
          </p>
          <h1 className="mt-1 text-lg font-bold text-verde-escuro">
            Tudo certo, {proposta.cliente}!
          </h1>
          <p className="mt-2 text-sm text-arcom-gray">
            Contrato {proposta.contrato} · {moeda(acordo.valorTotal)} em{" "}
            {acordo.parcelas}x
            {acordo.descontoPct > 0
              ? ` · ${porcentagem(acordo.descontoPct)} de desconto`
              : ""}
          </p>

          <ul className="mt-5 flex flex-col divide-y divide-surface-border border-y border-surface-border">
            {acordo.lista.map((p) => (
              <li
                key={p.numero}
                className="flex items-center justify-between gap-4 py-3 text-sm"
              >
                <span className="text-arcom-gray">
                  Parcela {p.numero} · vence {data(p.vencimento)}
                </span>
                <span className="font-bold tabular-nums text-verde-escuro">
                  {moeda(p.valor)}
                </span>
              </li>
            ))}
          </ul>

          <p className="mt-5 text-sm text-arcom-gray">
            {acordo.tipoPagamento === "pix"
              ? "Você vai receber os dados do Pix pelo mesmo canal desta mensagem."
              : "Os boletos serão enviados pelo mesmo canal desta mensagem."}
          </p>
        </Cartao>
      </Moldura>
    );
  }

  const parcelado = proposta.ofertas.parcelado;
  const escolhida = tipo === "avista" ? proposta.ofertas.avista : parcelado;
  const erroAceite = aceitar.error as ErroApi | null;

  return (
    <Moldura>
      <Cartao>
        <p className="text-xs font-bold uppercase tracking-wider text-arcom-gray">
          Proposta de regularização
        </p>
        <h1 className="mt-1 text-lg font-bold text-verde-escuro">
          Olá, {proposta.cliente}
        </h1>
        <p className="mt-2 text-sm text-arcom-gray">
          O contrato <strong>{proposta.contrato}</strong> está com{" "}
          {proposta.diasAtraso} dias de atraso, no valor de{" "}
          {moeda(proposta.valorOriginal)}. Veja as condições que preparamos:
        </p>

        <form
          className="mt-5 flex flex-col gap-4"
          onSubmit={(e) => {
            e.preventDefault();
            aceitar.mutate();
          }}
        >
          <fieldset className="flex flex-col gap-3">
            <legend className="sr-only">Escolha a condição de pagamento</legend>

            <OpcaoPagamento
              selecionada={tipo === "avista"}
              aoSelecionar={() => setTipo("avista")}
              titulo="Pagar à vista"
              valor={proposta.ofertas.avista.valorTotal}
              detalhe={
                proposta.ofertas.avista.descontoPct > 0
                  ? `${porcentagem(proposta.ofertas.avista.descontoPct)} de desconto · você economiza ${moeda(
                      proposta.ofertas.avista.economia,
                    )}`
                  : "Sem desconto para esta faixa de atraso"
              }
            />

            {parcelado ? (
              <OpcaoPagamento
                selecionada={tipo === "parcelado"}
                aoSelecionar={() => setTipo("parcelado")}
                titulo={`Parcelar em até ${parcelado.maxParcelas}x`}
                valor={parcelado.valorTotal}
                detalhe={
                  parcelado.entrada > 0
                    ? `Entrada de ${moeda(parcelado.entrada)} · ${porcentagem(parcelado.descontoPct)} de desconto`
                    : `${porcentagem(parcelado.descontoPct)} de desconto`
                }
              />
            ) : null}
          </fieldset>

          {tipo === "parcelado" && parcelado ? (
            <Selecao
              rotulo="Em quantas vezes"
              value={parcelas}
              onChange={(e) => setParcelas(e.target.value)}
            >
              {Array.from(
                { length: parcelado.maxParcelas - 1 },
                (_, i) => i + 2,
              ).map((n) => (
                <option key={n} value={n}>
                  {n}x de {moeda(parcelado.valorTotal / n)}
                </option>
              ))}
            </Selecao>
          ) : null}

          {erroAceite ? (
            <Aviso tom="perigo">{erroAceite.mensagem}</Aviso>
          ) : null}

          <Botao
            type="submit"
            carregando={aceitar.isPending}
            disabled={!escolhida}
          >
            Confirmar{" "}
            {tipo === "avista"
              ? "pagamento à vista"
              : `parcelamento em ${parcelas}x`}
          </Botao>

          <p className="text-xs text-arcom-gray">
            Ao confirmar, você aceita as condições acima. Em caso de dúvida,
            responda a mensagem que recebeu.
          </p>
        </form>
      </Cartao>
    </Moldura>
  );
}

function Moldura({ children }: { children: React.ReactNode }) {
  return (
    <div className="flex min-h-screen flex-col items-center bg-surface px-4 py-8">
      <p className="mb-6 text-xl font-black tracking-tight text-verde-escuro">
        Facilita <span className="text-verde-arcom">ARCOM</span>
      </p>
      <div className="w-full max-w-md">{children}</div>
    </div>
  );
}

function OpcaoPagamento({
  selecionada,
  aoSelecionar,
  titulo,
  valor,
  detalhe,
}: {
  selecionada: boolean;
  aoSelecionar: () => void;
  titulo: string;
  valor: number;
  detalhe: string;
}) {
  return (
    // O <label> envolve o radio e o texto: a área inteira do cartão fica
    // clicável, o que importa numa tela aberta no celular. O texto fica
    // direto sob o label (sem span de agrupamento) para continuar sendo o
    // rótulo acessível do controle.
    <label
      className={
        "grid cursor-pointer grid-cols-[auto_1fr] items-start gap-x-3 rounded-md border p-4 transition-colors duration-fast " +
        (selecionada
          ? "border-verde-arcom bg-verde-arcom/5"
          : "border-surface-border bg-white hover:border-verde-arcom/40")
      }
    >
      <input
        type="radio"
        name="condicao"
        checked={selecionada}
        onChange={aoSelecionar}
        className="row-span-3 mt-1 accent-verde-arcom"
      />
      <span className="text-sm font-bold text-verde-escuro">{titulo}</span>
      <span className="mt-0.5 text-xl font-black tabular-nums text-verde-escuro">
        {moeda(valor)}
      </span>
      <span className="mt-0.5 text-xs text-arcom-gray">{detalhe}</span>
    </label>
  );
}
