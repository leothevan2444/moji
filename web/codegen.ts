import type { CodegenConfig } from "@graphql-codegen/cli";

const config: CodegenConfig = {
  schema: "../graphql/moji/schema.graphql",
  documents: ["src/**/*.graphql"],
  generates: {
    "src/graphql/generated/": {
      preset: "client",
      presetConfig: {
        gqlTagName: "graphql"
      }
    }
  },
  ignoreNoDocuments: true
};

export default config;
