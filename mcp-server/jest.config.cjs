/** @type {import('jest').Config} */
module.exports = {
  preset: "ts-jest",
  testEnvironment: "node",
  testMatch: ["**/__tests__/**/*.test.ts"],
  moduleNameMapper: {
    // Strip .js extensions from imports so ts-jest can resolve .ts files
    "^(\\.{1,2}/.*)\\.js$": "$1",
    // Map ESM-only MCP SDK imports to their CJS equivalents
    "^@modelcontextprotocol/sdk/server/mcp\\.js$":
      "<rootDir>/node_modules/@modelcontextprotocol/sdk/dist/cjs/server/mcp.js",
    "^@modelcontextprotocol/sdk/server/stdio\\.js$":
      "<rootDir>/node_modules/@modelcontextprotocol/sdk/dist/cjs/server/stdio.js",
  },
  transform: {
    "^.+\\.tsx?$": [
      "ts-jest",
      {
        tsconfig: {
          module: "CommonJS",
          moduleResolution: "node",
          esModuleInterop: true,
          strict: true,
          target: "ES2022",
          isolatedModules: true,
        },
        diagnostics: false,
      },
    ],
  },
};
