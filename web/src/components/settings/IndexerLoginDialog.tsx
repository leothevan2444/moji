import { useEffect, useRef, useState, type FormEvent } from "react";
import { createPortal } from "react-dom";
import { useTranslation } from "react-i18next";
import type { IndexerLoginInput } from "../../graphql/generated/graphql";
import { useIndexerLogin } from "../../hooks/useIndexerLogin";
import { describeQueryError } from "../../services/queryError";
import { Drawer } from "../layout/Drawer";

export function IndexerLoginDialog({ input, name, onClose }: { input: IndexerLoginInput; name: string; onClose: () => void }) {
  const { t, i18n } = useTranslation();
  const login = useIndexerLogin(input);
  const [answer, setAnswer] = useState("");
  const [expired, setExpired] = useState(false);
  const [imageFailed,setImageFailed]=useState(false);
  const answerField=useRef<HTMLInputElement>(null);
  const report = login.report;
  const waiting = report?.state === "WAITING";
  const error = login.error as { message?: string; graphQLErrors?: { message?: string; extensions?: { code?: string } }[] } | undefined;
  const expiredError = [error?.message, ...(error?.graphQLErrors?.flatMap(item => [item.message, item.extensions?.code]) ?? [])].some(value => value?.includes("INDEXER_RESOURCE_EXPIRED"));
  useEffect(() => {
    setAnswer(""); setExpired(false); setImageFailed(false);
    if (report?.state !== "WAITING") return;
    const remaining = Date.parse(report.expiresAt) - Date.now();
    if (!Number.isFinite(remaining) || remaining <= 0) { setExpired(true); return; }
    answerField.current?.focus();
    const timer = setTimeout(() => setExpired(true), remaining);
    return () => clearTimeout(timer);
  }, [report]);
  const submit = (event: FormEvent) => {
    event.preventDefault();
    if (login.busy || expired || imageFailed || !answer.trim()) return;
    const value = answer;
    setAnswer("");
    void login.answer(value);
  };
  return createPortal(<Drawer visibleDrawer="indexer-login" closing={false} title={t("indexerLogin.title")} onClose={onClose} footer={
    <div className="indexer-actions">
      {(!waiting || expired || imageFailed) && report?.state !== "READY" && report?.state !== "NOT_REQUIRED" ? <button type="button" className="primary-button" disabled={login.busy} onClick={() => void login.begin()}>{t(expired || imageFailed ? "indexerLogin.restart" : "indexerLogin.begin")}</button> : null}
      {waiting && !expired && !imageFailed ? <button type="submit" form="indexer-login-answer" className="primary-button" disabled={login.busy || imageFailed || !answer.trim()}>{t("indexerLogin.submit")}</button> : null}
      {waiting ? <button type="button" disabled={login.cancelling} onClick={() => void login.cancel()}>{t("indexerLogin.cancel")}</button> : null}
      <button type="button" disabled={login.busy} onClick={() => void login.refresh()}>{t("indexerLogin.refresh")}</button>
    </div>
  }>
    <p className="indexer-login__site">{name}</p><p className="settings-note">{t("indexerLogin.scope")}</p>
    {login.busy ? <p role="status">{t("indexerLogin.working")}</p> : null}
    {login.error ? <p role="alert" className="settings-feedback tone-danger">{expiredError ? t("indexerLogin.expired") : describeQueryError(login.error)}</p> : null}
    {login.cancelled ? <p role="status">{t("indexerLogin.cancelled")}</p> : null}
    {report?.state === "READY" ? <p role="status">{t("indexerLogin.ready")}</p> : null}
    {report?.state === "NOT_REQUIRED" ? <p role="status">{t("indexerLogin.notRequired")}</p> : null}
    {waiting ? expired ? <p role="status">{t("indexerLogin.expired")}</p> : <form id="indexer-login-answer" className="settings-form indexer-login" onSubmit={submit}>
      <p>{t("indexerLogin.waiting")}</p>
      {report.challenge?.prompt ? <p className="indexer-login__prompt">{report.challenge.prompt}</p> : null}
      {report.challenge?.imageData ? <img className="indexer-login__image" src={report.challenge.imageData} alt={t("indexerLogin.image")} onError={()=>setImageFailed(true)} referrerPolicy="no-referrer" /> : null}
      {imageFailed?<p role="alert">{t("indexerLogin.imageFailed")}</p>:null}
      <label className="settings-field"><span>{t("indexerLogin.answer")}</span><input ref={answerField} value={answer} onChange={event => setAnswer(event.target.value)} maxLength={1024} autoComplete="off" spellCheck={false} disabled={login.busy || imageFailed} /></label>
      <p className="settings-note">{t("indexerLogin.expires", { time: new Date(report.expiresAt).toLocaleTimeString(i18n.resolvedLanguage) })}</p>
    </form> : null}
  </Drawer>, document.body);
}
