// Cria o primeiro usuário do sistema, como gerência. O backend só aceita
// enquanto não existe ninguém cadastrado — depois disso responde 409.
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useState } from "react";
import { Link, useNavigate } from "react-router-dom";
import { Aviso, Botao, Campo, Cartao } from "../components/ui";
import { api } from "../core/api";
import type { ErroApi } from "../core/erro";
import { notificar } from "../core/toast";

export default function PrimeiroAcesso() {
  const navigate = useNavigate();
  const queryClient = useQueryClient();

  const [nome, setNome] = useState("");
  const [email, setEmail] = useState("");
  const [senha, setSenha] = useState("");

  const criar = useMutation({
    mutationFn: () =>
      api.post(
        "/sessao/primeiro-acesso",
        { nome, email, senha, papel: "gerencia", equipe: "interno" },
        { skipErroGlobal: true },
      ),
    onSuccess: () => {
      queryClient.clear();
      notificar.sucesso("Acesso criado. Faça login para continuar.");
      navigate("/entrar", { replace: true });
    },
  });

  const erro = criar.error as ErroApi | null;

  return (
    <div className="flex min-h-screen items-center justify-center bg-verde-escuro p-6">
      <div className="w-full max-w-sm">
        <p className="mb-6 text-center text-2xl font-black tracking-tight text-white">
          Facilita <span className="text-verde-lima">ARCOM</span>
        </p>

        <Cartao>
          <h1 className="mb-1 text-base font-bold text-verde-escuro">
            Primeiro acesso
          </h1>
          <p className="mb-4 text-sm text-arcom-gray">
            Cria a conta inicial de gerência. Só funciona enquanto nenhum
            usuário existe.
          </p>

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
              value={email}
              onChange={(e) => setEmail(e.target.value)}
            />
            <Campo
              rotulo="Senha"
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

            {erro && !erro.campos ? (
              <Aviso tom="perigo">{erro.mensagem}</Aviso>
            ) : null}

            <Botao type="submit" carregando={criar.isPending}>
              Criar acesso
            </Botao>
          </form>
        </Cartao>

        <p className="mt-4 text-center text-xs text-white/60">
          <Link to="/entrar" className="underline hover:text-verde-lima">
            Voltar para o login
          </Link>
        </p>
      </div>
    </div>
  );
}
