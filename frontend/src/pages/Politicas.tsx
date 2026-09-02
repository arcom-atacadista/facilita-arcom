// Política comercial por faixa de atraso: quanto de desconto e em quantas
// parcelas o cliente pode fechar sozinho pelo link.
//
// As faixas em si não são editáveis aqui: mudar o intervalo pode colidir com
// a faixa vizinha (o banco recusa faixas sobrepostas), e isso é decisão de
// régua, não ajuste de tela.
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useState } from "react";
import { Cabecalho } from "../components/Shell";
import {
  Aviso,
  Botao,
  Carregando,
  Cartao,
  TituloDeSecao,
} from "../components/ui";
import { api } from "../core/api";
import type { ErroApi } from "../core/erro";
import { notificar } from "../core/toast";
import type { Politica } from "../tipos/api";

export default function Politicas() {
  const consulta = useQuery({
    queryKey: ["politicas"],
    queryFn: async () =>
      (await api.get<{ itens: Politica[] }>("/politicas")).data.itens,
  });

  if (consulta.isLoading) return <Carregando />;

  return (
    <>
      <Cabecalho
        titulo="Políticas de negociação"
        descricao="O que o cliente recebe de oferta em cada faixa de atraso, sem passar por um analista."
      />

      <div className="grid gap-5 lg:grid-cols-3">
        {(consulta.data ?? []).map((p) => (
          <CartaoPolitica key={p.id} politica={p} />
        ))}
      </div>
    </>
  );
}

function CartaoPolitica({ politica }: { politica: Politica }) {
  const queryClient = useQueryClient();

  const [descontoAvista, setDescontoAvista] = useState(
    String(politica.descontoAvista),
  );
  const [descontoParcelado, setDescontoParcelado] = useState(
    String(politica.descontoParcelado),
  );
  const [maxParcelas, setMaxParcelas] = useState(String(politica.maxParcelas));
  const [entradaMinimaPct, setEntradaMinimaPct] = useState(
    String(politica.entradaMinimaPct),
  );

  const salvar = useMutation({
    mutationFn: () =>
      api.patch(
        `/politicas/${politica.id}`,
        {
          descontoAvista: Number(descontoAvista),
          descontoParcelado: Number(descontoParcelado),
          maxParcelas: Number(maxParcelas),
          entradaMinimaPct: Number(entradaMinimaPct),
        },
        { skipErroGlobal: true },
      ),
    onSuccess: () => {
      notificar.sucesso(`Política "${politica.nome}" atualizada.`);
      void queryClient.invalidateQueries({ queryKey: ["politicas"] });
    },
  });

  const erro = salvar.error as ErroApi | null;

  return (
    <Cartao>
      <TituloDeSecao>
        {politica.nome}
        <span className="ml-2 text-xs font-medium text-arcom-gray">
          {politica.faixaMin} a {politica.faixaMax} dias
        </span>
      </TituloDeSecao>

      <form
        className="flex flex-col gap-3"
        onSubmit={(e) => {
          e.preventDefault();
          salvar.mutate();
        }}
      >
        <CampoNumero
          rotulo="Desconto à vista (%)"
          id={`avista-${politica.id}`}
          valor={descontoAvista}
          aoMudar={setDescontoAvista}
          erro={erro?.campos?.descontoAvista}
        />
        <CampoNumero
          rotulo="Desconto parcelado (%)"
          id={`parcelado-${politica.id}`}
          valor={descontoParcelado}
          aoMudar={setDescontoParcelado}
          erro={erro?.campos?.descontoParcelado}
        />
        <CampoNumero
          rotulo="Máximo de parcelas"
          id={`parcelas-${politica.id}`}
          valor={maxParcelas}
          aoMudar={setMaxParcelas}
          min={1}
          max={24}
          erro={erro?.campos?.maxParcelas}
        />
        <CampoNumero
          rotulo="Entrada mínima (%)"
          id={`entrada-${politica.id}`}
          valor={entradaMinimaPct}
          aoMudar={setEntradaMinimaPct}
          erro={erro?.campos?.entradaMinimaPct}
        />

        {erro && !erro.campos ? (
          <Aviso tom="perigo">{erro.mensagem}</Aviso>
        ) : null}

        <div className="pt-1">
          <Botao type="submit" carregando={salvar.isPending} className="w-full">
            Salvar
          </Botao>
        </div>
      </form>

      <p className="mt-3 text-xs text-arcom-gray">
        O parcelamento oferecido também respeita o valor mínimo de parcela,
        então uma dívida pequena pode aceitar menos vezes do que o máximo
        configurado aqui.
      </p>
    </Cartao>
  );
}

function CampoNumero({
  rotulo,
  id,
  valor,
  aoMudar,
  erro,
  min = 0,
  max = 100,
}: {
  rotulo: string;
  id: string;
  valor: string;
  aoMudar: (v: string) => void;
  erro?: string;
  min?: number;
  max?: number;
}) {
  return (
    <div className="flex items-center justify-between gap-3">
      <label htmlFor={id} className="text-sm text-arcom-gray">
        {rotulo}
      </label>
      <div className="flex flex-col items-end">
        <input
          id={id}
          type="number"
          min={min}
          max={max}
          value={valor}
          onChange={(e) => aoMudar(e.target.value)}
          aria-invalid={erro ? true : undefined}
          className={
            "w-24 rounded-md border bg-white px-2 py-1.5 text-right text-sm tabular-nums text-verde-escuro focus:outline focus:outline-2 focus:outline-offset-1 focus:outline-verde-arcom " +
            (erro ? "border-danger" : "border-surface-border")
          }
        />
        {erro ? <span className="mt-1 text-xs text-danger">{erro}</span> : null}
      </div>
    </div>
  );
}
