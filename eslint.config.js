import js from "@eslint/js";
import tseslint from "typescript-eslint";
import prettier from "eslint-config-prettier";
import globals from "globals";

export default [
  { ignores: ["node_modules/**", "openspec/**"] },
  js.configs.recommended,
  ...tseslint.configs.recommended,
  // Must come last: it switches off every rule that would fight the formatter.
  prettier,
  {
    files: ["**/*.ts"],
    rules: {
      "@typescript-eslint/no-unused-vars": [
        "error",
        {
          argsIgnorePattern: "^_",
          varsIgnorePattern: "^_",
          caughtErrorsIgnorePattern: "^_",
        },
      ],
      "@typescript-eslint/no-explicit-any": "warn",
    },
  },
  {
    // Plain Node scripts have no TypeScript project, and need Node globals.
    files: ["**/*.js"],
    languageOptions: { globals: globals.node },
  },
];
