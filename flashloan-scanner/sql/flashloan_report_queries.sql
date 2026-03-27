-- Flash-loan scanner query templates
-- Replace the placeholders before running:
--   :chain_id -> target chain id, e.g. 11155111
--   :limit    -> row limit, e.g. 20
--   :tx_hash  -> full transaction hash string, e.g. '0xabc...'

-- 1. Latest flash-loan transactions
SELECT
    chain_id,
    tx_hash,
    block_number,
    contains_candidate_interaction,
    contains_verified_interaction,
    contains_verified_strict_interaction,
    interaction_count,
    strict_interaction_count,
    protocol_count,
    protocols,
    updated_at
FROM flashloan_transactions
WHERE chain_id = :chain_id
ORDER BY block_number DESC, tx_hash DESC
LIMIT :limit;

-- 2. Latest strict-only transactions
SELECT
    chain_id,
    tx_hash,
    block_number,
    strict_interaction_count,
    protocols,
    updated_at
FROM flashloan_transactions
WHERE chain_id = :chain_id
  AND contains_verified_strict_interaction = TRUE
ORDER BY block_number DESC, tx_hash DESC
LIMIT :limit;

-- 3. Drill down one transaction
SELECT
    tx.chain_id,
    tx.tx_hash,
    tx.block_number,
    tx.contains_verified_interaction,
    tx.contains_verified_strict_interaction,
    pi.interaction_id,
    pi.interaction_ordinal,
    pi.protocol,
    pi.entrypoint,
    pi.provider_address,
    pi.receiver_address,
    pi.verified,
    pi.strict,
    pi.callback_seen,
    pi.repayment_seen,
    pi.contains_debt_opening,
    pi.exclusion_reason,
    pi.verification_notes
FROM flashloan_transactions tx
JOIN protocol_interactions pi
  ON pi.chain_id = tx.chain_id
 AND pi.tx_hash = tx.tx_hash
WHERE tx.chain_id = :chain_id
  AND tx.tx_hash = :tx_hash
ORDER BY pi.interaction_ordinal ASC;

-- 4. Drill down one transaction with legs
SELECT
    pi.chain_id,
    pi.tx_hash,
    pi.block_number,
    pi.protocol,
    pi.entrypoint,
    pi.interaction_id,
    pi.strict,
    legs.leg_index,
    legs.asset_address,
    legs.asset_role,
    legs.token_side,
    legs.amount_out,
    legs.amount_in,
    legs.amount_borrowed,
    legs.amount_repaid,
    legs.premium_amount,
    legs.fee_amount,
    legs.interest_rate_mode,
    legs.opened_debt,
    legs.strict_leg,
    legs.settlement_mode
FROM protocol_interactions pi
LEFT JOIN interaction_asset_legs legs
  ON legs.interaction_id = pi.interaction_id
WHERE pi.chain_id = :chain_id
  AND pi.tx_hash = :tx_hash
ORDER BY pi.interaction_ordinal ASC, legs.leg_index ASC;

-- 5. Protocol-level counts
SELECT
    protocol,
    COUNT(*) AS interaction_count,
    COUNT(*) FILTER (WHERE verified) AS verified_count,
    COUNT(*) FILTER (WHERE strict) AS strict_count
FROM protocol_interactions
WHERE chain_id = :chain_id
GROUP BY protocol
ORDER BY strict_count DESC, verified_count DESC, protocol ASC;

-- 6. Candidate vs verified vs strict summary
SELECT
    COUNT(*) AS total_interactions,
    COUNT(*) FILTER (WHERE candidate_level >= 2) AS candidate_interactions,
    COUNT(*) FILTER (WHERE verified) AS verified_interactions,
    COUNT(*) FILTER (WHERE strict) AS strict_interactions
FROM protocol_interactions
WHERE chain_id = :chain_id;

-- 7. Top exclusion reasons
SELECT
    protocol,
    exclusion_reason,
    COUNT(*) AS count
FROM protocol_interactions
WHERE chain_id = :chain_id
  AND exclusion_reason <> ''
GROUP BY protocol, exclusion_reason
ORDER BY count DESC, protocol ASC, exclusion_reason ASC
LIMIT :limit;

-- 8. Recent interactions that were verified but downgraded from strict
SELECT
    chain_id,
    tx_hash,
    block_number,
    protocol,
    entrypoint,
    callback_seen,
    repayment_seen,
    contains_debt_opening,
    verification_notes
FROM protocol_interactions
WHERE chain_id = :chain_id
  AND verified = TRUE
  AND strict = FALSE
ORDER BY block_number DESC, tx_hash DESC, interaction_ordinal ASC
LIMIT :limit;

-- 9. Asset-level borrowed volume by protocol
SELECT
    pi.protocol,
    legs.asset_address,
    COUNT(*) AS leg_count,
    SUM(legs.amount_borrowed) AS total_amount_borrowed
FROM protocol_interactions pi
JOIN interaction_asset_legs legs
  ON legs.interaction_id = pi.interaction_id
WHERE pi.chain_id = :chain_id
  AND pi.verified = TRUE
GROUP BY pi.protocol, legs.asset_address
ORDER BY total_amount_borrowed DESC NULLS LAST, leg_count DESC;

-- 10. Strict interactions with trace-related notes
SELECT
    chain_id,
    tx_hash,
    block_number,
    protocol,
    entrypoint,
    verification_notes
FROM protocol_interactions
WHERE chain_id = :chain_id
  AND strict = TRUE
  AND verification_notes ILIKE '%trace%'
ORDER BY block_number DESC, tx_hash DESC, interaction_ordinal ASC
LIMIT :limit;
