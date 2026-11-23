import { useEffect,useRef,useState } from "react";
import { useQuery } from "urql";
import { useTranslation } from "react-i18next";
import { TorrentPiecesDocumentDocument,type TorrentPiecesDocumentQuery } from "../../graphql/generated/graphql";

type PieceMap=TorrentPiecesDocumentQuery["torrentPieces"];
export function TorrentPieces({hash,observedAt,enabled}:{hash:string;observedAt:string;enabled:boolean}) {
  const {t}=useTranslation();
  const [result,reload]=useQuery({query:TorrentPiecesDocumentDocument,variables:{hash},pause:!enabled,requestPolicy:"network-only"});
  const [confirmed,setConfirmed]=useState<PieceMap|null>(null);
  const previous=useRef(observedAt);
  useEffect(()=>{if(enabled && previous.current!==observedAt) reload({requestPolicy:"network-only"});previous.current=observedAt;},[observedAt,enabled,reload]);
  useEffect(()=>{if(result.data?.torrentPieces && !result.error && !result.fetching) setConfirmed(result.data.torrentPieces);},[result.data,result.error,result.fetching]);
  const candidate=result.error?confirmed:result.data?.torrentPieces??confirmed;
  const pieces=candidate?.hash===hash?candidate:null;
  const totals=pieces?.buckets.reduce((sum,b)=>({missing:sum.missing+b.missing,downloading:sum.downloading+b.downloading,downloaded:sum.downloaded+b.downloaded}),{missing:0,downloading:0,downloaded:0});
  return <section className="torrent-pieces" aria-label={t("torrentPieces.title")}>
    {pieces && pieces.total>0 && totals ? <>
      <svg className="torrent-pieces__bar" viewBox={`0 0 ${pieces.total} 1`} preserveAspectRatio="none" role="img" aria-label={t("torrentPieces.summary",{...totals,total:pieces.total})} shapeRendering="crispEdges">
        {pieces.buckets.map(b=><g key={b.start}><title>{t("torrentPieces.range",{start:b.start+1,end:b.start+b.count,missing:b.missing,downloading:b.downloading,downloaded:b.downloaded})}</title>
          <rect x={b.start} y={0} width={b.count} height={1} className="torrent-pieces__missing"/>
          {b.downloaded>0?<rect x={b.start} y={0} width={b.count} height={b.downloaded/b.count} className="torrent-pieces__downloaded"/>:null}
          {b.downloading>0?<rect x={b.start} y={b.downloaded/b.count} width={b.count} height={b.downloading/b.count} className="torrent-pieces__downloading"/>:null}
        </g>)}
      </svg>
      <div className="torrent-pieces__legend">{(["missing","downloading","downloaded"] as const).map(state=><span key={state}><i className={`torrent-pieces__${state}`} aria-hidden="true"/>{t(`torrentPieces.${state}`)} {totals[state]}</span>)}</div>
    </>:null}
    {result.error?<p role="status">{t(pieces?"torrentPieces.stale":"torrentPieces.failed")}</p>:!enabled&&pieces?<p role="status">{t("torrentPieces.stale")}</p>:!pieces?<p role="status">{t(enabled?"torrentPieces.loading":"torrentPieces.unavailable")}</p>:pieces.total===0?<p role="status">{t("torrentPieces.empty")}</p>:null}
  </section>;
}
