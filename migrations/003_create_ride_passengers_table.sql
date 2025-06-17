CREATE TABLE IF NOT EXISTS ride_passengers (
    id SERIAL PRIMARY KEY,
    ride_id INTEGER NOT NULL REFERENCES rides(id) ON DELETE CASCADE,
    passenger_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    joined_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    status VARCHAR(20) NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'cancelled')),
    UNIQUE(ride_id, passenger_id)
);

CREATE INDEX IF NOT EXISTS idx_ride_passengers_ride_id ON ride_passengers(ride_id);
CREATE INDEX IF NOT EXISTS idx_ride_passengers_passenger_id ON ride_passengers(passenger_id);
CREATE INDEX IF NOT EXISTS idx_ride_passengers_status ON ride_passengers(status);