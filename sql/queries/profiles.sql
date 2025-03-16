-- name: InsertProfile :exec
INSERT INTO atgraph.profiles(did, type, handle, created, ingested, rev, sig, version) VALUES ($1, $2, $3, $4, $5, $6, $7, $8);

-- name: SelectProfiles :many
SELECT * FROM atgraph.profiles;
