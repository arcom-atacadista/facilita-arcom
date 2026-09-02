// Estado global de UI que não é dado de servidor (isso é o TanStack Query) —
// só o que sobra: um contador manual de "ocupado" pra operação assíncrona que
// não passa por uma query/mutation (ex.: gerar um PDF no navegador).
import { create } from "zustand";

type EstadoUI = {
  ocupadoManual: number;
  iniciarOcupado: () => void;
  terminarOcupado: () => void;
};

export const useEstadoUI = create<EstadoUI>((set) => ({
  ocupadoManual: 0,
  iniciarOcupado: () => set((s) => ({ ocupadoManual: s.ocupadoManual + 1 })),
  terminarOcupado: () =>
    set((s) => ({ ocupadoManual: Math.max(0, s.ocupadoManual - 1) })),
}));
