-- Why a build ended up failed, in words the operator can act on. Never holds a credential.
ALTER TABLE TenantConfig ADD COLUMN build_status_message TEXT;
