import { useTranslation } from "react-i18next";

export function IndexerPrivacyBadge({ privacy }: { privacy: string }) {
  const { t } = useTranslation();
  if (!privacy) return null;
  const kind = ["public", "private", "semi-private"].includes(privacy) ? privacy : "unknown";
  return <span className={`indexer-privacy indexer-privacy--${kind}`}>{t(`indexers.privacyKinds.${privacy}`, { defaultValue: privacy })}</span>;
}
