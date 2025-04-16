CREATE TABLE ticks (
    id uuid DEFAULT gen_random_uuid(),
    symbol VARCHAR(255),
    price DOUBLE PRECISION,
    quantity DOUBLE PRECISION,
    timestamp TIMESTAMP,
    PRIMARY KEY (id)
);

CREATE INDEX idx_tick_symbol_timestamp_desc ON ticks(symbol, timestamp DESC);
CREATE INDEX idx_tick_timestamp ON ticks(timestamp);
CREATE INDEX idx_tick_price ON ticks(price);
CREATE INDEX idx_tick_quantity ON ticks(quantity);

CREATE TABLE ohlcs (
    id uuid DEFAULT gen_random_uuid(),
    symbol VARCHAR(255),
    open DOUBLE PRECISION,
    high DOUBLE PRECISION,
    low DOUBLE PRECISION,
    close DOUBLE PRECISION,
    volume DOUBLE PRECISION,
    open_time TIMESTAMP,
    close_time TIMESTAMP,
    PRIMARY KEY (id)
);

CREATE INDEX idx_ohlc_symbol_open_time_desc ON ohlcs(symbol, open_time DESC);
CREATE INDEX idx_ohlc_close_time ON ohlcs(close_time);
CREATE INDEX idx_ohlc_volume ON ohlcs(volume);