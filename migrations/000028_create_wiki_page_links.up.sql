CREATE TABLE wiki_page_links (
    id         BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    source_page_id UUID     NOT NULL REFERENCES wiki_pages(id) ON DELETE CASCADE,
    target_page_id UUID     NOT NULL REFERENCES wiki_pages(id) ON DELETE CASCADE,
    project_id     BIGINT   NOT NULL REFERENCES projects(id)   ON DELETE CASCADE,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(source_page_id, target_page_id)
);

CREATE INDEX idx_wiki_page_links_source  ON wiki_page_links(source_page_id);
CREATE INDEX idx_wiki_page_links_target  ON wiki_page_links(target_page_id);
CREATE INDEX idx_wiki_page_links_project ON wiki_page_links(project_id);
