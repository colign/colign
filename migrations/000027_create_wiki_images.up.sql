CREATE TABLE wiki_images (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    project_id BIGINT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    page_id UUID NOT NULL REFERENCES wiki_pages(id) ON DELETE CASCADE,
    filename TEXT NOT NULL,
    content_type TEXT NOT NULL,
    data BYTEA NOT NULL,
    size INT NOT NULL CHECK (size > 0 AND size <= 5242880),
    created_by BIGINT NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_wiki_images_page_id ON wiki_images(page_id);
CREATE INDEX idx_wiki_images_project_id ON wiki_images(project_id);
CREATE INDEX idx_wiki_images_created_by ON wiki_images(created_by);
