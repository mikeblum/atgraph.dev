SELECT create_graph('atproto_graph_viz');

CREATE INDEX idx_gin_user
ON atproto_graph_viz."User" USING gin (properties);
