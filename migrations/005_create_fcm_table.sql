CREATE TABLE user_fcm_tokens (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    fcm_token VARCHAR(255) NOT NULL,
    device_id VARCHAR(255),
    platform VARCHAR(20) CHECK (platform IN ('android', 'ios', 'web')),
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(user_id, device_id)
);

CREATE INDEX idx_user_fcm_tokens_user_id ON user_fcm_tokens(user_id);
CREATE INDEX idx_user_fcm_tokens_active ON user_fcm_tokens(is_active);