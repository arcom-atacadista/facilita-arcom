import { Link } from "react-router-dom";

export default function NaoEncontrado() {
  return (
    <main className="min-h-screen grid place-items-center p-6 text-center">
      <div>
        <p className="text-xs font-bold uppercase tracking-wider text-verde-arcom">
          ARCOM
        </p>
        <h1 className="mt-1 text-3xl font-black text-verde-escuro">
          Página não encontrada
        </h1>
        <p className="mt-2 text-sm text-arcom-gray">
          O endereço que você acessou não existe.
        </p>
        <Link
          to="/"
          className="mt-4 inline-block text-sm font-medium text-verde-arcom underline"
        >
          Voltar ao início
        </Link>
      </div>
    </main>
  );
}
