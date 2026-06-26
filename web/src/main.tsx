import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import { Provider } from "urql";
import { App } from "./App";
import { client } from "./graphql/client";
import "./styles.scss";

createRoot(document.getElementById("root") as HTMLElement).render(
  <StrictMode>
    <Provider value={client}>
      <App />
    </Provider>
  </StrictMode>
);
