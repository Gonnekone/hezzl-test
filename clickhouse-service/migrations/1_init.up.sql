CREATE TABLE IF NOT EXISTS hezzl.goods
(
    Id          UInt64,
    ProjectId   UInt32,
    Name        String,
    Description String,
    Priority    UInt32,
    Removed     UInt8,
    EventTime   DateTime('UTC') DEFAULT now()
) ENGINE = MergeTree()
      PARTITION BY toYYYYMM(EventTime)
      ORDER BY (ProjectId, EventTime, Id)
      TTL EventTime + INTERVAL 1 YEAR DELETE;

ALTER TABLE hezzl.goods
    ADD INDEX idx_name Name TYPE bloom_filter(0.01) GRANULARITY 1;

ALTER TABLE hezzl.goods
    ADD INDEX idx_name_ngram Name TYPE ngrambf_v1(3, 256, 2, 0) GRANULARITY 1;

ALTER TABLE hezzl.goods
    ADD INDEX idx_project_id ProjectId TYPE set(100) GRANULARITY 1;

ALTER TABLE hezzl.goods
    ADD INDEX idx_id Id TYPE minmax GRANULARITY 8;