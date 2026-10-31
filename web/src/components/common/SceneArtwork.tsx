import { faFilm } from "@fortawesome/free-solid-svg-icons/faFilm";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { useState } from "react";

interface SceneArtworkProps {
  src?: string | null;
  alt: string;
  loading?: "lazy" | "eager";
}

/** Shared scene poster geometry and fallback for catalog cards and media views. */
export function SceneArtwork({ src, alt, loading = "lazy" }: SceneArtworkProps) {
  const [failedSrc, setFailedSrc] = useState<string | null>(null);
  return src && src !== failedSrc ? (
    <img src={src} alt={alt} loading={loading} className="performer-scene-card__image" onError={() => setFailedSrc(src)} />
  ) : (
    <div className="performer-scene-card__image-placeholder" aria-hidden="true">
      <FontAwesomeIcon icon={faFilm} />
    </div>
  );
}
