// Detalhe do contrato: condições da faixa, fechamento de acordo pelo operador
// e disparo da mensagem. A alçada aparece aqui, mas quem a aplica é o
// backend — o limite na tela é orientação, não proteção.
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useState } from "react";
import { Link, useParams } from "react-router-dom";
import { Cabecalho } from "../components/Shell";
import {
  Aviso,
  Botao,
  Carregando,
  Cartao,
  Etiqueta,
  Selecao,
  Tabela,
  Td,
  TdNumero,
  Th,
  TituloDeSecao,
} from "../components/ui";
import { tomDaFaixa } from "../components/tons";
import { api } from "../core/api";
import type { ErroApi } from "../core/erro";
import {
  ROTULO_FAIXA,
  ROTULO_STATUS_DIVIDA,
  data,
  documento,
  moeda,
  porcentagem,
  telefone,
} from "../core/formato";
import { useSessao } from "../core/sessao";
import { notificar } from "../core/toast";
import type {
  Acordo,
  Divida as TipoDivida,
  Ofertas,
  Posicao,
} from "../tipos/api";

type RespostaDivida = { divida: TipoDivida; posicao: Posicao; ofertas: Ofertas };

export default function Divida() {
  const { id = "" } = useParams();
  const { usuario } = useSessao();
  const queryClient = useQueryClient();

  const consulta = useQuery({
    queryKey: ["divida", id],
    queryFn: async () =>
      (await api.get<RespostaDivida>(`/carteira/${id}`)).data,
  });

  const [desconto, setDesconto] = useState("0");
  const [parcelas, setParcelas] = useState("1");
  const [pagamento, setPagamento] = useState<"pix" | "boleto">("pix");

  const fechar = useMutation({
    mutationFn: async () => {
      const { data: acordo } = await api.post<Acordo>(
        "/acordos",
        {
          // O acordo é do CNPJ, não deste título: a API recebe o cliente.
          clienteId: consulta.data?.divida.cliente.id,
          tipoPagamento: pagamento,
          descontoPct: Number(desconto),
          parcelas: Number(parcelas),
        },
        { skipErroGlobal: true },
      );
      return acordo;
    },
    onSuccess: () => {
      notificar.sucesso("Acordo registrado.");
      void queryClient.invalidateQueries({ queryKey: ["divida", id] });
      void queryClient.invalidateQueries({ queryKey: ["carteira"] });
      void queryClient.invalidateQueries({ queryKey: ["acordos"] });
    },
  });

  const disparar = useMutation({
    mutationFn: async () => {
      const { data: resposta } = await api.post<{ aviso?: string }>(
        "/disparos",
        { dividaId: id },
        { skipErroGlobal: true },
      );
      return resposta;
    },
    onSuccess: (resposta) => {
      // O backend avisa quando não há canal de envio: sem isso o operador
      // acharia que o cliente foi notificado.
      if (resposta.aviso) notificar.aviso(resposta.aviso);
      else notificar.sucesso("Mensagem enviada para a fila.");
      void queryClient.invalidateQueries({ queryKey: ["divida", id] });
      void queryClient.invalidateQueries({ queryKey: ["disparos"] });
    },
    onError: (erro) => notificar.erro((erro as ErroApi).mensagem),
  });

  if (consulta.isLoading) return <Carregando />;
  if (!consulta.data)
    return <Aviso tom="perigo">Contrato não encontrado.</Aviso>;

  const { divida, ofertas } = consulta.data;
  const erroAcordo = fechar.error as ErroApi | null;
  const acordoFechado = fechar.data;
  const podeNegociar = divida.status === "aberto" && !acordoFechado;

  const tetoDeParcelas = ofertas.parcelado?.maxParcelas ?? 1;
  const alcada = usuario?.alcadaMaxima ?? 0;
  const acimaDaAlcada = Number(desconto) > alcada;

  return (
    <>
      <Cabecalho
        titulo={divida.cliente.nome}
        descricao={`${documento(divida.cliente.documento)} · contrato ${divida.contrato}`}
        acao={
          <Link
            to="/carteira"
            className="text-sm font-bold text-verde-arcom underline-offset-2 hover:underline"
          >
            Voltar para a carteira
          </Link>
        }
      />

      <div className="grid gap-5 lg:grid-cols-3">
        <Cartao className="lg:col-span-1">
          <TituloDeSecao>Contrato</TituloDeSecao>
          <dl className="flex flex-col gap-3 text-sm">
            <Linha
              rotulo="Valor original"
              valor={moeda(divida.valorOriginal)}
            />
            <Linha rotulo="Vencimento" valor={data(divida.vencimento)} />
            <Linha
              rotulo="Atraso"
              valor={
                <Etiqueta tom={tomDaFaixa(divida.faixa)}>
                  {divida.diasAtraso} dias · {ROTULO_FAIXA[divida.faixa]}
                </Etiqueta>
              }
            />
            <Linha
              rotulo="Situação"
              valor={ROTULO_STATUS_DIVIDA[divida.status]}
            />
            <Linha
              rotulo="Telefone"
              valor={telefone(divida.cliente.telefone)}
            />
            <Linha rotulo="Filial" valor={divida.filial ?? "—"} />
          </dl>

          <div className="mt-5 border-t border-surface-border pt-4">
            <Botao
              variante="secundario"
              className="w-full"
              carregando={disparar.isPending}
              disabled={divida.status !== "aberto"}
              onClick={() => disparar.mutate()}
            >
              Enviar cobrança por mensagem
            </Botao>
            <p className="mt-2 text-xs text-arcom-gray">
              Gera um novo link de negociação e coloca a mensagem da faixa na
              fila. O link anterior deixa de valer.
            </p>
          </div>
        </Cartao>

        <div className="flex flex-col gap-5 lg:col-span-2">
          <Cartao>
            <TituloDeSecao>Condições da faixa</TituloDeSecao>
            <div className="grid gap-4 sm:grid-cols-2">
              <CartaoOferta titulo="À vista" oferta={ofertas.avista} />
              {ofertas.parcelado ? (
                <CartaoOferta
                  titulo={`Parcelado em até ${ofertas.parcelado.maxParcelas}x`}
                  oferta={ofertas.parcelado}
                />
              ) : (
                <div className="rounded-md border border-dashed border-surface-border p-4 text-sm text-arcom-gray">
                  A política desta faixa não permite parcelamento.
                </div>
              )}
            </div>
          </Cartao>

          {acordoFechado ? (
            <Cartao>
              <TituloDeSecao>Acordo registrado</TituloDeSecao>
              <ParcelasDoAcordo acordo={acordoFechado} />
            </Cartao>
          ) : podeNegociar ? (
            <Cartao>
              <TituloDeSecao>Fechar acordo na mesa</TituloDeSecao>
              <form
                className="flex flex-col gap-4"
                onSubmit={(e) => {
                  e.preventDefault();
                  fechar.mutate();
                }}
              >
                <div className="grid gap-4 sm:grid-cols-3">
                  <div className="flex flex-col gap-1.5">
                    <label
                      htmlFor="desconto"
                      className="text-xs font-bold uppercase tracking-wider text-arcom-gray"
                    >
                      Desconto (%)
                    </label>
                    <input
                      id="desconto"
                      type="number"
                      min={0}
                      max={100}
                      step={1}
                      value={desconto}
                      onChange={(e) => setDesconto(e.target.value)}
                      className="rounded-md border border-surface-border bg-white px-3 py-2 text-sm text-verde-escuro focus:outline focus:outline-2 focus:outline-offset-1 focus:outline-verde-arcom"
                    />
                    <span className="text-xs text-arcom-gray">
                      {alcada >= 100
                        ? "Sua alçada não tem limite."
                        : `Sua alçada vai até ${alcada}%.`}
                    </span>
                  </div>

                  <Selecao
                    rotulo="Parcelas"
                    id="parcelas"
                    value={parcelas}
                    onChange={(e) => setParcelas(e.target.value)}
                  >
                    {Array.from(
                      { length: tetoDeParcelas },
                      (_, i) => i + 1,
                    ).map((n) => (
                      <option key={n} value={n}>
                        {n}x
                      </option>
                    ))}
                  </Selecao>

                  <Selecao
                    rotulo="Pagamento"
                    id="pagamento"
                    value={pagamento}
                    onChange={(e) =>
                      setPagamento(e.target.value as "pix" | "boleto")
                    }
                  >
                    <option value="pix">Pix</option>
                    <option value="boleto">Boleto</option>
                  </Selecao>
                </div>

                {acimaDaAlcada ? (
                  <Aviso>
                    {porcentagem(Number(desconto))} está acima da sua alçada de{" "}
                    {porcentagem(alcada)}. O servidor vai recusar — peça
                    aprovação à coordenação.
                  </Aviso>
                ) : null}

                {erroAcordo ? (
                  <Aviso tom="perigo">{erroAcordo.mensagem}</Aviso>
                ) : null}

                <div>
                  <Botao type="submit" carregando={fechar.isPending}>
                    Registrar acordo
                  </Botao>
                </div>
              </form>
            </Cartao>
          ) : (
            <Cartao>
              <p className="text-sm text-arcom-gray">
                Este contrato está como{" "}
                <strong>{ROTULO_STATUS_DIVIDA[divida.status]}</strong> e não
                aceita nova negociação.
              </p>
            </Cartao>
          )}
        </div>
      </div>
    </>
  );
}

function Linha({ rotulo, valor }: { rotulo: string; valor: React.ReactNode }) {
  return (
    <div className="flex items-center justify-between gap-4">
      <dt className="text-arcom-gray">{rotulo}</dt>
      <dd className="text-right font-bold text-verde-escuro">{valor}</dd>
    </div>
  );
}

function CartaoOferta({
  titulo,
  oferta,
}: {
  titulo: string;
  oferta: Ofertas["avista"];
}) {
  return (
    <div className="rounded-md border border-surface-border p-4">
      <p className="text-xs font-bold uppercase tracking-wider text-arcom-gray">
        {titulo}
      </p>
      <p className="mt-1 text-2xl font-black tabular-nums text-verde-escuro">
        {moeda(oferta.valorTotal)}
      </p>
      <p className="mt-1 text-sm text-verde-arcom">
        {porcentagem(oferta.descontoPct)} de desconto · economia de{" "}
        {moeda(oferta.economia)}
      </p>
      {oferta.entrada > 0 ? (
        <p className="mt-1 text-xs text-arcom-gray">
          Entrada de {moeda(oferta.entrada)}
        </p>
      ) : null}
    </div>
  );
}

function ParcelasDoAcordo({ acordo }: { acordo: Acordo }) {
  return (
    <div className="flex flex-col gap-4">
      <p className="text-sm text-arcom-gray">
        {moeda(acordo.valorTotal)} em {acordo.parcelas}x ·{" "}
        {porcentagem(acordo.descontoPct)} de desconto ·{" "}
        {acordo.tipoPagamento === "pix" ? "Pix" : "Boleto"}
      </p>
      <Tabela
        cabecalho={
          <>
            <Th>Parcela</Th>
            <Th>Vencimento</Th>
            <Th className="text-right">Valor</Th>
          </>
        }
      >
        {acordo.lista.map((p) => (
          <tr key={p.numero}>
            <Td>{p.numero}</Td>
            <Td>{data(p.vencimento)}</Td>
            <TdNumero>{moeda(p.valor)}</TdNumero>
          </tr>
        ))}
      </Tabela>
    </div>
  );
}
