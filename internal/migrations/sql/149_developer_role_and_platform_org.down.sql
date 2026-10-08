-- Postgres cannot drop a value from an enum, so 'developer' stays in USER_ROLE after this migration (the same
-- convention as FEATURE values, e.g. 129_advisories.down.sql). The role is only ever assigned by the developer
-- join path; removing that code is the real revert. The platform organization is left in place too: it may hold
-- live listings, and deleting it would cascade those away.
SELECT 1;
