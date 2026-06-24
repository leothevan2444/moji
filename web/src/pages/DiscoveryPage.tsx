import { FormEvent } from "react";

interface DiscoveryPageProps {
  query: string;
  searchingPrimary: boolean;
  searchingFallback: boolean;
  onQueryChange: (value: string) => void;
  onSubmitPrimary: (event: FormEvent<HTMLFormElement>) => void;
  onSubmitFallback: () => void;
  onOpenHelp: () => void;
}

export function DiscoveryPage({
  query,
  searchingPrimary,
  searchingFallback,
  onQueryChange,
  onSubmitPrimary,
  onSubmitFallback,
  onOpenHelp
}: DiscoveryPageProps) {
  return (
    <>
      <section className="section-band">
        <div className="band-head">
          <div>
            <p className="section-kicker">发现</p>
            <h2>搜索</h2>
          </div>
          <p className="band-note">先查首选 StashBox 元数据，再送入 Moji 任务闭环。</p>
        </div>

        <form className="discovery-bar" onSubmit={onSubmitPrimary}>
          <input
            value={query}
            onChange={(event) => onQueryChange(event.target.value)}
            placeholder="输入番号、标题、演员或关键词"
          />
          <button type="submit" disabled={searchingPrimary || query.trim() === ""}>
            {searchingPrimary ? "搜索中" : "搜索"}
          </button>
          <button
            type="button"
            className="ghost-button"
            onClick={onSubmitFallback}
            disabled={searchingFallback || query.trim() === ""}
          >
            {searchingFallback ? "搜索中" : "备用 Jackett 搜索"}
          </button>
        </form>
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
