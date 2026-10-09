-- 0001_init.up.sql
CREATE EXTENSION IF NOT EXISTS pgcrypto;  -- gen_random_uuid（PG16 内置，保险起见）

CREATE TABLE flags (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name          TEXT NOT NULL CHECK (length(trim(name)) BETWEEN 1 AND 100),
    key           TEXT NOT NULL CHECK (key ~ '^[a-z][a-z0-9._-]{1,63}$'),
    environment   TEXT NOT NULL CHECK (environment IN ('development','staging','production')),
    enabled       BOOLEAN NOT NULL DEFAULT true,
    default_value BOOLEAN NOT NULL DEFAULT false,
    description   TEXT NOT NULL DEFAULT '',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT uq_flags_env_key UNIQUE (environment, key)      -- ★ 题目硬性要求
);

CREATE TABLE rules (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    flag_id        UUID NOT NULL REFERENCES flags(id) ON DELETE CASCADE,
    attribute      TEXT NOT NULL CHECK (length(trim(attribute)) BETWEEN 1 AND 64),
    operator       TEXT NOT NULL CHECK (operator IN ('equals','in')),
    expected_value JSONB NOT NULL,
    roll_value     BOOLEAN NOT NULL,
    priority       INT NOT NULL CHECK (priority >= 1),
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT uq_rules_flag_priority UNIQUE (flag_id, priority)  -- ★ 重复优先级=拒绝
);

CREATE TABLE histories (
    id         BIGSERIAL PRIMARY KEY,
    flag_id    UUID NOT NULL REFERENCES flags(id) ON DELETE CASCADE,
    action     TEXT NOT NULL CHECK (action IN (
        'flag_created','flag_updated','flag_enabled','flag_disabled',
        'rule_created','rule_updated','rule_deleted','flag_copied')),
    operator   TEXT NOT NULL DEFAULT 'admin',
    summary    JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_histories_flag ON histories (flag_id, created_at DESC);
