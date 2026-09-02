# 08 — Design System ARCOM (obrigatório em toda UI)

**Regra:** toda tela web sai com a identidade da ARCOM. Cores, tipografia, raios,
sombras e tom de voz vêm SÓ dos tokens abaixo (já mapeados no
`frontend/tailwind.config.js`). Nunca invente cor, fonte ou estilo solto.

Referência completa da marca: `design/ARCOM-Design-System.md`.

## Cores (classe Tailwind → uso)

| Classe                 | Hex       | Uso                                            |
|------------------------|-----------|------------------------------------------------|
| `verde-escuro`         | `#1F4033` | sidebar, hero, texto de autoridade, fundos escuros |
| `verde-arcom`          | `#007840` | ações primárias, links, botões, destaques      |
| `verde-lima`           | `#BAE64F` | acentos, promoções, friso de nav ativo         |
| `arcom-gray`           | `#636466` | corpo de texto, labels secundários             |
| `danger`               | `#D13D29` | erros, alertas, ações destrutivas              |
| `surface`              | `#F6F6F6` | **fundo da página** (nunca branco puro)        |
| `surface-border`       | `#E6E7E8` | borda de card / divisor                        |

Ex.: `bg-verde-arcom text-white`, `bg-surface`, `border border-surface-border`.
O verde é **acento**, não fundo dominante: interface em branco + cinza claro.

## Tipografia — Red Hat Display

Já é a fonte padrão (`font-sans`). Importe uma vez no CSS de entrada
(`src/index.css`), antes das diretivas do Tailwind:

```css
@import url('https://fonts.googleapis.com/css2?family=Red+Hat+Display:ital,wght@0,300..900;1,300..900&display=swap');
@tailwind base;
@tailwind components;
@tailwind utilities;
```

Hierarquia (peso / tamanho Tailwind aproximado):
- Display — `font-black text-5xl/6xl` (heroes, chamadas)
- Heading — `font-bold text-3xl/4xl` (títulos de seção)
- Subheading — `font-semibold text-xl/2xl` (card headers)
- Body — `font-normal text-base` (parágrafos)
- Label — `font-bold text-xs uppercase tracking-wider` (tags, metadados)
- Caption — `font-medium text-xs` (datas, SKUs)

## Raios / sombras / transições

- Raios: `rounded-md` (8px, botões/inputs/cards), `rounded-lg` (12px, modais),
  `rounded-full` (pills). Sem raios exagerados — a marca é sólida, não fofa.
- Sombras: `shadow-sm` (card em repouso), `shadow-md` (hover), `shadow-lg`
  (modal/dropdown). São verdes, sem black.
- Transições: `duration-fast` (hover), `duration-normal` (estado),
  `duration-slow` (entrada). Sem bounce/spring, sem loop decorativo.

## Componentes (padrão shadcn + tokens ARCOM)

Monte os componentes shadcn/Radix usando os tokens. Variantes esperadas:
- **Button:** primary (`bg-verde-arcom text-white`), secondary (contorno),
  ghost, danger (`bg-danger`), dark (`bg-verde-escuro`), accent (`bg-verde-lima
  text-verde-escuro`). Hover escurece ~15% (sem opacity); pressed `scale-95`.
- **Card:** branco, `border border-surface-border`, `shadow-sm`; hover
  interativo `-translate-y-0.5 shadow-md`. Variantes: default/brand/accent/outlined.
- **Badge / Tag:** pill (`rounded-full`), compacto.
- **Input / Select:** `rounded-md`, borda `surface-border`, com label/hint/erro.
- **Alert:** success / danger / warning / info / brand.

## Layout

- Fundo da página: `bg-surface` (nunca branco puro).
- Cards: brancos com `border-surface-border` + `shadow-sm`.
- Sidebar: `bg-verde-escuro`, logo em destaque.
- Top bar: branca com borda inferior `surface-border`.
- Nav ativo: friso lateral `border-l-[3px] border-verde-lima` + texto `verde-lima`.
- Densidade confortável (contexto B2B com dados), não minimalista demais.

## Ícones

`lucide-react` (já no stack). Stroke 2px, 18–20px em UI, 24px em navegação.
Cor herda do contexto (`currentColor`) ou `verde-arcom`/`arcom-gray`. Sem emoji.

## Logos

Quatro versões oficiais, em
`design/manual completo marca Arcom SA/1 - Logo/Logo Oficial/` (uma pasta por
fundo, com SVG e, quando aplicável, PNG/PDF):
- **Fundo Branco** — padrão, primeira escolha sobre superfícies claras.
- **Fundo Verde Escuro** — peças escuras/institucionais (sidebar, hero).
- **Fundo Verde** — quando o verde sólido é o fundo (escrita branca).
- **Fundo Verde Limão** — promoção/energia, com moderação.

Área de proteção = largura da letra "O" do logo. **Proibido** distorcer, girar,
inclinar, sombrear, trocar cores ou aplicar sobre foto.

## Tom de voz (para qualquer texto na UI)

- Direto e profissional, sem rodeios. Confiante mas acessível (parceiro B2B).
- **Sem emoji.** Cliente é o estabelecimento, não o consumidor final.
- "Faça seu pedido" (não "efetue"). "Produtos em estoque".
- Dinheiro: `R$ 1.250,00`. Quantidades: `24un`, `12cx`, `5kg`.
- Marca sempre **ARCOM** (caixa alta em destaque); em texto corrido "Arcom S/A".
