-- +goose Up
CREATE TABLE tag_implications (
    antecedent text NOT NULL,
    consequent text NOT NULL,
    created_by uuid REFERENCES users (id),
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (antecedent, consequent),
    CHECK (antecedent <> consequent)
);

-- +goose Down
DROP TABLE tag_implications;
