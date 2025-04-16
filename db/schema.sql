CREATE TABLE tick (
    id uuid DEFAULT gen_random_uuid(),
    symbol VARCHAR(255),
    price DOUBLE PRECISION,
    quantity DOUBLE PRECISION,
    timestamp TIMESTAMP,
    PRIMARY KEY (symbol, timestamp)
);

CREATE INDEX idx_tick_symbol_timestamp_desc ON tick(symbol, timestamp DESC);
CREATE INDEX idx_tick_timestamp ON tick(timestamp);
CREATE INDEX idx_tick_price ON tick(price);
CREATE INDEX idx_tick_quantity ON tick(quantity);

CREATE TABLE ohlc (
    id uuid DEFAULT gen_random_uuid(),
    symbol VARCHAR(255),
    open DOUBLE PRECISION,
    high DOUBLE PRECISION,
    low DOUBLE PRECISION,
    close DOUBLE PRECISION,
    volume DOUBLE PRECISION,
    open_time TIMESTAMP,
    close_time TIMESTAMP,
    PRIMARY KEY (symbol, open_time)
);

CREATE INDEX idx_ohlc_symbol_open_time_desc ON ohlc(symbol, open_time DESC);
CREATE INDEX idx_ohlc_close_time ON ohlc(close_time);
CREATE INDEX idx_ohlc_volume ON ohlc(volume);