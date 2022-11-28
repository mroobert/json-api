SELECT id, created_at, name, email, password_hash, activated, version
FROM users
WHERE email = $1