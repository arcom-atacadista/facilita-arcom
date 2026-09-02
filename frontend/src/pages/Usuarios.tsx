// Cadastro de operadores. Não há cadastro público: quem cria acesso é a
// coordenação ou a gerência.
//
// O código de cobrança é o que liga o operador à carteira dele. Deixá-lo
// vazio é seguro por padrão: quem não vê a carteira inteira e não tem código
// não vê dívida nenhuma, em vez de ver todas.
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useState } from "react";
import { Cabecalho } from "../components/Shell";
import {
  Aviso,
  Botao,
  Campo,
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
import { api } from "../core/api";
import type { ErroApi } from "../core/erro";
import { ROTULO_PAPEL, dataHora } from "../core/formato";
import { useSessao } from "../core/sessao";
import { notificar } from "../core/toast";
import type { Papel, Usuario } from "../tipos/api";

const NIVEL: Record<Papel, number> = {
  aprendiz: 1,
  analista: 2,
  coordenacao: 3,
  gerencia: 4,
};

export default function Usuarios() {
  const { usuario: eu } = useSessao();
  const queryClient = useQueryClient();

  const consulta = useQuery({
    queryKey: ["usuarios"],
    queryFn: async () =>
      (await api.get<{ itens: Usuario[] }>("/usuarios")).data.itens,
  });

  const [nome, setNome] = useState("");
  const [email, setEmail] = useState("");
  const [senha, setSenha] = useState("");
  const [papel, setPapel] = useState<Papel>("analista");
  const [codigoCobranca, setCodigoCobranca] = useState("");

  const criar = useMutation({
    mutationFn: () =>
      api.post(
        "/usuarios",
        {
          nome,
          email,
          senha,
          papel,
          equipe: "interno",
          codigoCobranca: codigoCobranca || undefined,
        },
        { skipErroGlobal: true },
      ),
    onSuccess: () => {
      notificar.sucesso("Usuário criado.");
      setNome("");
      setEmail("");
      setSenha("");
      setCodigoCobranca("");
      void queryClient.invalidateQueries({ queryKey: ["usuarios"] });
    },
  });

  const alternarAtivo = useMutation({
    mutationFn: ({ id, ativo }: { id: string; ativo: boolean }) =>
      api.patch(`/usuarios/${id}`, { ativo }, { skipErroGlobal: true }),
    onSuccess: (_, { ativo }) => {
      notificar.sucesso(
        ativo
          ? "Usuário reativado."
          : "Usuário desativado e sessões encerradas.",
      );
      void queryClient.invalidateQueries({ queryKey: ["usuarios"] });
    },
    onError: (erro) => notificar.erro((erro as ErroApi).mensagem),
  });

  const erro = criar.error as ErroApi | null;
  // Ninguém cria alguém acima de si — a lista de papéis já reflete isso.
  const papeisDisponiveis = (Object.keys(NIVEL) as Papel[]).filter(
    (p) => !eu || NIVEL[p] <= NIVEL[eu.papel],
  );

  return (
    <>
      <Cabecalho
        titulo="Usuários"
        descricao="Operadores com acesso ao Facilita ARCOM."
      />

      <div className="grid gap-5 lg:grid-cols-3">
        <Cartao className="lg:col-span-1">
          <TituloDeSecao>Novo usuário</TituloDeSecao>
          <form
            className="flex flex-col gap-4"
            onSubmit={(e) => {
              e.preventDefault();
              criar.mutate();
            }}
          >
            <Campo
              rotulo="Nome"
              name="nome"
              required
              value={nome}
              onChange={(e) => setNome(e.target.value)}
            />
            <Campo
              rotulo="E-mail corporativo"
              name="email"
              type="email"
              required
              erro={erro?.campos?.email}
              value={email}
              onChange={(e) => setEmail(e.target.value)}
            />
            <Campo
              rotulo="Senha provisória"
              name="senha"
              type="password"
              autoComplete="new-password"
              required
              minLength={10}
              dica="No mínimo 10 caracteres."
              erro={erro?.campos?.senha}
              value={senha}
              onChange={(e) => setSenha(e.target.value)}
            />
            <Selecao
              rotulo="Papel"
              value={papel}
              onChange={(e) => setPapel(e.target.value as Papel)}
            >
              {papeisDisponiveis.map((p) => (
                <option key={p} value={p}>
                  {ROTULO_PAPEL[p]}
                </option>
              ))}
            </Selecao>
            <Campo
              rotulo="Código de cobrança"
              name="codigoCobranca"
              dica="Liga o operador à carteira dele. Em branco, ele não vê nenhuma dívida."
              value={codigoCobranca}
              onChange={(e) => setCodigoCobranca(e.target.value)}
            />

            {erro && !erro.campos ? (
              <Aviso tom="perigo">{erro.mensagem}</Aviso>
            ) : null}

            <Botao type="submit" carregando={criar.isPending}>
              Criar usuário
            </Botao>
          </form>
        </Cartao>

        <div className="lg:col-span-2">
          {consulta.isLoading ? (
            <Carregando />
          ) : (
            <Tabela
              cabecalho={
                <>
                  <Th>Nome</Th>
                  <Th>Papel</Th>
                  <Th>Carteira</Th>
                  <Th className="text-right">Alçada</Th>
                  <Th>Criado em</Th>
                  <Th />
                </>
              }
            >
              {(consulta.data ?? []).map((u) => (
                <tr key={u.id} className="hover:bg-surface/60">
                  <Td>
                    <span className="block font-bold">{u.nome}</span>
                    <span className="block text-xs text-arcom-gray">
                      {u.email}
                    </span>
                  </Td>
                  <Td>{ROTULO_PAPEL[u.papel]}</Td>
                  <Td>
                    {u.codigoCobranca ?? (
                      <span className="text-arcom-gray">nenhuma</span>
                    )}
                  </Td>
                  <TdNumero>
                    {u.alcadaMaxima >= 100
                      ? "sem limite"
                      : `${u.alcadaMaxima}%`}
                  </TdNumero>
                  <Td className="whitespace-nowrap">{dataHora(u.criadoEm)}</Td>
                  <Td className="text-right">
                    {u.id === eu?.id ? (
                      <Etiqueta tom="destaque">você</Etiqueta>
                    ) : (
                      <button
                        type="button"
                        onClick={() =>
                          alternarAtivo.mutate({ id: u.id, ativo: !u.ativo })
                        }
                        disabled={alternarAtivo.isPending}
                        className="text-sm font-bold text-verde-arcom underline-offset-2 hover:underline disabled:opacity-50"
                      >
                        {u.ativo ? "Desativar" : "Reativar"}
                      </button>
                    )}
                  </Td>
                </tr>
              ))}
            </Tabela>
          )}
        </div>
      </div>
    </>
  );
}
