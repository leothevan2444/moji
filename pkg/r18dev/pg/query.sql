-- name: GetActressID :one
SELECT id
FROM public.derived_actress
WHERE name_romaji=sqlc.arg(name) OR name_kanji=sqlc.arg(name) OR name_kana=sqlc.arg(name)
LIMIT 1;

-- name: GetActressVideos :many
SELECT
  sqlc.embed(dv)
FROM derived_video_actress dva
JOIN derived_video dv
  ON dv.content_id = dva.content_id
WHERE dva.actress_id = sqlc.arg(actress_id)
ORDER BY
  COALESCE(dva.release_date, dv.release_date) DESC,
  dva.ordinality ASC;

-- name: GetVideo :one
SELECT *
FROM public.derived_video
WHERE content_id = sqlc.arg(content_id)
LIMIT 1;