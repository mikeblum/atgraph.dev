-- app.bsky.actor.profile
CREATE TABLE IF NOT EXISTS atgraph.profiles
(
    did         String NOT NULL                              COMMENT 'did: (string, required): the account DID associated with the repo, in strictly normalized form (eg, lowercase as appropriate)',
    lexicon     String NOT NULL                              COMMENT 'lexicon: atproto lexicon type ex. app.bsky.actor.profile',
    handle      String                                       COMMENT 'handle: atproto handle (changes when using custom domains)',
    -- 9 = nanosecond precision
    created     DateTime64(9, 'UTC') NOT NULL                COMMENT 'created: app.bsky.actor.profile created timestamp',
    ingested    DateTime64(9, 'UTC') NOT NULL DEFAULT now()  COMMENT 'ingested: timestamp for tracking ingestion lag time',
    updated     DateTime64(9, 'UTC') DEFAULT NULL            COMMENT 'updated: timestamp for tracking ingestion lag time',
    rev         String                                       COMMENT '(string, TID format, required): revision of the repo, used as a logical clock. Must increase monotonically. Recommend using current timestamp as TID; rev values in the "future" (beyond a fudge factor) should be ignored and not processed.',
    sig         String                                       COMMENT 'sig: (byte array, required): cryptographic signature of this commit, as raw bytes',
    version     UInt8                                        COMMENT 'version: (integer, required): fixed value of 3 for this repo format version',
    description String DEFAULT NULL                          COMMENT 'description: (string, optional): short overview of the Lexicon, usually one or two sentences'
)
ENGINE = ReplacingMergeTree(created)
PRIMARY KEY(did)
ORDER BY did;
