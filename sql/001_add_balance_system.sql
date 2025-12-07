-- 万联APIrouter 余额系统扩展
-- 执行时间：第一阶段 MVP

-- 1. 余额交易记录表
CREATE TABLE IF NOT EXISTS balance_transactions (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    user_id BIGINT NOT NULL,
    amount DECIMAL(20, 8) NOT NULL COMMENT '交易金额（美元，可正可负）',
    balance_after DECIMAL(20, 8) NOT NULL COMMENT '交易后余额（美元）',
    type ENUM('recharge', 'usage', 'refund', 'adjustment') NOT NULL COMMENT '交易类型',
    reference_id VARCHAR(100) COMMENT '关联ID（如充值订单号、日志ID）',
    description VARCHAR(500) COMMENT '交易描述',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_user_created (user_id, created_at DESC),
    INDEX idx_type (type),
    INDEX idx_reference (reference_id),
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='余额交易记录表';

-- 2. 模型定价表
CREATE TABLE IF NOT EXISTS model_pricing (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    model_name VARCHAR(100) UNIQUE NOT NULL COMMENT '模型名称（如 openai/gpt-4）',
    display_name VARCHAR(200) COMMENT '显示名称',
    provider VARCHAR(50) NOT NULL COMMENT '供应商（如 openrouter）',
    description TEXT COMMENT '模型描述',
    context_length INT COMMENT '上下文长度',
    pricing_input DECIMAL(12, 8) COMMENT '输入价格（美元/Token）',
    pricing_output DECIMAL(12, 8) COMMENT '输出价格（美元/Token）',
    is_active BOOLEAN DEFAULT TRUE COMMENT '是否启用',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_model_name (model_name),
    INDEX idx_provider (provider),
    INDEX idx_active (is_active)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='模型定价表';

-- 3. 为日志表添加成本字段（如果不存在）
ALTER TABLE logs ADD COLUMN IF NOT EXISTS cost DECIMAL(20, 8) DEFAULT 0 COMMENT '成本（美元）' AFTER quota;

-- 4. 添加索引优化
ALTER TABLE logs ADD INDEX IF NOT EXISTS idx_created_cost (created_time, cost);
ALTER TABLE users ADD INDEX IF NOT EXISTS idx_quota (quota);

-- 注释：
-- - balance_transactions: 记录所有余额变动（充值、消费、退款等）
-- - model_pricing: 存储各模型价格，支持动态更新
-- - logs.cost: 记录每次请求的实际成本（美元）
-- - users.quota: 复用为美元余额（单位：分，1美元=100分）
