-- PostgreSQL-compatible schema for sqlc code generation
CREATE SCHEMA IF NOT EXISTS atgraph;

CREATE TABLE IF NOT EXISTS atgraph.profiles
(
    did         String NOT NULL PRIMARY KEY, -- did: (string, required): the account DID associated with the repo, in strictly normalized form (eg, lowercase as appropriate)
    lexicon     String NOT NULL,             -- lexicon: atproto lexicon type ex. app.bsky.actor.profile
    handle      String,                      -- handle: atproto handle (changes when using custom domains)
    created     DateTime('UTC') NOT NULL,    -- created: app.bsky.actor.profile created timestamp
    ingested    DateTime('UTC') NOT NULL,    -- ingested: timestamp for tracking ingestion lag time
    updated     DateTime('UTC'),             -- updated: timestamp for tracking ingestion lag time
    rev         String,                      -- (string, TID format, required): revision of the repo, used as a logical clock. Must increase monotonically. Recommend using current timestamp as TID; rev values in the "future" (beyond a fudge factor) should be ignored and not processed.
    sig         String,                      -- sig: (byte array, required): cryptographic signature of this commit, as raw bytes
    version     UInt8                        -- version: (integer, required): fixed value of 3 for this repo format version
);

CREATE INDEX IF NOT EXISTS idx_profiles_handle ON atgraph.profiles(handle);
CREATE INDEX IF NOT EXISTS idx_profiles_lexicon ON atgraph.profiles(lexicon);
CREATE INDEX IF NOT EXISTS idx_profiles_created ON atgraph.profiles(created);
CREATE INDEX IF NOT EXISTS idx_profiles_updated ON atgraph.profiles(updated);
