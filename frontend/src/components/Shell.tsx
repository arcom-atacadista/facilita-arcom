// Casca do app: barra lateral verde escuro com o friso lima no item ativo,
// topo branco e a área de conteúdo — o layout do Design System ARCOM
// (system-design/padroes/08-design-system.md).
import { useMutation, useQueryClient } from "@tanstack/react-query";
import type { ReactNode } from "react";
import { NavLink, Outlet, useNavigate } from "react-router-dom";
import { api } from "../core/api";
import { cn } from "../core/cn";
import { ROTULO_PAPEL } from "../core/formato";
import { useSessao } from "../core/sessao";
import type { Papel } from "../tipos/api";

const NIVEL: Record<Papel, number> = {
  aprendiz: 1,
  analista: 2,
  coordenacao: 3,
  gerencia: 4,
};

type ItemNav = {
  para: string;
  rotulo: string;
  /** Papel mínimo para o item aparecer. Esconder é conveniência: quem
   *  realmente barra o acesso é a rota do backend. */
  minimo?: Papel;
};

const NAVEGACAO: ItemNav[] = [
  { para: "/carteira", rotulo: "Carteira" },
  { para: "/acordos", rotulo: "Acordos" },
  { para: "/disparos", rotulo: "Régua de cobrança" },
  { para: "/mesa", rotulo: "Mesa de atendimento" },
  { para: "/politicas", rotulo: "Políticas", minimo: "coordenacao" },
  { para: "/usuarios", rotulo: "Usuários", minimo: "coordenacao" },
];

export function Shell() {
  const { usuario } = useSessao();
  const navigate = useNavigate();
  const queryClient = useQueryClient();

  const sair = useMutation({
    mutationFn: () => api.post("/sessao/logout"),
    // onSettled e não onSuccess: mesmo se o logout falhar no servidor, o
    // usuário tem que sair da tela — o contrário prende alguém logado.
    onSettled: () => {
      queryClient.clear();
      navigate("/entrar", { replace: true });
    },
  });

  if (!usuario) return null;

  const itens = NAVEGACAO.filter(
    (i) => !i.minimo || NIVEL[usuario.papel] >= NIVEL[i.minimo],
  );

  return (
    <div className="flex min-h-screen flex-col lg:flex-row">
      <nav
        aria-label="Menu principal"
        className="flex flex-col bg-verde-escuro lg:w-64"
      >
        <div className="border-b border-verde-lima/20 px-6 py-5">
          <p className="text-lg font-black tracking-tight text-white">
            Facilita <span className="text-verde-lima">ARCOM</span>
          </p>
          <p className="text-xs text-white/60">Crédito e Cobrança</p>
        </div>

        <ul className="flex flex-1 flex-wrap gap-1 p-3 lg:flex-col lg:flex-nowrap">
          {itens.map((item) => (
            <li key={item.para}>
              <NavLink
                to={item.para}
                className={({ isActive }) =>
                  cn(
                    "block border-l-[3px] px-4 py-2.5 text-sm transition-colors duration-fast",
                    isActive
                      ? "border-verde-lima bg-verde-lima/10 font-bold text-verde-lima"
                      : "border-transparent text-white/80 hover:bg-verde-lima/10 hover:text-white",
                  )
                }
              >
                {item.rotulo}
              </NavLink>
            </li>
          ))}
        </ul>

        <div className="border-t border-verde-lima/20 px-6 py-4 text-xs text-white/70">
          <p className="truncate font-bold text-white">
            {usuario.nome || usuario.email}
          </p>
          <p>
            {ROTULO_PAPEL[usuario.papel]}
            {usuario.codigoCobranca
              ? ` · carteira ${usuario.codigoCobranca}`
              : ""}
          </p>
          <p>
            {usuario.alcadaMaxima >= 100
              ? "Alçada sem limite"
              : `Alçada até ${usuario.alcadaMaxima}%`}
          </p>
          <button
            type="button"
            onClick={() => sair.mutate()}
            className="mt-3 w-full rounded-md border border-white/25 px-3 py-2 text-xs font-bold text-white transition-colors duration-fast hover:bg-white/10"
          >
            Sair
          </button>
        </div>
      </nav>

      <main className="flex-1 bg-surface p-5 lg:p-8">
        <Outlet />
      </main>
    </div>
  );
}

export function Cabecalho({
  titulo,
  descricao,
  acao,
}: {
  titulo: string;
  descricao?: string;
  acao?: ReactNode;
}) {
  return (
    <header className="mb-6 flex flex-wrap items-start justify-between gap-4">
      <div>
        <h1 className="text-xl font-black text-verde-escuro">{titulo}</h1>
        {descricao ? (
          <p className="mt-0.5 text-sm text-arcom-gray">{descricao}</p>
        ) : null}
      </div>
      {acao}
    </header>
  );
}
