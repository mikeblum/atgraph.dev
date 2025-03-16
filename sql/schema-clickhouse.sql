-- Clickhouse database schema (incompat with sqlc)
CREATE DATABASE IF NOT EXISTS atgraph ENGINE = postgresql COMMENT 'atgraph.dev ClickHouse database';

CREATE TABLE IF NOT EXISTS atgraph.profile
(
    did         String NOT NULL             COMMENT 'did: (string, required): the account DID associated with the repo, in strictly normalized form (eg, lowercase as appropriate)',
    type        String NOT NULL             COMMENT 'type: atproto lexicon type ex. app.bsky.actor.profile'
    handle      String                      COMMENT 'handle: atproto handle (changes when using custom domains)'
    created     DateTime('UTC') NOT NULL    COMMENT 'created: app.bsky.actor.profile created timestamp'
    ingested    DateTime('UTC') NOT NULL    COMMENT 'ingested: timestamp for tracking ingestion lag time'
    updated     DateTime('UTC')             COMMENT 'updated: timestamp for tracking ingestion lag time'
    rev         String                      COMMENT '(string, TID format, required): revision of the repo, used as a logical clock. Must increase monotonically. Recommend using current timestamp as TID; rev values in the "future" (beyond a fudge factor) should be ignored and not processed.'
    sig         String                      COMMENT 'sig: (byte array, required): cryptographic signature of this commit, as raw bytes'
    version     UInt8                       COMMENT 'version: (integer, required): fixed value of 3 for this repo format version'
)
ENGINE = MergeTree()
PRIMARY KEY(did)
ORDER BY did;
