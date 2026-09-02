# Template do relatório de auditoria

Nome do arquivo: `seguranca_{slug-do-projeto}_{AAAA-MM-DD}.md`.

```markdown
# Auditoria de segurança — {Nome do Projeto}

**Data:** {AAAA-MM-DD}  ·  **Classificação:** INTERNA | EXTERNA | MISTA
**Escopo:** {frontend/backend/ambos}  ·  **Metodologia:** revisão estática de
código (sem execução contra produção)

## Sumário executivo

{2-3 parágrafos em português simples — sem jargão técnico. O que foi olhado,
o que preocupa mais, se está OK pra ir pra produção ou não.}

**Ações imediatas (antes do próximo deploy):**
1. {ação mais urgente}
2. {segunda mais urgente}
3. {terceira}

### Resumo quantitativo

| Severidade | Quantidade |
|---|---|
| Crítica | N |
| Alta | N |
| Média | N |
| Baixa | N |
| Info | N |

## Achados detalhados

Ordenados por CVSS decrescente. Um bloco por achado:

### {ID} — {Título}

- **Severidade:** {CRITICA/ALTA/MEDIA/BAIXA/INFO} (CVSS {nota})
- **CWE:** {CWE-XXX}  ·  **Categoria:** {A01 - ...}
- **Onde:** `{arquivo}:{linha}`
- **Evidência:**
  ```
  {trecho de código real, máx. 5 linhas}
  ```
- **Impacto:** {o que um atacante consegue fazer, concretamente}
- **Correção:**
  ```
  {código corrigido ou passo a passo}
  ```
- **Status vs. relatório anterior:** {CONFIRMADO | MITIGADO | AGRAVADO | NOVO | —}

## Apêndice

- **Falsos positivos descartados:** {lista curta — o que foi considerado e
  por que não virou achado}
- **Fora de escopo:** {o que não foi auditado nesta rodada e por quê}
```

## Vereditos de segundo turno (quando aplicável)

Ao revisar os achados do primeiro turno com um agente novo, cada finding
recebe um veredito: `CONFIRMADA` / `CONFIRMADA_AJUSTADA` (severidade mudou) /
`FALSO_POSITIVO` / `DUPLICATA` / `NOVA`. Taxa de precisão =
`confirmadas / total_recebido`; se ficar abaixo de 70%, revise a metodologia
do primeiro turno antes de confiar no relatório.
