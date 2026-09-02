// Acordos fechados — pelo cliente no link ou pelo operador na mesa. Respeita
// o mesmo recorte de carteira da listagem de dívidas.
import { keepPreviousData, useQuery } from "@tanstack/react-query";
import { useState } from "react";
import { Cabecalho } from "../components/Shell";
import {
  Carregando,
  Etiqueta,
  Paginador,
  Tabela,
  Td,
  TdNumero,
  Th,
  Vazio,
} from "../components/ui";
import { api } from "../core/api";
import { dataHora, moeda, porcentagem } from "../core/formato";
import type { Acordo, Lista } from "../tipos/api";

const TOM_STATUS = {
  ativo: "sucesso",
  quitado: "sucesso",
  rompido: "perigo",
  cancelado: "neutro",
} as const;

export default function Acordos() {
  const [pagina, setPagina] = useState(1);

  const consulta = useQuery({
    queryKey: ["acordos", pagina],
    queryFn: async () =>
      (await api.get<Lista<Acordo>>("/acordos", { params: { pagina } })).data,
    placeholderData: keepPreviousData,
  });

  if (consulta.isLoading) return <Carregando />;

  return (
    <>
      <Cabecalho
        titulo="Acordos"
        descricao="Negociações fechadas pelo cliente ou pela mesa."
      />

      {!consulta.data || consulta.data.itens.length === 0 ? (
        <Vazio
          titulo="Nenhum acordo ainda"
          descricao="Acordos aparecem aqui assim que um cliente aceita uma proposta."
        />
      ) : (
        <div className="flex flex-col gap-4">
          <Tabela
            cabecalho={
              <>
                <Th>Fechado em</Th>
                <Th>Origem</Th>
                <Th className="text-right">Desconto</Th>
                <Th className="text-right">Parcelas</Th>
                <Th className="text-right">Total</Th>
                <Th>Situação</Th>
              </>
            }
          >
            {consulta.data.itens.map((a) => (
              <tr key={a.id} className="hover:bg-surface/60">
                <Td className="whitespace-nowrap">{dataHora(a.criadoEm)}</Td>
                <Td>{a.origem === "cliente" ? "Cliente (link)" : "Mesa"}</Td>
                <TdNumero>{porcentagem(a.descontoPct)}</TdNumero>
                <TdNumero>{a.parcelas}x</TdNumero>
                <TdNumero>{moeda(a.valorTotal)}</TdNumero>
                <Td>
                  <Etiqueta tom={TOM_STATUS[a.status]}>{a.status}</Etiqueta>
                </Td>
              </tr>
            ))}
          </Tabela>

          <Paginador
            pagina={consulta.data.paginacao.pagina}
            totalPaginas={consulta.data.paginacao.totalPaginas}
            total={consulta.data.paginacao.total}
            aoMudar={setPagina}
          />
        </div>
      )}
    </>
  );
}
