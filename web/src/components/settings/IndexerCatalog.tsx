import { useState } from "react";
import { useQuery } from "urql";
import { useTranslation } from "react-i18next";
import { IndexerDefinitionsDocument, IndexerSettingFieldsFragmentDoc } from "../../graphql/generated/graphql";
import { useFragment } from "../../graphql/generated/fragment-masking";
import { Drawer } from "../layout/Drawer";
import type { SiteEditorDefinition } from "./IndexerSiteEditor";
import { describeQueryError } from "../../services/queryError";

export function IndexerCatalog({ onSelect, onClose }: { onSelect: (definition: SiteEditorDefinition) => void; onClose: () => void }) {
  const { t } = useTranslation();
  const [query, setQuery] = useState(""); const [search, setSearch] = useState("");
  const [privacy, setPrivacy] = useState(""); const [language, setLanguage] = useState(""); const [capability, setCapability] = useState(""); const [offset, setOffset] = useState(0);
  const [{ data, fetching, error }, refresh] = useQuery({ query: IndexerDefinitionsDocument, variables: { input: { query: search, language, privacy, capability, offset, limit: 30 } }, requestPolicy: "network-only" });
  const page = data?.indexerDefinitions;
  return <Drawer visibleDrawer="settings" closing={false} title={t("indexers.addSite")} onClose={onClose}>
    <form className="indexer-filters" onSubmit={event => { event.preventDefault(); setOffset(0); setSearch(query.trim()); }}>
      <label><span>{t("indexers.search")}</span><input value={query} maxLength={256} onChange={event => setQuery(event.target.value)} /></label>
      <button type="submit">{t("indexers.search")}</button>
      <label><span>{t("indexers.privacy")}</span><select value={privacy} onChange={event => { setPrivacy(event.target.value); setOffset(0); }}><option value="">{t("indexers.all")}</option>{["public", "private", "semi-private"].map(value => <option key={value} value={value}>{t(`indexers.privacyKinds.${value}`)}</option>)}</select></label>
      <label><span>{t("indexers.language")}</span><input value={language} maxLength={64} placeholder="en-US" onChange={event => { setLanguage(event.target.value); setOffset(0); }} /></label>
      <label><span>{t("indexers.capability")}</span><select value={capability} onChange={event => { setCapability(event.target.value); setOffset(0); }}><option value="">{t("indexers.all")}</option><option value="search">{t("indexers.keywordSearch")}</option><option value="movie-search">{t("indexers.movieSearch")}</option><option value="tv-search">{t("indexers.tvSearch")}</option></select></label>
    </form>
    {error ? <p role="alert" className="settings-feedback tone-danger">{describeQueryError(error)} <button onClick={() => refresh({ requestPolicy: "network-only" })}>{t("common.retry")}</button></p> : null}
    <div aria-busy={fetching} className="indexer-list indexer-catalog-list">
      {page?.items.map(definition => <div className="indexer-row" key={`${definition.id}:${definition.digest}`}>
        <div className="indexer-row__content"><strong>{definition.name}</strong><span className="settings-note">{definition.language} · {t(`indexers.privacyKinds.${definition.privacy}`, { defaultValue: definition.privacy })}</span>{definition.description ? <p>{definition.description}</p> : null}{definition.warnings.length ? <details><summary>{t("indexers.ruleWarnings")}</summary>{definition.warnings.map((warning, index) => <p className="settings-note" key={index}>{warning}</p>)}</details> : null}</div>
        <button disabled={fetching} onClick={() => onSelect({ id: definition.id, name: definition.name, links: definition.links, settings: definition.settings.map(setting => useFragment(IndexerSettingFieldsFragmentDoc, setting)) })}>{t("indexers.configure")}</button>
      </div>)}
      {!fetching && page?.totalCount === 0 ? <p>{t("indexers.noMatch")}</p> : null}
      {fetching && !page ? <p role="status">{t("common.loading")}</p> : null}
    </div>
    {page ? <div className="indexer-actions"><button disabled={fetching || offset === 0} onClick={() => setOffset(value => Math.max(0, value - 30))}>{t("indexers.previous")}</button><span>{t("indexers.catalogCount", { total: page.totalCount })}</span><button disabled={fetching || offset + 30 >= page.totalCount} onClick={() => setOffset(value => value + 30)}>{t("indexers.next")}</button></div> : null}
  </Drawer>;
}
