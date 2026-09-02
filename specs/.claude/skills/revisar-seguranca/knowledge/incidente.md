# Achado crítico ativo — o que fazer além de reportar

Use isto quando a auditoria encontra algo que já pode estar sendo explorado
agora (segredo vazado publicamente, IDOR trivial em produção) — não é só
"reportar e seguir", é agir.

## Rotação de credencial

| Credencial exposta | Ação | Prioridade |
|---|---|---|
| Segredo de assinatura de sessão/JWT | Gerar novo (`openssl rand -base64 48`), invalida todas as sessões ativas | Imediata |
| Senha de banco | Trocar no Postgres + `.env.prod` + reiniciar backend | Imediata |
| Chave de API de terceiro | Revogar no provedor, gerar nova | Imediata |
| Token de webhook | Rotacionar no provedor e no backend | Alta |

Depois de rotacionar: monitore tentativa de autenticação falhando com a
credencial antiga por um tempo — indica que alguém (ou algo automatizado)
tinha a credencial e ainda está tentando usá-la.

## Preservar evidência antes de corrigir

Antes de aplicar o fix, registre (fora do repositório, num lugar seguro):
- o commit/timestamp em que o problema foi introduzido (`git log -p -S`
  procurando pela credencial/padrão vulnerável);
- se há indício de exploração (log de acesso incomum ao endpoint afetado, se
  existir);
- o texto do achado tal como reportado (pra comparação de baseline depois).

## Narrativa de cadeia de ataque (kill chain) — pra explicar o risco

Formato curto, pra quem não é técnico entender a gravidade:

```
1. Atacante encontra {onde} (ex.: chave no bundle JS público)
2. Com isso, consegue {o que} (ex.: chamar endpoint X diretamente)
3. Isso permite {impacto} (ex.: ler/alterar dado de outro usuário)
4. Resultado: {consequência de negócio, em uma frase}
```

## LGPD — se dado pessoal foi exposto

Se o achado envolve PII acessível sem autorização (severidade crítica por
`10-seguranca.md`), há prazo legal de notificação à ANPD (72h, Art. 48) em
caso de incidente confirmado — avise quem no projeto é responsável por essa
decisão; não é decisão técnica isolada.

## Checklist de regressão (rodadas seguintes)

- [ ] Os achados CRÍTICOS/ALTOS da última auditoria foram corrigidos?
- [ ] Alguma migration nova introduziu policy/campo permissivo demais?
- [ ] Alguma rota nova ficou fora do grupo autenticado?
- [ ] Credenciais foram rotacionadas conforme recomendado?
- [ ] Headers de segurança/CSP continuam presentes?

Cadência sugerida: reteste focado (só CRÍTICO/ALTO) a cada 30 dias; auditoria
completa a cada 90 dias; imediato após incidente confirmado.
