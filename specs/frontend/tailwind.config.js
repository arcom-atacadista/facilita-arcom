/** @type {import('tailwindcss').Config} */
// Tokens do Design System ARCOM mapeados pro Tailwind.
// Regra: toda cor/tipografia/raio/sombra sai daqui. Nada de valor solto.
export default {
  content: ["./index.html", "./src/**/*.{js,ts,jsx,tsx}"],
  theme: {
    extend: {
      colors: {
        "verde-escuro": "#1F4033",  // fundos escuros, sidebar, texto de autoridade
        "verde-arcom":  "#007840",  // ações primárias, links, botões, destaques
        "verde-lima":   "#BAE64F",  // acentos, promoções, friso de nav ativo
        "arcom-gray":   "#636466",  // corpo de texto, labels secundários
        danger:         "#D13D29",  // erros, alertas, ações destrutivas
        surface:        "#F6F6F6",  // fundo de página (NUNCA branco puro)
        "surface-border": "#E6E7E8", // borda de cards / divisores
      },
      fontFamily: {
        sans: ['"Red Hat Display"', "system-ui", "sans-serif"],
      },
      borderRadius: {
        md: "8px",     // botões, inputs, cards pequenos
        lg: "12px",    // modais, painéis grandes
        full: "9999px",// badges, tags, pills
      },
      boxShadow: {
        // sombras com verde escuro (coerência cromática, sem black)
        sm: "0 1px 2px rgba(31,64,51,0.08)",
        md: "0 4px 12px rgba(31,64,51,0.12)",
        lg: "0 12px 32px rgba(31,64,51,0.18)",
      },
      transitionDuration: {
        fast: "150ms",   // hover de botões
        normal: "250ms", // menus, transições de estado
        slow: "400ms",   // entradas de página, modais
      },
    },
  },
  plugins: [require("tailwindcss-animate")],
};
