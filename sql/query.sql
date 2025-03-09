-- name: ExplainProfile :one
SELECT * FROM cypher('atproto_graph_viz', $$
    EXPLAIN MATCH (p:Profile {did: $1})
    RETURN u
$$) as (n agtype);

-- name: InsertProfile :one
SELECT * 
FROM cypher('atproto_graph_viz', $$
    MERGE (p:Profile {id: $1})
        ON CREATE
            SET
                p.type		= 'app.bsky.actor.profile',
                -- tracking ingestion lag time
                p.ingested 	= timestamp(),
                p.created 	= $6
        ON MATCH
            SET 
                p.handle	= $2,
                p.rev		= $3,
                p.sig 		= $4,
                p.version 	= $5,
                -- tracking firehose lag time
                p.updated 	= timestamp()
        RETURN p.id AS did, p.ingested AS ingested_ts
$$, $1) as (v agtype);