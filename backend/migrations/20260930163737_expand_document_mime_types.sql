-- +goose Up

ALTER TABLE documents
DROP CONSTRAINT documents_mime_type_check;

ALTER TABLE documents
ADD CONSTRAINT documents_mime_type_check
CHECK (
    mime_type IN (
        'application/pdf',
        'application/vnd.openxmlformats-officedocument.wordprocessingml.document',
        'image/jpeg',
        'image/png'
    )
);

-- +goose Down

ALTER TABLE documents
DROP CONSTRAINT documents_mime_type_check;

ALTER TABLE documents
ADD CONSTRAINT documents_mime_type_check
CHECK (
    mime_type IN (
        'application/pdf',
        'application/vnd.openxmlformats-officedocument.wordprocessingml.document'
    )
);
