-- name: CreateUser
INSERT INTO users (
  username,
  email,
  password_hash,
  keycloak_id,
  keycloak_access_token,
  keycloak_refresh_token,
  keycloak_access_expires_at,
  keycloak_refresh_expires_at,
  is_active,
  roles,
  metadata
) VALUES (
  :username,
  :email,
  :password_hash,
  :keycloak_id,
  :keycloak_access_token,
  :keycloak_refresh_token,
  :keycloak_access_expires_at,
  :keycloak_refresh_expires_at,
  :is_active,
  :roles,
  :metadata
)
RETURNING *;

-- name: GetUserByID
SELECT * FROM users WHERE id = $1;

-- name: GetUserByEmail
SELECT * FROM users WHERE email = $1;

-- name: GetUserByKeycloakID
SELECT * FROM users WHERE keycloak_id = $1;

-- name: UpdateUser
UPDATE users SET
  username = COALESCE(:username, username),
  email = COALESCE(:email, email),
  password_hash = COALESCE(:password_hash, password_hash),
  keycloak_id = COALESCE(:keycloak_id, keycloak_id),
  keycloak_access_token = COALESCE(:keycloak_access_token, keycloak_access_token),
  keycloak_refresh_token = COALESCE(:keycloak_refresh_token, keycloak_refresh_token),
  keycloak_access_expires_at = COALESCE(:keycloak_access_expires_at, keycloak_access_expires_at),
  keycloak_refresh_expires_at = COALESCE(:keycloak_refresh_expires_at, keycloak_refresh_expires_at),
  is_active = COALESCE(:is_active, is_active),
  roles = COALESCE(:roles, roles),
  metadata = COALESCE(:metadata, metadata),
  updated_at = now()
WHERE id = :id
RETURNING *;

-- name: DeleteUser
DELETE FROM users WHERE id = $1;
