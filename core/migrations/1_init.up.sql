CREATE TABLE IF NOT EXISTS projects
(
    id         SERIAL PRIMARY KEY,
    name       VARCHAR(255) NOT NULL,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS goods
(
    id          SERIAL,
    project_id  INT          NOT NULL REFERENCES projects (id),
    name        VARCHAR(255) NOT NULL,
    description TEXT      DEFAULT 'NO DESC',
    priority    INT          NOT NULL,
    removed     BOOLEAN   DEFAULT FALSE,
    created_at  TIMESTAMP DEFAULT NOW(),

    PRIMARY KEY (id, project_id)
);

CREATE INDEX idx_goods_project_id ON goods (project_id);
CREATE INDEX idx_goods_name ON goods (name);

INSERT INTO projects (name)
VALUES ('First record');
