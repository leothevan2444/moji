import { useEffect, useState } from "react";
import { createPortal } from "react-dom";
import { useQuery } from "urql";
import { useTranslation } from "react-i18next";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { faWrench, faPlus } from "@fortawesome/free-solid-svg-icons";
import { IndexerDefinitionsDocument, IndexerSettingFieldsFragmentDoc, IndexerSourceKind } from "../../graphql/generated/graphql";
import { useFragment } from "../../graphql/generated/fragment-masking";
import type { IndexerSite } from "../../hooks/useIndexerManagement";
import { Drawer } from "../layout/Drawer";
import { IndexerPrivacyBadge } from "../indexers/IndexerPrivacyBadge";
import type { SiteEditorDefinition } from "./IndexerSiteEditor";
import { describeQueryError } from "../../services/queryError";

export function IndexerCatalog({ onSelect, onQuickAdd, onClose, sites, busy, mutationError }: {
  onSelect: (definition: SiteEditorDefinition) => void;
  onQuickAdd: (definition: SiteEditorDefinition) => Promise<boolean>;
  onClose: () => void;
  sites: IndexerSite[];
  busy: boolean;
  mutationError: unknown;
}) {
  const { t } = useTranslation();
  const [query, setQuery] = useState("");
  const [search, setSearch] = useState("");
  const [privacy, setPrivacy] = useState("");
  const [language, setLanguage] = useState("");
  const [capability, setCapability] = useState("");
  const [offset, setOffset] = useState(0);
  // Fence a second click after a confirmed save while the authoritative list
  // rereads. This is not a replacement for the persisted instance snapshot.
  const [acknowledged, setAcknowledged] = useState<Set<string>>(new Set());
  useEffect(() => {
    setAcknowledged(current => {
      const next = new Set([...current].filter(id => !sites.some(site => site.kind === IndexerSourceKind.Builtin && site.definitionId === id)));
      return next.size === current.size ? current : next;
    });
  }, [sites]);
  const [{ data, fetching, error }, refresh] = useQuery({ query: IndexerDefinitionsDocument, variables: { input: { query: search, language, privacy, capability, offset, limit: 30 } }, requestPolicy: "network-only" });
  const page = data?.indexerDefinitions;
  const close = () => { if (!busy) onClose(); };
  const failure = mutationError ?? error;
  return createPortal(<Drawer className="indexer-catalog" visibleDrawer="settings" closing={false} title={t("indexers.addSite")} closeDisabled={busy} onClose={close}>
    <form className="indexer-filters" onSubmit={event => { event.preventDefault(); setOffset(0); setSearch(query.trim()); }}>
      <label><span>{t("indexers.search")}</span><input value={query} maxLength={256} onChange={event => setQuery(event.target.value)} /></label>
      <button type="submit">{t("indexers.search")}</button>
      <label><span>{t("indexers.privacy")}</span><select value={privacy} onChange={event => { setPrivacy(event.target.value); setOffset(0); }}><option value="">{t("indexers.all")}</option>{["public", "private", "semi-private"].map(value => <option key={value} value={value}>{t(`indexers.privacyKinds.${value}`)}</option>)}</select></label>
      <label><span>{t("indexers.language")}</span><input value={language} maxLength={64} placeholder="en-US" onChange={event => { setLanguage(event.target.value); setOffset(0); }} /></label>
      <label><span>{t("indexers.capability")}</span><select value={capability} onChange={event => { setCapability(event.target.value); setOffset(0); }}><option value="">{t("indexers.all")}</option><option value="search">{t("indexers.keywordSearch")}</option><option value="movie-search">{t("indexers.movieSearch")}</option><option value="tv-search">{t("indexers.tvSearch")}</option></select></label>
    </form>
    {failure ? <p role="alert" className="settings-feedback tone-danger">{describeQueryError(failure)}{error ? <button onClick={() => refresh({ requestPolicy: "network-only" })}>{t("common.retry")}</button> : null}</p> : null}
    <div aria-busy={fetching || busy} className="indexer-list indexer-catalog-list">
      {page?.items.map(definition => {
        const instances = sites.filter(site => site.kind === IndexerSourceKind.Builtin && site.definitionId === definition.id);
        const added = instances.length > 0 || acknowledged.has(definition.id);
        const editorDefinition = { id: definition.id, name: definition.name, links: definition.links, settings: definition.settings.map(setting => useFragment(IndexerSettingFieldsFragmentDoc, setting)) };
        const configureLabel = t(added ? "indexers.addAnother" : "indexers.configure");
        return <div className="indexer-row" key={`${definition.id}:${definition.digest}`} data-definition-id={definition.id}>
          <div className="indexer-row__content">
            <div className="indexer-row__identity"><strong>{/^https?:\/\//i.test(definition.links[0] ?? "") ? <a href={definition.links[0]} target="_blank" rel="noreferrer">{definition.name}</a> : definition.name}</strong><span className="indexer-language">{definition.language}</span><IndexerPrivacyBadge privacy={definition.privacy} /></div>
            {definition.description ? <p className="indexer-description" title={definition.description}>{definition.description}</p> : null}
            {definition.categories.length ? <span className="indexer-categories" title={definition.categories.join(" · ")}><span>{t("indexers.categories")}: </span>{definition.categories.join(" · ")}</span> : null}
            {definition.warnings.length ? <details><summary>{t("indexers.ruleWarnings")}</summary>{definition.warnings.map((warning, index) => <p className="settings-note" key={index}>{warning}</p>)}</details> : null}
          </div>
          <div className="indexer-row__controls">
          <div className="indexer-row__actions">
            {definition.privacy === "public" && !definition.loginSupported && !added ? <button className="icon-button" title={t("indexers.addAndEnable")} aria-label={t("indexers.addAndEnable")} disabled={fetching || busy} onClick={async () => { if (await onQuickAdd(editorDefinition)) setAcknowledged(current => new Set(current).add(definition.id)); }}><FontAwesomeIcon icon={faPlus} aria-hidden="true" /></button> : null}
            <button className="icon-button" title={configureLabel} aria-label={configureLabel} disabled={fetching || busy} onClick={() => onSelect(editorDefinition)}><FontAwesomeIcon icon={faWrench} aria-hidden="true" /></button>
          </div>
          {added ? <span className="indexer-added">{t("indexers.alreadyAdded")}{instances.length ? ` · ${t(instances.some(site => site.enabled) ? "indexers.enabledState" : "indexers.disabledState")}` : ""}</span> : null}
          </div>
        </div>;
      })}
      {!fetching && page?.totalCount === 0 ? <p>{t("indexers.noMatch")}</p> : null}
      {fetching && !page ? <p role="status">{t("common.loading")}</p> : null}
    </div>
    {page ? <div className="indexer-actions"><button disabled={fetching || offset === 0} onClick={() => setOffset(value => Math.max(0, value - 30))}>{t("indexers.previous")}</button><span>{t("indexers.catalogCount", { total: page.totalCount })}</span><button disabled={fetching || offset + 30 >= page.totalCount} onClick={() => setOffset(value => value + 30)}>{t("indexers.next")}</button></div> : null}
  </Drawer>, document.body);
}
