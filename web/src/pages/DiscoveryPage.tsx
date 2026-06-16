import { FormEvent } from "react";
import { formatBytes, formatRelativeDate } from "../utils";
import type {
  SearchDocumentQuery
} from "../graphql/generated/graphql";

type JackettResult = SearchDocumentQuery["jackettSearch"][number];

interface DiscoveryPageProps {
  jackettQuery: string;
  searching: boolean;
  searchError: Error | null;
  searchResults: JackettResult[];
  deferredJackettQuery: string;
  pendingAddId: string | null;
  onQueryChange: (value: string) => void;
  onSubmit: (event: FormEvent<HTMLFormElement>) => void;
  onAdd: (result: JackettResult) => void;
  onOpenHelp: () => void;
}

export function DiscoveryPage({
  jackettQuery,
  searching,
  searchError,
  searchResults,
  deferredJackettQuery,
  pendingAddId,
  onQueryChange,
  onSubmit,
  onAdd,
  onOpenHelp
}: DiscoveryPageProps) {
  return (
    <>
      <section className="section-band">
        <div className="band-head">
          <div>
            <p className="section-kicker">发现</p>
            <h2>Jackett 搜索</h2>
          </div>
          <p className="band-note">搜索候选后直接创建 Moji task。</p>
        </div>

        <form className="discovery-bar" onSubmit={onSubmit}>
          <input
            value={jackettQuery}
            onChange={(event) => onQueryChange(event.target.value)}
            placeholder="输入番号、标题、女优或关键词"
          />
          <button type="submit" disabled={searching || jackettQuery.trim() === ""}>
            {searching ? "搜索中" : "搜索"}
          </button>
        </form>

        {searchError ? <p className="inline-error">{searchError.message}</p> : null}

        <div className="discovery-results">
          {searchResults.map((result) => (
            <article key={`${result.tracker}-${result.link}`} className="candidate-card">
              <div className="candidate-card__head">
                <div>
                  <h3>{result.title}</h3>
                  <p>
                    {result.tracker} · {formatBytes(Number(result.size) || 0)} · {result.seeders} seeders
                  </p>
                </div>
                <span className="status-chip tone-info">{result.categoryDesc || "候选"}</span>
              </div>
              <div className="candidate-card__foot">
                <span>{formatRelativeDate(result.publishDate)}</span>
                <div className="inline-actions">
                  <a href={result.link} target="_blank" rel="noreferrer">
                    原始链接
                  </a>
                  <button type="button" onClick={() => onAdd(result)} disabled={pendingAddId === result.link}>
                    {pendingAddId === result.link ? "添加中" : "创建任务"}
                  </button>
                </div>
              </div>
            </article>
          ))}
          {deferredJackettQuery && !searching && searchResults.length === 0 ? (
            <article className="empty-card empty-card--wide">
              <h3>没有候选</h3>
              <p>Jackett 没有返回结果，换个关键词再试。</p>
            </article>
          ) : null}
          {!deferredJackettQuery ? (
            <article className="empty-card empty-card--wide">
              <h3>先搜索</h3>
              <p>输入关键词后会在这里列出候选项。</p>
            </article>
          ) : null}
        </div>
      </section>

      <section className="section-band section-band--preview">
        <div className="band-head">
          <div>
            <p className="section-kicker">推荐</p>
            <h2>推荐系统占位区</h2>
          </div>
          <p className="band-note">后续可接入推荐、通知和批量操作。</p>
        </div>
        <div className="preview-panel">
          <div>
            <h3>推荐系统未启用</h3>
            <p>先把健康、任务和扫描闭环跑顺，再把推荐位接进来。</p>
          </div>
          <button type="button" className="ghost-button" onClick={onOpenHelp}>
            看帮助
          </button>
        </div>
      </section>
    </>
  );
}
