CREATE TABLE IF NOT EXISTS protocol_addresses (
    chain_id BIGINT NOT NULL,
    protocol VARCHAR(32) NOT NULL,
    address_role VARCHAR(32) NOT NULL,
    contract_address VARCHAR NOT NULL,
    is_official BOOLEAN NOT NULL DEFAULT TRUE,
    source TEXT,
    created_at INTEGER NOT NULL DEFAULT CAST(EXTRACT(EPOCH FROM NOW()) AS BIGINT),
    updated_at INTEGER NOT NULL DEFAULT CAST(EXTRACT(EPOCH FROM NOW()) AS BIGINT),
    PRIMARY KEY (chain_id, protocol, address_role, contract_address),
    CONSTRAINT protocol_addresses_protocol_chk
        CHECK (protocol IN ('aave_v3', 'balancer_v2', 'uniswap_v2')),
    CONSTRAINT protocol_addresses_role_chk
        CHECK (address_role IN ('pool', 'vault', 'factory', 'pair'))
);

CREATE INDEX IF NOT EXISTS protocol_addresses_chain_protocol_idx
    ON protocol_addresses (chain_id, protocol);

CREATE INDEX IF NOT EXISTS protocol_addresses_address_idx
    ON protocol_addresses (contract_address);

CREATE TABLE IF NOT EXISTS uniswap_v2_pairs (
    chain_id BIGINT NOT NULL,
    factory_address VARCHAR NOT NULL,
    pair_address VARCHAR NOT NULL,
    token0 VARCHAR NOT NULL,
    token1 VARCHAR NOT NULL,
    created_block UINT256 NOT NULL,
    is_official BOOLEAN NOT NULL DEFAULT TRUE,
    created_at INTEGER NOT NULL DEFAULT CAST(EXTRACT(EPOCH FROM NOW()) AS BIGINT),
    updated_at INTEGER NOT NULL DEFAULT CAST(EXTRACT(EPOCH FROM NOW()) AS BIGINT),
    PRIMARY KEY (chain_id, pair_address)
);

CREATE INDEX IF NOT EXISTS uniswap_v2_pairs_factory_idx
    ON uniswap_v2_pairs (chain_id, factory_address);

CREATE INDEX IF NOT EXISTS uniswap_v2_pairs_token0_idx
    ON uniswap_v2_pairs (chain_id, token0);

CREATE INDEX IF NOT EXISTS uniswap_v2_pairs_token1_idx
    ON uniswap_v2_pairs (chain_id, token1);

CREATE TABLE IF NOT EXISTS observed_transactions (
    chain_id BIGINT NOT NULL,
    tx_hash VARCHAR NOT NULL,
    block_number UINT256 NOT NULL,
    tx_index INTEGER,
    from_address VARCHAR NOT NULL,
    to_address VARCHAR,
    status SMALLINT,
    value UINT256,
    input_data TEXT,
    method_selector VARCHAR(10),
    gas_used UINT256,
    effective_gas_price UINT256,
    created_at INTEGER NOT NULL DEFAULT CAST(EXTRACT(EPOCH FROM NOW()) AS BIGINT),
    updated_at INTEGER NOT NULL DEFAULT CAST(EXTRACT(EPOCH FROM NOW()) AS BIGINT),
    PRIMARY KEY (chain_id, tx_hash)
);

CREATE INDEX IF NOT EXISTS observed_transactions_chain_block_idx
    ON observed_transactions (chain_id, block_number);

CREATE INDEX IF NOT EXISTS observed_transactions_chain_to_idx
    ON observed_transactions (chain_id, to_address);

CREATE INDEX IF NOT EXISTS observed_transactions_chain_selector_idx
    ON observed_transactions (chain_id, method_selector);

CREATE TABLE IF NOT EXISTS protocol_interactions (
    interaction_id TEXT PRIMARY KEY DEFAULT replace(uuid_generate_v4()::text, '-', ''),
    chain_id BIGINT NOT NULL,
    tx_hash VARCHAR NOT NULL,
    block_number UINT256 NOT NULL,
    tx_index INTEGER,
    interaction_ordinal INTEGER NOT NULL,
    timestamp INTEGER NOT NULL,
    protocol VARCHAR(32) NOT NULL,
    entrypoint VARCHAR(64) NOT NULL,
    provider_address VARCHAR NOT NULL,
    factory_address VARCHAR,
    pair_address VARCHAR,
    receiver_address VARCHAR,
    callback_target VARCHAR,
    initiator VARCHAR,
    on_behalf_of VARCHAR,
    candidate_level SMALLINT NOT NULL,
    verified BOOLEAN NOT NULL DEFAULT FALSE,
    strict BOOLEAN NOT NULL DEFAULT FALSE,
    callback_seen BOOLEAN NOT NULL DEFAULT FALSE,
    settlement_seen BOOLEAN NOT NULL DEFAULT FALSE,
    repayment_seen BOOLEAN NOT NULL DEFAULT FALSE,
    contains_debt_opening BOOLEAN NOT NULL DEFAULT FALSE,
    data_non_empty BOOLEAN,
    exclusion_reason TEXT,
    verification_notes TEXT,
    raw_method_selector VARCHAR(10),
    source_confidence SMALLINT,
    created_at INTEGER NOT NULL DEFAULT CAST(EXTRACT(EPOCH FROM NOW()) AS BIGINT),
    updated_at INTEGER NOT NULL DEFAULT CAST(EXTRACT(EPOCH FROM NOW()) AS BIGINT),
    CONSTRAINT protocol_interactions_chain_tx_ordinal_uniq
        UNIQUE (chain_id, tx_hash, interaction_ordinal),
    CONSTRAINT protocol_interactions_candidate_level_chk
        CHECK (candidate_level IN (1, 2, 3)),
    CONSTRAINT protocol_interactions_protocol_chk
        CHECK (protocol IN ('aave_v3', 'balancer_v2', 'uniswap_v2')),
    CONSTRAINT protocol_interactions_confidence_chk
        CHECK (source_confidence IS NULL OR source_confidence BETWEEN 1 AND 3)
);

CREATE INDEX IF NOT EXISTS protocol_interactions_chain_block_idx
    ON protocol_interactions (chain_id, block_number);

CREATE INDEX IF NOT EXISTS protocol_interactions_chain_tx_idx
    ON protocol_interactions (chain_id, tx_hash);

CREATE INDEX IF NOT EXISTS protocol_interactions_protocol_strict_idx
    ON protocol_interactions (protocol, strict);

CREATE INDEX IF NOT EXISTS protocol_interactions_protocol_candidate_idx
    ON protocol_interactions (protocol, candidate_level);

CREATE INDEX IF NOT EXISTS protocol_interactions_provider_idx
    ON protocol_interactions (provider_address);

CREATE INDEX IF NOT EXISTS protocol_interactions_pair_idx
    ON protocol_interactions (pair_address);

CREATE TABLE IF NOT EXISTS interaction_asset_legs (
    leg_id TEXT PRIMARY KEY DEFAULT replace(uuid_generate_v4()::text, '-', ''),
    interaction_id TEXT NOT NULL REFERENCES protocol_interactions(interaction_id) ON DELETE CASCADE,
    leg_index INTEGER NOT NULL,
    asset_address VARCHAR NOT NULL,
    asset_role VARCHAR(32) NOT NULL,
    token_side VARCHAR(16),
    amount_out UINT256,
    amount_in UINT256,
    amount_borrowed UINT256,
    amount_repaid UINT256,
    premium_amount UINT256,
    fee_amount UINT256,
    interest_rate_mode SMALLINT,
    repaid_to_address VARCHAR,
    opened_debt BOOLEAN NOT NULL DEFAULT FALSE,
    strict_leg BOOLEAN NOT NULL DEFAULT FALSE,
    event_seen BOOLEAN NOT NULL DEFAULT FALSE,
    settlement_mode VARCHAR(32),
    created_at INTEGER NOT NULL DEFAULT CAST(EXTRACT(EPOCH FROM NOW()) AS BIGINT),
    updated_at INTEGER NOT NULL DEFAULT CAST(EXTRACT(EPOCH FROM NOW()) AS BIGINT),
    CONSTRAINT interaction_asset_legs_interaction_leg_uniq
        UNIQUE (interaction_id, leg_index),
    CONSTRAINT interaction_asset_legs_asset_role_chk
        CHECK (asset_role IN ('borrowed', 'repaid', 'token0_flow', 'token1_flow')),
    CONSTRAINT interaction_asset_legs_token_side_chk
        CHECK (token_side IS NULL OR token_side IN ('token0', 'token1')),
    CONSTRAINT interaction_asset_legs_settlement_mode_chk
        CHECK (
            settlement_mode IS NULL OR
            settlement_mode IN ('full_repayment', 'debt_opening', 'invariant_restored')
        )
);

CREATE INDEX IF NOT EXISTS interaction_asset_legs_interaction_idx
    ON interaction_asset_legs (interaction_id);

CREATE INDEX IF NOT EXISTS interaction_asset_legs_asset_idx
    ON interaction_asset_legs (asset_address);

CREATE INDEX IF NOT EXISTS interaction_asset_legs_interest_mode_idx
    ON interaction_asset_legs (interest_rate_mode);

CREATE INDEX IF NOT EXISTS interaction_asset_legs_opened_debt_idx
    ON interaction_asset_legs (opened_debt);

CREATE TABLE IF NOT EXISTS flashloan_transactions (
    chain_id BIGINT NOT NULL,
    tx_hash VARCHAR NOT NULL,
    block_number UINT256 NOT NULL,
    timestamp INTEGER NOT NULL,
    contains_candidate_interaction BOOLEAN NOT NULL DEFAULT FALSE,
    contains_verified_interaction BOOLEAN NOT NULL DEFAULT FALSE,
    contains_verified_strict_interaction BOOLEAN NOT NULL DEFAULT FALSE,
    interaction_count INTEGER NOT NULL DEFAULT 0,
    strict_interaction_count INTEGER NOT NULL DEFAULT 0,
    protocol_count INTEGER NOT NULL DEFAULT 0,
    protocols TEXT,
    created_at INTEGER NOT NULL DEFAULT CAST(EXTRACT(EPOCH FROM NOW()) AS BIGINT),
    updated_at INTEGER NOT NULL DEFAULT CAST(EXTRACT(EPOCH FROM NOW()) AS BIGINT),
    PRIMARY KEY (chain_id, tx_hash)
);

CREATE INDEX IF NOT EXISTS flashloan_transactions_chain_block_idx
    ON flashloan_transactions (chain_id, block_number);

CREATE INDEX IF NOT EXISTS flashloan_transactions_strict_idx
    ON flashloan_transactions (contains_verified_strict_interaction);

CREATE TABLE IF NOT EXISTS scanner_cursors (
    scanner_name VARCHAR(64) NOT NULL,
    chain_id BIGINT NOT NULL,
    cursor_type VARCHAR(32) NOT NULL,
    block_number UINT256 NOT NULL,
    updated_at INTEGER NOT NULL DEFAULT CAST(EXTRACT(EPOCH FROM NOW()) AS BIGINT),
    PRIMARY KEY (scanner_name, chain_id, cursor_type),
    CONSTRAINT scanner_cursors_type_chk
        CHECK (cursor_type IN ('raw_logs', 'tx_fetch', 'candidate_extract', 'verification'))
);

CREATE INDEX IF NOT EXISTS scanner_cursors_chain_idx
    ON scanner_cursors (chain_id, cursor_type);
