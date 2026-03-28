CREATE TABLE IF NOT EXISTS transaction_traces (
    chain_id BIGINT NOT NULL,
    tx_hash VARCHAR NOT NULL,
    trace_json JSONB NOT NULL,
    created_at INTEGER NOT NULL DEFAULT CAST(EXTRACT(EPOCH FROM NOW()) AS BIGINT),
    updated_at INTEGER NOT NULL DEFAULT CAST(EXTRACT(EPOCH FROM NOW()) AS BIGINT),
    PRIMARY KEY (chain_id, tx_hash)
);

CREATE INDEX IF NOT EXISTS transaction_traces_chain_idx
    ON transaction_traces (chain_id);
