-- 分组级 Astra/Ultrafast 档位策略。
-- force_openai_fast 开启时，Astra 模型是否升级为 service_tier=ultrafast；
-- ultrafast_multiplier 为该分组 ultrafast 请求的计费倍率（0=用模型默认）。
-- 均为叠加式新增，默认值保持既有行为不变。

ALTER TABLE groups
    ADD COLUMN IF NOT EXISTS force_openai_astra_tier VARCHAR(16) NOT NULL DEFAULT 'auto',
    ADD COLUMN IF NOT EXISTS ultrafast_multiplier DECIMAL(10, 4) NOT NULL DEFAULT 0;

COMMENT ON COLUMN groups.force_openai_astra_tier IS
    'force_openai_fast 开启时 Astra 模型档位：auto=priority（默认），ultrafast=Astra 升级为 ultrafast';
COMMENT ON COLUMN groups.ultrafast_multiplier IS
    '分组级 ultrafast 计费倍率；0=使用模型默认（Astra 为 6）';
