// Config endurecida (não é a do Airbnb, mas não é mais o mínimo recomendado
// sozinho). Regras novas existem pra pegar em CI o que a skill `seguranca`
// pede em prosa: sem localStorage/sessionStorage (sessão é cookie), sem
// dangerouslySetInnerHTML, sem promise solta, sem import cruzado entre
// core/components e pages.
import js from "@eslint/js";
import globals from "globals";
import reactPlugin from "eslint-plugin-react";
import reactHooks from "eslint-plugin-react-hooks";
import reactRefresh from "eslint-plugin-react-refresh";
import jsxA11y from "eslint-plugin-jsx-a11y";
import importPlugin from "eslint-plugin-import";
import tseslint from "typescript-eslint";

export default tseslint.config(
  {
    // shadcn é gerado; vite.config.{js,d.ts} e *.tsbuildinfo são artefatos do
    // `tsc -b` (project reference de tsconfig.node.json) — nunca código do
    // projeto (ver .dockerignore).
    ignores: ["dist", "src/components/ui/**", "vite.config.js", "vite.config.d.ts", "**/*.tsbuildinfo"],
  },
  {
    extends: [js.configs.recommended, ...tseslint.configs.recommendedTypeChecked],
    files: ["**/*.{ts,tsx}"],
    languageOptions: {
      ecmaVersion: 2020,
      globals: globals.browser,
      parserOptions: {
        projectService: true,
        tsconfigRootDir: import.meta.dirname,
      },
    },
    plugins: {
      react: reactPlugin,
      "react-hooks": reactHooks,
      "react-refresh": reactRefresh,
      "jsx-a11y": jsxA11y,
      import: importPlugin,
    },
    settings: {
      react: { version: "detect" },
    },
    rules: {
      ...reactHooks.configs.recommended.rules,
      ...jsxA11y.configs.recommended.rules,
      "react/react-in-jsx-scope": "off", // Vite/React 18 com o novo JSX transform
      "react/prop-types": "off", // tipagem é do TypeScript
      "react/no-danger": "error", // dangerouslySetInnerHTML só via componente único e sanitizado
      "react-refresh/only-export-components": ["warn", { allowConstantExport: true }],

      // Sessão é cookie HttpOnly — nada de token (nem qualquer outro dado)
      // guardado nestas APIs do navegador.
      "no-restricted-globals": [
        "error",
        { name: "localStorage", message: "Sessão é cookie HttpOnly — não guarde nada aqui (ver core/sessao.tsx)." },
        { name: "sessionStorage", message: "Sessão é cookie HttpOnly — não guarde nada aqui (ver core/sessao.tsx)." },
      ],

      "@typescript-eslint/no-explicit-any": "error",
      "@typescript-eslint/no-floating-promises": "error",
      "@typescript-eslint/consistent-type-imports": "warn",
      "@typescript-eslint/no-unused-vars": ["warn", { argsIgnorePattern: "^_" }],

      "import/no-cycle": "error",
      "import/no-restricted-paths": [
        "error",
        {
          zones: [
            // core/components/hooks nunca importam de pages — o fluxo é
            // sempre core/components -> pages, nunca o contrário.
            {
              target: ["./src/core", "./src/components", "./src/hooks", "./src/tipos"],
              from: ["./src/pages"],
            },
          ],
        },
      ],
    },
  },
);
