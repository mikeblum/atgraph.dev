# atproto-graph-viz

## Apache Age

```
CREATE EXTENSION IF NOT EXISTS age;
LOAD 'age';
SET search_path = ag_catalog, "atproto_graph_viz";

CREATE INDEX idx_gin_user
ON atproto_graph_viz."User" USING gin (properties);

SELECT * FROM cypher('atproto_graph_viz', $$
    EXPLAIN MATCH (u:User {did: 1})
    RETURN u
$$) as (n agtype);
```
