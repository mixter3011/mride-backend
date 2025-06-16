CREATE TABLE IF NOT EXISTS rides (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    car_number VARCHAR(20) NOT NULL,
    car_model VARCHAR(50) NOT NULL,
    passenger_count INTEGER NOT NULL CHECK (passenger_count >= 1 AND passenger_count <= 8),
    price DECIMAL(10,2) CHECK (price >= 0),
    from_location VARCHAR(255) NOT NULL,
    to_location VARCHAR(255) NOT NULL,
    from_latitude DECIMAL(10,8) NOT NULL,
    from_longitude DECIMAL(11,8) NOT NULL,
    to_latitude DECIMAL(10,8) NOT NULL,
    to_longitude DECIMAL(11,8) NOT NULL,
    departure_time TIMESTAMP NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'completed', 'cancelled')),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_rides_user_id ON rides(user_id);
CREATE INDEX idx_rides_departure_time ON rides(departure_time);
CREATE INDEX idx_rides_status ON rides(status);
CREATE INDEX idx_rides_location ON rides(from_latitude, from_longitude, to_latitude, to_longitude);