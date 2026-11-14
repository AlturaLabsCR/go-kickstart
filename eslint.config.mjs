import js from '@eslint/js';
import ts from 'typescript-eslint';

export default ts.config(
  { ignores: ['handlers/static/**', 'node_modules/**'] },
  js.configs.recommended,
  ...ts.configs.recommended,
  { files: ['frontend/**/*.ts'], rules: { '@typescript-eslint/no-non-null-assertion': 'off', 'no-undef': 'off' } },
);
