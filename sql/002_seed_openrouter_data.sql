-- 万联APIrouter OpenRouter 初始化数据
-- 执行时间：第一阶段 MVP

-- 1. 插入 OpenRouter 渠道（需要手动替换 API Key）
INSERT IGNORE INTO channels (
    id, type, `key`, status, name, weight, created_time,
    base_url, models, `group`, priority
) VALUES (
    1000,
    0,  -- type: 通用类型
    'YOUR_OPENROUTER_API_KEY_HERE',  -- 需要替换为实际的 API Key
    1,  -- status: 启用
    'OpenRouter',
    100,  -- weight: 默认权重
    UNIX_TIMESTAMP(),
    'https://openrouter.ai/api/v1',
    'openai/gpt-4,openai/gpt-3.5-turbo,anthropic/claude-3-opus,anthropic/claude-3-sonnet,google/gemini-pro,meta-llama/llama-3-70b-instruct',  -- 常用模型
    'default',
    0  -- priority: 默认优先级
);

-- 2. 插入常用模型定价（来自 OpenRouter 官方价格）
INSERT INTO model_pricing (model_name, display_name, provider, description, context_length, pricing_input, pricing_output) VALUES
-- OpenAI 模型
('openai/gpt-4', 'GPT-4', 'openrouter', 'OpenAI GPT-4 - 最强大的语言模型', 8192, 0.00003, 0.00006),
('openai/gpt-3.5-turbo', 'GPT-3.5 Turbo', 'openrouter', 'OpenAI GPT-3.5 - 快速且经济', 4096, 0.0000015, 0.000002),
('openai/gpt-4o', 'GPT-4o', 'openrouter', 'OpenAI GPT-4o - 多模态旗舰模型', 128000, 0.000005, 0.000015),
('openai/gpt-4o-mini', 'GPT-4o Mini', 'openrouter', 'OpenAI GPT-4o Mini - 经济实惠的智能模型', 128000, 0.00000015, 0.0000006),

-- Anthropic Claude 模型
('anthropic/claude-3-opus', 'Claude 3 Opus', 'openrouter', 'Anthropic Claude 3 Opus - 最强推理能力', 200000, 0.000015, 0.000075),
('anthropic/claude-3-sonnet', 'Claude 3 Sonnet', 'openrouter', 'Anthropic Claude 3 Sonnet - 平衡性能与成本', 200000, 0.000003, 0.000015),
('anthropic/claude-3.5-sonnet', 'Claude 3.5 Sonnet', 'openrouter', 'Anthropic Claude 3.5 Sonnet - 最新版本', 200000, 0.000003, 0.000015),
('anthropic/claude-3-haiku', 'Claude 3 Haiku', 'openrouter', 'Anthropic Claude 3 Haiku - 快速响应', 200000, 0.00000025, 0.00000125),

-- Google 模型
('google/gemini-pro', 'Gemini Pro', 'openrouter', 'Google Gemini Pro - 多模态能力', 32000, 0.000000125, 0.000000375),
('google/gemini-flash-1.5', 'Gemini 1.5 Flash', 'openrouter', 'Google Gemini 1.5 Flash - 快速高效', 1000000, 0.000000075, 0.0000003),
('google/gemini-pro-1.5', 'Gemini 1.5 Pro', 'openrouter', 'Google Gemini 1.5 Pro - 超长上下文', 2000000, 0.000000125, 0.000000375),

-- Meta Llama 模型
('meta-llama/llama-3-70b-instruct', 'Llama 3 70B', 'openrouter', 'Meta Llama 3 70B - 开源强大模型', 8192, 0.00000088, 0.00000088),
('meta-llama/llama-3.3-70b-instruct', 'Llama 3.3 70B', 'openrouter', 'Meta Llama 3.3 70B - 最新开源模型', 128000, 0.00000088, 0.00000088),
('meta-llama/llama-3.1-405b-instruct', 'Llama 3.1 405B', 'openrouter', 'Meta Llama 3.1 405B - 超大规模开源模型', 128000, 0.000003, 0.000003),

-- DeepSeek 模型
('deepseek/deepseek-chat', 'DeepSeek Chat', 'openrouter', 'DeepSeek Chat - 高性价比对话模型', 64000, 0.00000014, 0.00000028),
('deepseek/deepseek-r1', 'DeepSeek R1', 'openrouter', 'DeepSeek R1 - 推理增强模型', 64000, 0.00000055, 0.00000219),

-- Mistral 模型
('mistralai/mistral-large-2411', 'Mistral Large', 'openrouter', 'Mistral Large - 欧洲顶级模型', 128000, 0.000002, 0.000006),
('mistralai/mistral-small-24b-instruct-2501', 'Mistral Small', 'openrouter', 'Mistral Small - 经济高效', 32000, 0.0000001, 0.0000003),

-- Qwen 模型
('qwen/qwen-2.5-72b-instruct', 'Qwen 2.5 72B', 'openrouter', 'Qwen 2.5 72B - 阿里千问大模型', 32000, 0.00000036, 0.00000036),
('qwen/qwq-32b-preview', 'QwQ 32B', 'openrouter', 'QwQ 32B - 推理专用模型', 32000, 0.00000012, 0.00000012)
ON DUPLICATE KEY UPDATE
    pricing_input = VALUES(pricing_input),
    pricing_output = VALUES(pricing_output),
    description = VALUES(description),
    context_length = VALUES(context_length),
    updated_at = CURRENT_TIMESTAMP;

-- 3. 为 OpenRouter 渠道创建 abilities 映射
-- 注意：这里需要根据实际的 group 和 model 配置
-- 如果使用内存缓存，启动后会自动同步
INSERT IGNORE INTO abilities (`group`, model, channel_id, enabled, priority)
SELECT 'default', model_name, 1000, true, 0
FROM model_pricing
WHERE provider = 'openrouter' AND is_active = true
LIMIT 20;  -- 先添加前20个常用模型

-- 注释：
-- 1. 渠道 ID 1000 专门用于 OpenRouter
-- 2. 需要手动替换 YOUR_OPENROUTER_API_KEY_HERE 为实际的 API Key
-- 3. 价格定期从 OpenRouter 官网更新
-- 4. abilities 表用于模型路由，启动后自动同步
