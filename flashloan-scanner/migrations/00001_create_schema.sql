DO
$$
    BEGIN
        IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'uint256') THEN
            CREATE DOMAIN UINT256 AS NUMERIC
                CHECK (VALUE >= 0 AND VALUE < POWER(CAST(2 AS NUMERIC), CAST(256 AS NUMERIC)) AND SCALE(VALUE) = 0);
        ELSE
            ALTER DOMAIN UINT256 DROP CONSTRAINT uint256_check;
            ALTER DOMAIN UINT256 ADD
                CHECK (VALUE >= 0 AND VALUE < POWER(CAST(2 AS NUMERIC), CAST(256 AS NUMERIC)) AND SCALE(VALUE) = 0);
        END IF;
    END
$$;
CREATE EXTENSION IF NOT EXISTS "uuid-ossp" cascade;
-- CREATE EXTENSION "uuid-ossp";

CREATE TABLE IF NOT EXISTS block_headers(
    guid        TEXT PRIMARY KEY DEFAULT replace(uuid_generate_v4()::text, '-', ''),
    hash        VARCHAR NOT NULL,
    parent_hash VARCHAR NOT NULL,
    number      UINT256 NOT NULL,
    rlp_bytes   VARCHAR NOT NULL,
    timestamp   INTEGER CHECK (timestamp > 0)
);
CREATE INDEX IF NOT EXISTS block_headers_timestamp ON block_headers (timestamp);
CREATE INDEX IF NOT EXISTS block_headers_number ON block_headers (number);

CREATE TABLE IF NOT EXISTS contract_events(
    guid             TEXT PRIMARY KEY DEFAULT replace(uuid_generate_v4()::text, '-', ''),
    block_hash       VARCHAR NOT NULL,
    contract_address VARCHAR NOT NULL,
    transaction_hash VARCHAR NOT NULL,
    log_index        INTEGER NOT NULL,
    block_number     UINT256 NOT NULL,
    event_signature  VARCHAR NOT NULL,
    rlp_bytes        VARCHAR NOT NULL,
    timestamp        INTEGER CHECK (timestamp > 0)
);
CREATE INDEX IF NOT EXISTS contract_events_number ON contract_events(block_number);
CREATE INDEX IF NOT EXISTS contract_events_timestamp ON contract_events(timestamp);
CREATE INDEX IF NOT EXISTS contract_events_block_hash ON contract_events(block_hash);
CREATE INDEX IF NOT EXISTS contract_events_event_signature ON contract_events(event_signature);
CREATE INDEX IF NOT EXISTS contract_events_contract_address ON contract_events(contract_address);
CREATE INDEX IF NOT EXISTS contract_events_transaction_hash ON contract_events(transaction_hash);

create table if not exists event_block(
    guid           TEXT PRIMARY KEY DEFAULT replace(uuid_generate_v4()::text, '-', ''),
    chain_id       VARCHAR,
    block_number   UINT256     default 0,
    created        INTEGER CHECK (created > 0),
    updated        INTEGER CHECK (updated > 0)
);
CREATE INDEX IF NOT EXISTS event_block_guid ON event_block(guid);

create table if not exists bridge_msg_sent(
    guid                   TEXT PRIMARY KEY DEFAULT replace(uuid_generate_v4()::text, '-', ''),
    tx_hash                VARCHAR,
    block_number           UINT256  default 0,
    source_token_address   VARCHAR,
    dest_token_address     VARCHAR,
    source_chain_id         UINT256,
    dest_chain_id           UINT256,
    from_address           VARCHAR,
    to_address             VARCHAR,
    fee                    UINT256  default 0,
    msg_nonce              UINT256  default 0,
    amount                 UINT256  default 0,
    relation               BOOLEAN  default false,
    msg_hash               VARCHAR,
    timestamp              INTEGER CHECK (timestamp > 0)
);
CREATE INDEX IF NOT EXISTS bridge_msg_sent_tx_hash ON bridge_msg_sent(tx_hash);
CREATE INDEX IF NOT EXISTS bridge_msg_sent_msg_hash ON bridge_msg_sent(msg_hash);

create table if not exists bridge_msg_hash(
    guid                   TEXT PRIMARY KEY DEFAULT replace(uuid_generate_v4()::text, '-', ''),
    tx_hash                VARCHAR,
    block_number            UINT256          default 0,
    source_chain_id         UINT256,
    dest_chain_id           UINT256,
    source_token_address   VARCHAR,
    dest_token_address     VARCHAR,
    msg_hash               VARCHAR,
    msg_nonce              UINT256  default 0,
    relation               BOOLEAN default false,
    timestamp              INTEGER CHECK (timestamp > 0)
);
CREATE INDEX IF NOT EXISTS bridge_msg_hash_tx_hash ON bridge_msg_hash (tx_hash);
CREATE INDEX IF NOT EXISTS bridge_msg_hash_msg_hash ON bridge_msg_hash (msg_hash);

create table if not exists bridge_initiate(
    guid                   TEXT PRIMARY KEY DEFAULT replace(uuid_generate_v4()::text, '-', ''),
    tx_hash                VARCHAR,
    block_number           UINT256          default 0,
    source_chain_id        UINT256,
    dest_chain_id          UINT256,
    source_token_address   VARCHAR,
    dest_token_address     VARCHAR,
    from_address           VARCHAR,
    to_address             VARCHAR,
    amount                 UINT256,
    status                 SMALLINT not null, --0: 同步到合约事件，1:目标链已发起交易；2:目标链资金解锁已完成--
    timestamp              INTEGER CHECK (timestamp > 0)
 );
CREATE INDEX IF NOT EXISTS bridge_initiate_guid ON bridge_initiate(guid);
CREATE INDEX IF NOT EXISTS bridge_initiate_tx_hash ON bridge_initiate(tx_hash);

create table if not exists bridge_finalize(
    guid                   TEXT PRIMARY KEY DEFAULT replace(uuid_generate_v4()::text, '-', ''),
    tx_hash                VARCHAR,
    block_number           UINT256          default 0,
    source_chain_id        UINT256,
    dest_chain_id          UINT256,
    source_token_address   VARCHAR,
    dest_token_address     VARCHAR,
    from_address           VARCHAR,
    to_address             VARCHAR,
    amount                 UINT256,
    status                 SMALLINT not null, --0:同步到合约事件，1:已到账--
    timestamp              INTEGER CHECK (timestamp > 0)
);
CREATE INDEX IF NOT EXISTS bridge_finalize_tx_hash ON bridge_finalize (tx_hash);

create table if not exists claim_reward(
    guid                   TEXT PRIMARY KEY DEFAULT replace(uuid_generate_v4()::text, '-', ''),
    tx_hash                VARCHAR,
    block_number           UINT256          default 0,
    claim_address          VARCHAR,
    start_pool_id          UINT256,
    end_pool_id            UINT256,
    token_address          VARCHAR,
    reward_amount          UINT256,
    timestamp              INTEGER CHECK (timestamp > 0)
);
CREATE INDEX IF NOT EXISTS claim_reward_guid ON claim_reward(guid);
CREATE INDEX IF NOT EXISTS claim_reward_tx_hash ON claim_reward(tx_hash);

create table if not exists lp_withdraw(
    guid            TEXT PRIMARY KEY DEFAULT replace(uuid_generate_v4()::text, '-', ''),
    tx_hash         VARCHAR,
    block_number    UINT256          default 0,
    withdrawer      VARCHAR,
    start_pool_id   UINT256,
    end_pool_id     UINT256,
    token_address   VARCHAR,
    amount          UINT256,
    reward_amount   UINT256,
    timestamp       INTEGER CHECK (timestamp > 0)
);
CREATE INDEX IF NOT EXISTS claim_reward_guid ON claim_reward(guid);
CREATE INDEX IF NOT EXISTS claim_reward_tx_hash ON claim_reward(tx_hash);


CREATE TABLE IF NOT EXISTS staking_record(
    guid          TEXT  PRIMARY KEY DEFAULT replace(uuid_generate_v4()::text, '-', ''),
    tx_hash       VARCHAR  NOT NULL,
    block_number  UINT256  NOT NULL,
    user_address  VARCHAR  not null,
    token_address   VARCHAR,
    amount        UINT256  not null,
    reward        UINT256,
    start_pool_id UINT256,
    end_pool_id   UINT256,
    status        SMALLINT not null,
    timestamp     INTEGER  NOT NULL CHECK (timestamp > 0)
);
CREATE INDEX IF NOT EXISTS staking_record_tx_hash ON staking_record (tx_hash);
CREATE INDEX IF NOT EXISTS staking_record_block_number ON staking_record (block_number);
CREATE INDEX IF NOT EXISTS staking_record_user_address ON staking_record (user_address);
CREATE INDEX IF NOT EXISTS staking_record_token_address ON staking_record (token_address);
CREATE INDEX IF NOT EXISTS staking_record_status ON staking_record (status);
CREATE INDEX IF NOT EXISTS staking_record_timestamp ON staking_record (timestamp);


CREATE TABLE IF NOT EXISTS bridge_record(
    guid                   TEXT PRIMARY KEY DEFAULT replace(uuid_generate_v4()::text, '-', ''),
    source_chain_id        UINT256,
    dest_chain_id          UINT256,
    source_tx_hash         VARCHAR,
    dest_tx_hash           VARCHAR,
    source_block_number    UINT256,
    dest_block_number      UINT256,
    source_token_address   VARCHAR,
    dest_token_address     VARCHAR,
    token_decimal          VARCHAR,
    token_name             VARCHAR,
    msg_hash               VARCHAR,
    from_address           VARCHAR,
    to_address             VARCHAR,
    status                 SMALLINT not null,
    amount                 UINT256,
    nonce                  UINT256,
    fee                    UINT256,
    msg_sent_timestamp     INTEGER CHECK (msg_sent_timestamp > 0),
    claim_timestamp        INTEGER CHECK (claim_timestamp > 0)
);
CREATE INDEX IF NOT EXISTS bridge_record_source_chain_id ON bridge_record (source_chain_id);
CREATE INDEX IF NOT EXISTS bridge_record_dest_chain_id ON bridge_record (dest_chain_id);
CREATE INDEX IF NOT EXISTS bridge_record_source_tx_hash ON bridge_record (source_tx_hash);
CREATE INDEX IF NOT EXISTS bridge_record_dest_tx_hash ON bridge_record (dest_tx_hash);
CREATE INDEX IF NOT EXISTS bridge_record_msg_hash ON bridge_record (msg_hash);
CREATE INDEX IF NOT EXISTS bridge_record_source_block_number ON bridge_record (source_block_number);
CREATE INDEX IF NOT EXISTS bridge_record_dest_block_number ON bridge_record (dest_block_number);
CREATE INDEX IF NOT EXISTS bridge_record_source_token_address ON bridge_record (source_token_address);
CREATE INDEX IF NOT EXISTS bridge_record_dest_token_address ON bridge_record (dest_token_address);
CREATE INDEX IF NOT EXISTS bridge_record_from_address ON bridge_record (from_address);
CREATE INDEX IF NOT EXISTS bridge_record_to_address ON bridge_record (to_address);
CREATE INDEX IF NOT EXISTS bridge_record_status ON bridge_record (status);
CREATE INDEX IF NOT EXISTS bridge_record_msg_sent_timestamp ON bridge_record (msg_sent_timestamp);
CREATE INDEX IF NOT EXISTS bridge_record_claim_timestamp ON bridge_record (claim_timestamp);

CREATE TABLE IF NOT EXISTS token_config(
    guid                  TEXT    PRIMARY KEY DEFAULT replace(uuid_generate_v4()::text, '-', ''),
    chain_id               UINT256,
    token_address         VARCHAR  not null,
    token_name            VARCHAR,
    token_decimal         VARCHAR,
    timestamp             INTEGER  NOT NULL CHECK (timestamp > 0)
);
CREATE INDEX IF NOT EXISTS token_config_token_address ON token_config (token_address);
CREATE INDEX IF NOT EXISTS token_config_timestamp ON token_config (timestamp);
