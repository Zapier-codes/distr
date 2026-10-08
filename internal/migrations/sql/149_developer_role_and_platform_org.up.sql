-- g.iii-a (D6, D7, D8): a narrow `developer` role, and the one platform organization.
--
-- A developer is a distr UserAccount that sells on the marketplace. The role is the lowest one, below read_only,
-- so every existing vendor-portal route (which requires read_write or admin, or a specific higher role) refuses it
-- without touching those routes. It exists so a developer in the one platform organization cannot reach the
-- platform's own vendor data (D6 point 2).
ALTER TYPE USER_ROLE ADD VALUE IF NOT EXISTS 'developer';

-- The platform itself is one organization, found by this fixed slug, so developer sign-up can add a user to it
-- without creating a new organization (D6 point 1). Inserted idempotently: an instance that already has it keeps it.
INSERT INTO Organization (name, slug, features)
VALUES ('Distr Marketplace', 'distr-marketplace', ARRAY[]::FEATURE[])
ON CONFLICT (slug) DO NOTHING;
