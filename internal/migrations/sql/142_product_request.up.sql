CREATE TABLE ProductRequest (
  id            UUID      PRIMARY KEY DEFAULT gen_random_uuid(),
  created_at    TIMESTAMP NOT NULL DEFAULT now(),
  contact_email TEXT      NOT NULL,
  app_name      TEXT      NOT NULL,
  theme_color   TEXT      NOT NULL
);

CREATE INDEX ProductRequest_contact_email ON ProductRequest(contact_email);
