# ARCOM Design System 

**Empresa:** ARCOM S/A — Atacadista Distribuidor  
**Sede:** Uberlândia, Minas Gerais, Brasil  
**Segmento:** Distribuição atacadista — bebidas, alimentos, limpeza, higiene, hortifruti  
**Branding:** vbiasi design (consultoria de marca)

---

## Sobre a Empresa

A ARCOM é uma distribuidora atacadista com décadas de história em Uberlândia, MG. Atende supermercados, mercearias, bares, restaurantes e outros estabelecimentos comerciais da região. Seus valores institucionais são:

> **Extensão (praça) · Variedade (mix) · Distribuição · Ética · Solidez · Compromisso · Agilidade · Eficiência · Profissionalismo**

A **personalidade da marca** é: Confiante, Honesta, Comprometida, Amigável.

**Fontes dos materiais:**
- `uploads/ManualMarca_ARCOM.pdf` — Manual oficial de uso da marca (21 páginas)
- `uploads/BrandingARCOM.pdf` — Estratégia de branding e criação da marca (44 páginas, vbiasi design)
- `uploads/Paleta de Cores.pdf` — Especificações de cor (imagem, sem texto extraível)
- `uploads/BrandSheet.pdf` — Folha de marca (imagem, sem texto extraível)
- `assets/Logo_Fundo_Branco.svg`, `assets/Logo_Fundo_Verde.svg`, `assets/Logo_Fundo_VerdeEsc.svg`, `assets/Logo_Fundo_VerdeLim.svg` — Logos vetoriais oficiais (cores aplicadas como atributos `fill`)
- `uploads/README.md` — Instruções de instalação da fonte Red Hat Display

---

## CONTENT FUNDAMENTALS

### Tom de Voz
- **Direto e profissional** — sem rodeios, foco em clareza e eficiência
- **Confiante mas acessível** — fala como parceiro de negócios, não como corporação distante
- **Sem emoji** — a comunicação da ARCOM é limpa e institucional
- **Linguagem B2B** — o cliente é o estabelecimento, não o consumidor final

### Nomenclatura
| Forma correta | Formas incorretas |
|---|---|
| ARCOM (caixa alta em destaque) | ArCom · arcom · Arcom |
| Arcom S/A (em texto corrido) | arcom s.a. · ARCOM SA |

### Casing
- Títulos e chamadas: **caixa alta** ou **Title Case** para impacto
- Corpo de texto: sentença normal
- Labels de categoria, SKUs, seções: `MAIÚSCULAS` com `letter-spacing`

### Linguagem Típica
- "Faça seu pedido" (não "efetue seu pedido")
- "Produtos em estoque" (direto, sem adjetivos desnecessários)
- "Variedade, Agilidade, Compromisso" (tríade de valores da marca)
- Valores monetários: `R$ 1.250,00` (padrão brasileiro, espaço após R$)
- Quantidades: `24un`, `12cx`, `5kg` (unidade junto ao número)

---

## VISUAL FOUNDATIONS

### Paleta de Cores

| Token | Hex | Nome | Uso |
|---|---|---|---|
| `--color-verde-escuro` | `#1F4033` | Verde Escuro | Fundos escuros, texto primário, autoridade |
| `--color-verde-arcom` | `#007840` | Verde ARCOM | Ações primárias, links, botões, destaques |
| `--color-verde-lima` | `#BAE64F` | Verde Lima / Energia | Acentos, promoções, barras decorativas |
| `--color-white` | `#FFFFFF` | Branco | Fundo de cards, espaços limpos |
| `--color-gray-600` | `#636466` | Cinza Texto | Corpo de texto, labels secundários |
| `--color-danger` | `#D13D29` | Vermelho | Erros, alertas, ações destrutivas |

**Verde ARCOM** (#007840) transmite confiança e solidez (verde institucional). **Verde Lima** (#BAE64F) é a energia e o movimento — usado como accent em comunicação.

### Tipografia

**Fonte primária: Red Hat Display** (fonte open-source da Red Hat, licença OFL, distribuída oficialmente via Google Fonts — não é uma substituição, é a fonte de produção)  
- Moderna, humanista, legível em todas as escalas
- Usada em **todos** os elementos: chamadas, títulos, corpo, labels
- Pesos utilizados: 300 Light → 900 Black
- Itálico disponível em todos os pesos

**Hierarquia:**
- Display (Black 900, 48–64px): chamadas principais, heroes
- Heading (Bold 700, 28–40px): títulos de seção
- Subheading (SemiBold 600, 18–24px): subtítulos, card headers
- Body (Regular 400, 15–16px): parágrafos, listas
- Label (Bold 700, 11–12px, UPPERCASE, letter-spacing 0.1em): categorias, tags, metadados
- Caption (Medium 500, 12–13px): datas, SKUs, notas

### Layout e Composição

- **Fundo de página:** `#F6F6F6` (cinza levíssimo) — nunca branco puro no fundo
- **Cards:** branco `#FFFFFF` com `border: 1px solid #E6E7E8` + `box-shadow: shadow-sm`
- **Sidebar:** `#1F4033` (verde escuro) com logo em destaque
- **Top bar:** branco com `border-bottom: 1px solid #E6E7E8`
- **Acento de navegação ativo:** friso lateral `3px solid #BAE64F` + texto `#BAE64F`
- **Densidade:** confortável mas eficiente — não minimalista demais (contexto B2B com dados)

### Bordas e Raios

- Botões, inputs, cards pequenos: `border-radius: 8px` (--radius-md)
- Badges, tags, pills: `border-radius: 9999px` (--radius-full)
- Modais, painéis grandes: `border-radius: 12px` (--radius-lg)
- **Sem raios exagerados** — a marca é séria e sólida, não fofa

### Sombras

Baseadas em verde escuro (`rgba(31,64,51,…)`) para coerência cromática:
- `shadow-sm`: elevação sutil para cards em repouso
- `shadow-md`: hover/focus de cards interativos
- `shadow-lg`: modais e dropdowns
- **Sem sombras com black** — mantém temperatura quente e coerência com a paleta

### Animações e Transições

- **Fast (150ms ease):** hover de botões, mudança de cor
- **Normal (250ms ease):** expansão de menus, transições de estado
- **Slow (400ms ease):** entradas de página, modais
- **Sem bounce ou spring** — a marca é profissional, não lúdica
- **Sem animações decorativas em loop**

### Hover / Press States

- Botões: escurece a cor de fundo ~15% (sem opacity)
- Cards interativos: `translateY(-2px)` + `shadow-md`
- Botão primário pressed: `scale(0.97)`
- Links e textos: cor vai de `--color-gray-600` para `--color-verde-arcom`

### Imagens e Iconografia

- Fotos devem ser **nítidas, bem iluminadas**, com predominância de tons naturais/neutros
- **Nunca aplicar a marca sobre fotos** (conforme manual)
- Fundos de fotos: sempre lisos ou muito suaves
- Sem gradientes de fundo nas versões institucionais da marca

### Uso de Cor

- Interface principal em **branco + cinza claro** — o verde como acento, não fundo dominante
- **Verde escuro como fundo:** sidebar, hero sections, banners institucionais
- **Verde lima:** friso decorativo, badges de promoção, accent em comunicação — nunca como fundo de página inteira
- **Transparência:** raramente usada; quando sim, `rgba(31,64,51,…)` para sobreposições

---

## ICONOGRAFIA

O brand guide não especifica um sistema de ícones proprietário. Recomendações:

- **Lucide Icons** (CDN) — stroke 2px, sem fill, estilo consistente com a tipografia Red Hat Display
- Tamanho padrão: 18–20px em contexto de UI, 24px em navegação, 32px+ em contexto ilustrativo
- Cor: herda do contexto (`currentColor`) ou usa `--color-verde-arcom` / `--color-gray-600`
- **Sem emoji** em contextos institucionais
- Unicode não é usado como ícone funcional

### Logos

As quatro versões cromáticas oficiais. **Cada versão se encaixa melhor em um contexto** — a escolha deve seguir a recomendação abaixo (critério: contraste, hierarquia e coerência com o fundo da peça).

#### 1. Fundo Branco — wordmark em verde ARCOM, friso superior vermelho
![Logo fundo branco](https://www.arcom.com.br/imagens/produtos/Logo_Fundo_Branco.png)

**Versão principal / padrão.** Use sempre que possível: documentos, apresentações claras, cabeçalhos de e-mail, cards brancos, papelaria. É a aplicação de maior legibilidade e a primeira escolha do UI.
`assets/Logo_Fundo_Branco.svg`

#### 2. Fundo Verde Escuro (#1F4033) — wordmark em verde ARCOM, friso superior vermelho
![Logo fundo verde escuro](https://www.arcom.com.br/imagens/produtos/Logo_Fundo_VerdeEsc.png)

**Peças escuras e institucionais.** Sidebar do portal, hero sections, banners de rodapé, capas de apresentação, materiais premium. Preserva o friso vermelho, mantendo a marca completa sobre fundo escuro.
`assets/Logo_Fundo_VerdeEsc.svg`

#### 3. Fundo Verde ARCOM (#007840) — escrita branca
![Logo fundo verde](https://www.arcom.com.br/imagens/produtos/Logo_Fundo_Verde.png)

**Blocos de cor sólida da marca.** Botões grandes, faixas de destaque, cartões de identidade visual, aplicações onde o verde institucional é o fundo dominante. A escrita branca garante contraste máximo.
`assets/Logo_Fundo_Verde.svg`

#### 4. Fundo Verde Lima (#BAE64F) — wordmark e frisos em verde ARCOM
![Logo fundo verde lima](https://www.arcom.com.br/imagens/produtos/Logo_Fundo_VerdeLim.png)

**Contextos de energia / promoção.** Badges de campanha, chamadas de ofertas, adesivos, comunicação sazonal. Use com moderação — o verde-lima é acento, nunca fundo de página inteira.
`assets/Logo_Fundo_VerdeLim.svg`

#### Versões monocromáticas (aplicações restritas)

| Arquivo | Uso |
|---|---|
| `assets/Logo_Mono_Dark.svg` | Monocromático escuro — NF, carimbos, fax, gravação a 1 cor |
| `assets/Logo_Mono_White.svg` | Monocromático branco — fundos escuros/fotográficos sem o verde oficial |

**Recomendação do UI:** na dúvida, use a **versão fundo branco** (#1) sobre superfícies claras e a **fundo verde escuro** (#2) sobre superfícies escuras — são as duas que preservam todas as cores da marca. As versões #3 e #4 são para quando o próprio bloco de cor da marca é o fundo.

**Área de proteção:** espaço mínimo ao redor do logo = largura da letra "O" do logotipo  
**Tamanho mínimo:** 30mm de largura (impresso) / equivalente em digital  
**Proibido:** distorcer, girar, inclinar, adicionar sombra, alterar cores, aplicar sobre fotos

---

## COMPONENTES

Todos em `components/core/`. Namespace: `window.ArcomDesignSystem_b0b8d1`

| Componente | Arquivo | Descrição |
|---|---|---|
| `Button` | `Button.jsx` | Ação primária — primary / secondary / ghost / danger / dark / accent |
| `Badge` | `Badge.jsx` | Status / contagem — inline pill compacto |
| `Tag` | `Tag.jsx` | Filtro / categoria — pill togglável |
| `Card` | `Card.jsx` | Container de conteúdo — default / subtle / brand / accent / outlined |
| `CardHeader` | `Card.jsx` | Cabeçalho de card |
| `CardDivider` | `Card.jsx` | Divisor horizontal de card |
| `Input` | `Input.jsx` | Campo de texto com label, hint, erro, prefixo/sufixo |
| `Select` | `Input.jsx` | Dropdown estilizado |
| `Alert` | `Alert.jsx` | Feedback inline — success / danger / warning / info / brand |

---

## UI KITS

### Portal do Cliente (`ui_kits/portal/index.html`)

Interface B2B para clientes atacadistas da ARCOM fazerem pedidos:
- **Dashboard:** métricas do mês, pedidos recentes, alertas
- **Catálogo:** grade de produtos com filtro por categoria e busca, controle de quantidade, adicionar ao carrinho
- **Meus Pedidos:** histórico com status
- **Carrinho:** resumo de itens + confirmação de pedido

---

## GUIDELINES (em `guidelines/`)

| Arquivo | Grupo | Conteúdo |
|---|---|---|
| `brand-logo.card.html` | Brand | Logo nas 4 versões cromáticas de fundo |
| `brand-logo-mono.card.html` | Brand | Logo monocromático claro/escuro |
| `colors-brand.card.html` | Colors | Paleta principal (verde escuro, ARCOM, lima) |
| `colors-neutral.card.html` | Colors | Escala de cinzas |
| `colors-semantic.card.html` | Colors | Danger, Success, Warning |
| `type-display.card.html` | Type | Display e heading — pesos e tamanhos |
| `type-body.card.html` | Type | Body, caption, label, mono |
| `type-weights.card.html` | Type | Espectro de pesos Red Hat Display |
| `spacing.card.html` | Spacing | Escala de espaçamento (base 4px) |
| `spacing-radius.card.html` | Spacing | Raios de borda |
| `spacing-shadows.card.html` | Spacing | Sistema de sombras |

---

## TOKENS

125 tokens CSS em `tokens/`:
- `tokens/colors.css` — cores de marca, neutros, semânticas
- `tokens/typography.css` — famílias, pesos, escala de tamanhos, line-heights
- `tokens/spacing.css` — escala de espaçamento, raios, sombras, z-index, transições
- `tokens/fonts.css` — `@import` do Google Fonts (Red Hat Display)

**Importar tudo:** `<link rel="stylesheet" href="styles.css">`

---

## NOTAS E RESSALVAS

> ✅ **Fonte:** Red Hat Display é uma fonte open-source (OFL) mantida pela Red Hat — a versão servida via Google Fonts em `tokens/fonts.css` é o arquivo oficial, não uma substituição. `uploads/README.md` descreve um caminho alternativo de self-hosting (`red-hat-display.css` local), útil apenas se a ARCOM preferir não depender do Google Fonts em produção; nenhum arquivo proprietário adicional é necessário.

> ✅ **Cores das logos SVG:** Os `<defs>` dos SVGs originais vinham vazios (estilos perdidos na exportação do Illustrator). As cores foram restauradas usando os tokens oficiais da marca (`--color-verde-arcom: #007840`, `--color-verde-escuro: #1F4033`, `--color-verde-lima: #BAE64F`, `--color-white`), já extraídos de `Paleta de Cores.pdf` e `ManualMarca_ARCOM.pdf` em `tokens/colors.css` — mesma paleta usada nos demais componentes do sistema, e não um valor reconstruído isoladamente.

> ✅ **Ícones:** O brand guide não especifica um sistema de ícones proprietário. Lucide Icons (stroke 2px, sem fill) foi adotado como padrão do UI Kit — ver `## ICONOGRAFIA` acima.
