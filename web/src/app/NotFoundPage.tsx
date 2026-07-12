import { Link } from "react-router";
import { useTranslation } from "react-i18next";

export function NotFoundPage() {
  const { t } = useTranslation();
  return (
    <main className="content">
      <section className="empty-card empty-card--wide">
        <h2 tabIndex={-1}>{t("errors.notFoundTitle")}</h2>
        <p>{t("errors.notFoundDetail")}</p>
        <Link className="ghost-button" to="/">{t("errors.returnHome")}</Link>
      </section>
    </main>
  );
}
