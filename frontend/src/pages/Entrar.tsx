// Login. Não existe cadastro público: quem cria acesso é a coordenação. A
// única exceção é o primeiro usuário do sistema, em /primeiro-acesso.
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useState } from "react";
import { Link, Navigate, useNavigate } from "react-router-dom";
import { Botao, Campo, Cartao } from "../components/ui";
import { api } from "../core/api";
import type { ErroApi } from "../core/erro";
import { useSessao } from "../core/sessao";
import type { Usuario } from "../tipos/api";

export default function Entrar() {
  const { usuario, carregando } = useSessao();
  const navigate = useNavigate();
  const queryClient = useQueryClient();

  const [email, setEmail] = useState("");
  const [senha, setSenha] = useState("");

  const entrar = useMutation({
    mutationFn: async () => {
      // skipErroGlobal: o 401 aqui é "senha errada", não "sessão expirou" —
      // sem isso o interceptor global trataria como sessão perdida.
      const { data } = await api.post<Usuario>(
        "/sessao",
        { email, senha },
        { skipErroGlobal: true },
      );
      return data;
    },
    onSuccess: (dados) => {
      queryClient.setQueryData(["sessao"], dados);
      navigate("/carteira", { replace: true });
    },
  });

  if (carregando) return null;
  if (usuario) return <Navigate to="/carteira" replace />;

  const erro = entrar.error as ErroApi | null;

  return (
    <div className="flex min-h-screen items-center justify-center bg-verde-escuro p-6">
      <div className="w-full max-w-sm">
        <div className="mb-6 text-center">
          <p className="text-2xl font-black tracking-tight text-white">
            Facilita <span className="text-verde-lima">ARCOM</span>
          </p>
          <p className="mt-1 text-sm text-white/60">Crédito e Cobrança</p>
        </div>

        <Cartao>
          <form
            className="flex flex-col gap-4"
            onSubmit={(e) => {
              e.preventDefault();
              entrar.mutate();
            }}
          >
            <Campo
              rotulo="E-mail corporativo"
              name="email"
              type="email"
              autoComplete="username"
              required
              value={email}
              onChange={(e) => setEmail(e.target.value)}
            />
            <Campo
              rotulo="Senha"
              name="senha"
              type="password"
              autoComplete="current-password"
              required
              value={senha}
              onChange={(e) => setSenha(e.target.value)}
            />

            {erro ? (
              // role=alert para o leitor de tela anunciar a falha de login.
              <p role="alert" className="text-sm font-medium text-danger">
                {erro.mensagem}
              </p>
            ) : null}

            <Botao type="submit" carregando={entrar.isPending}>
              Entrar
            </Botao>
          </form>
        </Cartao>

        <p className="mt-4 text-center text-xs text-white/60">
          Não há cadastro público. Peça seu acesso à coordenação.
          <br />
          <Link
            to="/primeiro-acesso"
            className="underline hover:text-verde-lima"
          >
            Primeiro acesso do sistema
          </Link>
        </p>
      </div>
    </div>
  );
}
