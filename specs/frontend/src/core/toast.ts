// Notificações — sonner (é o toast oficial do shadcn hoje). Helpers
// semânticos + deduplicação: 5 requisições falhando ao mesmo tempo geram
// 1 toast, não 5 (janela de 5s por chave "tipo:mensagem").
import { toast } from "sonner";

const JANELA_DEDUP_MS = 5_000;
const LIMPEZA_MS = 10_000;

const recentes = new Map<string, number>();

function jaMostrado(chave: string): boolean {
  const agora = Date.now();
  const ultima = recentes.get(chave);
  if (ultima && agora - ultima < JANELA_DEDUP_MS) return true;

  recentes.set(chave, agora);
  for (const [k, quando] of recentes) {
    if (agora - quando > LIMPEZA_MS) recentes.delete(k);
  }
  return false;
}

function mostrar(
  tipo: "success" | "error" | "warning" | "info",
  texto: string,
) {
  const chave = `${tipo}:${texto}`;
  if (jaMostrado(chave)) return;
  toast[tipo](texto, { id: chave });
}

export const notificar = {
  sucesso: (texto: string) => mostrar("success", texto),
  erro: (texto: string) => mostrar("error", texto),
  aviso: (texto: string) => mostrar("warning", texto),
  info: (texto: string) => mostrar("info", texto),
};
