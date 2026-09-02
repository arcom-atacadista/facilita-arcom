// Carteira em atraso — a tela principal da gestão. O que aparece aqui já vem
// recortado pelo backend: analista vê só a própria carteira, coordenação pra
// cima vê tudo.
import { keepPreviousData, useQuery } from "@tanstack/react-query";
import { useState } from "react";
import { Link, useSearchParams } from "react-router-dom";
import {
  Carregando,
  Etiqueta,
  Paginador,
  Selecao,
  Tabela,
  Td,
  TdNumero,
  Th,
  Vazio,
} from "../components/ui";
import { tomDaFaixa } from "../components/tons";
import { Cabecalho } from "../components/Shell";
import { api } from "../core/api";
import {
  ROTULO_FAIXA,
  ROTULO_STATUS_DIVIDA,
  data,
  documento,
  moeda,
} from "../core/formato";
import type { Divida, Lista } from "../tipos/api";

export default function Carteira() {
  // Os filtros vivem na URL: recarregar a página ou mandar o link pra alguém
  // preserva o que a pessoa estava vendo.
  const [params, setParams] = useSearchParams();
  const faixa = params.get("faixa") ?? "";
  const status = params.get("status") ?? "aberto";
  const pagina = Number(params.get("pagina") ?? 1);

  const [busca, setBusca] = useState(params.get("busca") ?? "");

  function atualizar(mudancas: Record<string, string>) {
    const novo = new URLSearchParams(params);
    for (const [chave, valor] of Object.entries(mudancas)) {
      if (valor) novo.set(chave, valor);
      else novo.delete(chave);
    }
    // Qualquer mudança de filtro volta pra primeira página: manter a página 4
    // de um filtro que agora tem 1 página mostraria uma lista vazia.
    if (!("pagina" in mudancas)) novo.delete("pagina");
    setParams(novo, { replace: true });
  }

  const consulta = useQuery({
    queryKey: [
      "carteira",
      { faixa, status, busca: params.get("busca") ?? "", pagina },
    ],
    queryFn: async () => {
      const { data: resposta } = await api.get<Lista<Divida>>("/carteira", {
        params: {
          faixa: faixa || undefined,
          status: status || undefined,
          busca: params.get("busca") || undefined,
          pagina,
        },
      });
      return resposta;
    },
    // Mantém a lista anterior visível enquanto a nova carrega — sem isso a
    // tabela pisca a cada troca de filtro.
    placeholderData: keepPreviousData,
  });

  return (
    <>
      <Cabecalho
        titulo="Carteira em atraso"
        descricao="Contratos de 3 a 90 dias de atraso."
      />

      <form
        className="mb-4 flex flex-wrap items-end gap-3"
        onSubmit={(e) => {
          e.preventDefault();
          atualizar({ busca });
        }}
      >
        <Selecao
          rotulo="Faixa"
          value={faixa}
          onChange={(e) => atualizar({ faixa: e.target.value })}
        >
          <option value="">Todas</option>
          <option value="3-30">3 a 30 dias</option>
          <option value="31-60">31 a 60 dias</option>
          <option value="61-90">61 a 90 dias</option>
        </Selecao>

        <Selecao
          rotulo="Situação"
          value={status}
          onChange={(e) => atualizar({ status: e.target.value })}
        >
          <option value="aberto">Em aberto</option>
          <option value="negociado">Negociado</option>
          <option value="quitado">Quitado</option>
          <option value="">Todas</option>
        </Selecao>

        <div className="min-w-56 flex-1">
          <label
            htmlFor="busca"
            className="mb-1.5 block text-xs font-bold uppercase tracking-wider text-arcom-gray"
          >
            Buscar
          </label>
          <input
            id="busca"
            value={busca}
            onChange={(e) => setBusca(e.target.value)}
            placeholder="Nome, CNPJ ou contrato"
            className="w-full rounded-md border border-surface-border bg-white px-3 py-2 text-sm text-verde-escuro focus:outline focus:outline-2 focus:outline-offset-1 focus:outline-verde-arcom"
          />
        </div>
      </form>

      {consulta.isLoading ? (
        <Carregando />
      ) : !consulta.data || consulta.data.itens.length === 0 ? (
        <Vazio
          titulo="Nenhum contrato encontrado"
          descricao="Ajuste os filtros ou confira se a sua carteira já foi importada do Gateway ARCOM."
        />
      ) : (
        <div className="flex flex-col gap-4">
          <Tabela
            cabecalho={
              <>
                <Th>Cliente</Th>
                <Th>Contrato</Th>
                <Th>Vencimento</Th>
                <Th className="text-right">Atraso</Th>
                <Th className="text-right">Valor</Th>
                <Th>Situação</Th>
                <Th />
              </>
            }
          >
            {consulta.data.itens.map((d) => (
              <tr key={d.id} className="hover:bg-surface/60">
                <Td>
                  <span className="block font-bold">{d.cliente.nome}</span>
                  <span className="block text-xs text-arcom-gray">
                    {documento(d.cliente.documento)}
                  </span>
                </Td>
                <Td>{d.contrato}</Td>
                <Td className="whitespace-nowrap">{data(d.vencimento)}</Td>
                <TdNumero>
                  <Etiqueta tom={tomDaFaixa(d.faixa)}>
                    {d.diasAtraso} d · {ROTULO_FAIXA[d.faixa]}
                  </Etiqueta>
                </TdNumero>
                <TdNumero>{moeda(d.valorOriginal)}</TdNumero>
                <Td>{ROTULO_STATUS_DIVIDA[d.status]}</Td>
                <Td className="text-right">
                  <Link
                    to={`/carteira/${d.id}`}
                    className="text-sm font-bold text-verde-arcom underline-offset-2 hover:underline"
                  >
                    Abrir
                  </Link>
                </Td>
              </tr>
            ))}
          </Tabela>

          <Paginador
            pagina={consulta.data.paginacao.pagina}
            totalPaginas={consulta.data.paginacao.totalPaginas}
            total={consulta.data.paginacao.total}
            aoMudar={(p) => atualizar({ pagina: String(p) })}
          />
        </div>
      )}
    </>
  );
}
