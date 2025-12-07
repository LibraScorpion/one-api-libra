# 余额系统规格

## Purpose
管理用户美元余额、充值和消费记录。

## Requirements

### Requirement: 美元余额存储
系统 MUST 以美元为单位管理用户余额。

#### Scenario: 余额精度
- GIVEN 系统使用 DECIMAL(20,8) 存储余额
- WHEN 进行金额计算
- THEN 精确到小数点后 8 位

#### Scenario: 余额初始化
- WHEN 新用户注册
- THEN 初始余额为 0.00 美元

### Requirement: 交易记录
系统 SHALL 记录所有余额变动。

#### Scenario: 充值记录
- WHEN 管理员为用户充值
- THEN 创建 type=recharge 的交易记录
- AND 更新用户余额
- AND 记录交易后余额

#### Scenario: 消费记录
- WHEN 用户调用 API 产生费用
- THEN 创建 type=usage 的交易记录
- AND 扣除相应金额
- AND 记录关联的请求 ID

### Requirement: 余额预检查
系统 MUST 在 API 调用前检查余额。

#### Scenario: 余额充足
- GIVEN 用户余额 >= 预估成本
- WHEN 发起 API 请求
- THEN 允许请求继续

#### Scenario: 余额不足
- GIVEN 用户余额 < 预估成本
- WHEN 发起 API 请求
- THEN 返回 402 Payment Required
- AND 返回错误信息 "Insufficient balance"

### Requirement: 成本计算
系统 SHALL 基于实际 Token 用量计算成本。

#### Scenario: 计算请求成本
- GIVEN 模型定价为 input=$X/M tokens, output=$Y/M tokens
- WHEN 请求消耗 A 个输入 token，B 个输出 token
- THEN 成本 = (A * X + B * Y) / 1,000,000

## Data Model

```sql
CREATE TABLE balance_transactions (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  user_id BIGINT NOT NULL,
  amount DECIMAL(20,8) NOT NULL,
  balance_after DECIMAL(20,8) NOT NULL,
  type ENUM('recharge','usage','refund','adjustment') NOT NULL,
  reference_id VARCHAR(100),
  description VARCHAR(500),
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  INDEX idx_user_created (user_id, created_at DESC)
);
```
