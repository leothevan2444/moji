import { Link } from "react-router";

export function NotFoundPage() {
  return (
    <main className="content">
      <section className="empty-card empty-card--wide">
        <h2 tabIndex={-1}>页面不存在</h2>
        <p>这个地址没有对应的 Moji 页面。</p>
        <Link className="ghost-button" to="/">返回主页</Link>
      </section>
    </main>
  );
}
