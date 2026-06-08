import { cacheExchange, createClient, fetchExchange } from "urql";

export const client = createClient({
  url: "/graphql",
  preferGetMethod: false,
  exchanges: [cacheExchange, fetchExchange]
});
