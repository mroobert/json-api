INSERT INTO users (name, email, password_hash, activated) 
VALUES ($1, $2, $3, $4)
RETURNING id, created_at, version