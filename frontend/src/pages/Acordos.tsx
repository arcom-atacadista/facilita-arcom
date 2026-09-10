// Acordos fechados — pelo cliente no link ou pelo operador na mesa. Respeita
// o mesmo recorte de carteira da listagem de dívidas.
import {
  keepPreviousData,
  useMutation,
  useQuery,
  useQueryClient,
} from "@tanstack/react-query";
import { useState } from "react";
import { Cabecalho } from "../components/Shell";
import {
  Aviso,
  Botao,
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
import type { ErroApi } from "../core/erro";
import { dataHora, moeda, porcentagem } from "../core/formato";
import { useSessao } from "../core/sessao";
import { notificar } from "../core/toast";
import type { Acordo, Lista } from "../tipos/api";

const TOM_STATUS = {
  ativo: "sucesso",
  quitado: "sucesso",
  rompido: "perigo",
  cancelado: "neutro",
} as const;

/** Encerrar acordo é de coordenação pra cima. Esconder o botão é conveniência;
 *  quem barra de verdade é a rota do backend. */
const PODE_ENCERRAR = ["coordenacao", "gerencia"];

export default function Acordos() {
  const [pagina, setPagina] = useState(1);
  const [encerrando, setEncerrando] = useState<Acordo | null>(null);
  const { usuario } = useSessao();
  const podeEncerrar = !!usuario && PODE_ENCERRAR.includes(usuario.papel);

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
                <Th></Th>
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
                  {a.motivo ? (
                    <span className="mt-0.5 block text-xs text-arcom-gray">{a.motivo}</span>
                  ) : null}
                </Td>
                <Td className="text-right whitespace-nowrap">
                  {podeEncerrar && a.status === "ativo" ? (
                    <Botao
                      variante="secundario"
                      className="px-3 py-1 text-xs"
                      onClick={() => setEncerrando(a)}
                    >
                      Encerrar
                    </Botao>
                  ) : null}
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

      {encerrando ? (
        <PainelEncerrar acordo={encerrando} aoFechar={() => setEncerrando(null)} />
      ) : null}
    </>
  );
}

type Destino = "rompido" | "quitado" | "cancelado";

const DESTINOS: { valor: Destino; rotulo: string; explicacao: string }[] = [
  {
    valor: "rompido",
    rotulo: "Rompido — o cliente não pagou",
    explicacao:
      "Os títulos voltam para aberto e o cliente volta a ser cobrado pela régua.",
  },
  {
    valor: "quitado",
    rotulo: "Quitado — o cliente pagou tudo",
    explicacao: "Os títulos são dados por pagos e as parcelas viram quitadas.",
  },
  {
    valor: "cancelado",
    rotulo: "Cancelado — erro ou renegociação",
    explicacao:
      "Mesmo efeito do rompimento nos títulos; muda só o motivo registrado.",
  },
];

function PainelEncerrar({
  acordo,
  aoFechar,
}: {
  acordo: Acordo;
  aoFechar: () => void;
}) {
  const [destino, setDestino] = useState<Destino>("rompido");
  const [motivo, setMotivo] = useState("");
  const queryClient = useQueryClient();

  const encerrar = useMutation({
    mutationFn: async () =>
      (
        await api.patch<Acordo>(
          `/acordos/${acordo.id}`,
          { status: destino, motivo: motivo.trim() },
          { skipErroGlobal: true },
        )
      ).data,
    onSuccess: (a) => {
      notificar.sucesso(`Acordo ${a.status}.`);
      void queryClient.invalidateQueries({ queryKey: ["acordos"] });
      void queryClient.invalidateQueries({ queryKey: ["carteira"] });
      void queryClient.invalidateQueries({ queryKey: ["ofertas"] });
      aoFechar();
    },
    onError: (erro: ErroApi) => notificar.erro(erro.mensagem),
  });

  const escolhido = DESTINOS.find((d) => d.valor === destino);

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-verde-escuro/40 p-4">
      <div className="w-full max-w-lg rounded-lg border border-surface-border bg-white p-6 shadow-lg">
        <h2 className="text-lg font-bold text-verde-escuro">Encerrar acordo</h2>
        <p className="mt-1 text-sm text-arcom-gray">
          {moeda(acordo.valorTotal)} em {acordo.parcelas}x, cobrindo {acordo.titulos}{" "}
          {acordo.titulos === 1 ? "título" : "títulos"}.
        </p>

        <fieldset className="mt-4 flex flex-col gap-2">
          <legend className="sr-only">O que aconteceu com este acordo</legend>
          {DESTINOS.map((d) => (
            <label
              key={d.valor}
              className="flex cursor-pointer items-start gap-2 rounded-md border border-surface-border p-3 text-sm hover:bg-surface"
            >
              <input
                type="radio"
                name="destino"
                value={d.valor}
                aria-label={d.rotulo}
                checked={destino === d.valor}
                onChange={() => setDestino(d.valor)}
                className="mt-0.5"
              />
              <span>
                <span className="font-bold text-verde-escuro">{d.rotulo}</span>
                <span className="mt-0.5 block text-xs text-arcom-gray">
                  {d.explicacao}
                </span>
              </span>
            </label>
          ))}
        </fieldset>

        <label className="mt-4 block">
          <span className="text-xs font-bold text-arcom-gray">
            Motivo (opcional)
          </span>
          <input
            value={motivo}
            maxLength={500}
            onChange={(e) => setMotivo(e.target.value)}
            placeholder="Fica registrado no acordo"
            className="mt-1 w-full rounded-md border border-surface-border px-3 py-2 text-sm outline-none focus:border-verde-arcom"
          />
        </label>

        {/* Encerrar não tem volta: acordo encerrado não retorna a ativo, e
            renegociar exige fechar um acordo novo com a posição de hoje. */}
        <div className="mt-4">
          <Aviso>
            <strong>Não tem volta.</strong> {escolhido?.explicacao} Para
            renegociar depois, será preciso fechar um acordo novo com a posição
            recalculada.
          </Aviso>
        </div>

        <div className="mt-5 flex justify-end gap-2">
          <Botao variante="secundario" onClick={aoFechar} disabled={encerrar.isPending}>
            Voltar
          </Botao>
          <Botao onClick={() => encerrar.mutate()} carregando={encerrar.isPending}>
            Encerrar acordo
          </Botao>
        </div>
      </div>
    </div>
  );
}
