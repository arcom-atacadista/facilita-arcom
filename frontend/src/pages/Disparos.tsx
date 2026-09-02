// Régua de cobrança: o que foi para a fila e o que saiu.
//
// Enquanto não existe canal de envio configurado, esta tela é o lugar onde
// isso fica explícito — o pior resultado possível seria a operação achar que
// falou com o cliente.
import { keepPreviousData, useQuery } from "@tanstack/react-query";
import { useState } from "react";
import { Cabecalho } from "../components/Shell";
import {
  Aviso,
  Carregando,
  Cartao,
  Etiqueta,
  Paginador,
  Tabela,
  Td,
  Th,
  Vazio,
} from "../components/ui";
import { tomDoStatusDisparo } from "../components/tons";
import { api } from "../core/api";
import { ROTULO_STATUS_DISPARO, dataHora } from "../core/formato";
import type { Disparo, Lista, ResumoDisparos } from "../tipos/api";

export default function Disparos() {
  const [pagina, setPagina] = useState(1);

  const resumo = useQuery({
    queryKey: ["disparos", "resumo"],
    queryFn: async () =>
      (await api.get<ResumoDisparos>("/disparos/resumo")).data,
  });

  const consulta = useQuery({
    queryKey: ["disparos", pagina],
    queryFn: async () =>
      (await api.get<Lista<Disparo>>("/disparos", { params: { pagina } })).data,
    placeholderData: keepPreviousData,
  });

  const porStatus = resumo.data?.porStatus ?? {};

  return (
    <>
      <Cabecalho
        titulo="Régua de cobrança"
        descricao="Mensagens montadas por faixa de atraso."
      />

      {resumo.data && !resumo.data.ativo ? (
        <div className="mb-5">
          <Aviso>
            <strong>Nenhum canal de envio está configurado.</strong> As
            mensagens são montadas e ficam na fila, mas não saem — e nenhuma é
            marcada como enviada. Uma mensagem que espera mais de 7 dias é
            descartada, para não cobrar por dívida que já pode ter sido paga.
          </Aviso>
        </div>
      ) : null}

      <div className="mb-5 grid gap-4 sm:grid-cols-4">
        <Indicador rotulo="Na fila" valor={porStatus.na_fila ?? 0} />
        <Indicador rotulo="Enviadas" valor={porStatus.enviado ?? 0} />
        <Indicador rotulo="Falharam" valor={porStatus.erro ?? 0} />
        <Indicador rotulo="Canceladas" valor={porStatus.cancelado ?? 0} />
      </div>

      {consulta.isLoading ? (
        <Carregando />
      ) : !consulta.data || consulta.data.itens.length === 0 ? (
        <Vazio
          titulo="Nenhum disparo registrado"
          descricao="Abra um contrato na carteira e use “Enviar cobrança por mensagem”."
        />
      ) : (
        <div className="flex flex-col gap-4">
          <Tabela
            cabecalho={
              <>
                <Th>Criado em</Th>
                <Th>Telefone</Th>
                <Th>Canal</Th>
                <Th>Situação</Th>
                <Th>Enviado em</Th>
                <Th>Observação</Th>
              </>
            }
          >
            {consulta.data.itens.map((d) => (
              <tr key={d.id} className="hover:bg-surface/60">
                <Td className="whitespace-nowrap">{dataHora(d.criadoEm)}</Td>
                <Td className="tabular-nums">{d.telefone}</Td>
                <Td>{d.canal}</Td>
                <Td>
                  <Etiqueta tom={tomDoStatusDisparo(d.status)}>
                    {ROTULO_STATUS_DISPARO[d.status]}
                  </Etiqueta>
                </Td>
                <Td className="whitespace-nowrap">{dataHora(d.enviadoEm)}</Td>
                <Td className="max-w-72 text-xs text-arcom-gray">
                  {d.erroDetalhe ?? "—"}
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

function Indicador({ rotulo, valor }: { rotulo: string; valor: number }) {
  return (
    <Cartao>
      <p className="text-xs font-bold uppercase tracking-wider text-arcom-gray">
        {rotulo}
      </p>
      <p className="mt-1 text-2xl font-black tabular-nums text-verde-escuro">
        {valor}
      </p>
    </Cartao>
  );
}
