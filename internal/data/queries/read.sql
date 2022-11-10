SELECT id, created_at, title, year, runtime, genres, version
FROM movies
WHERE id = $1