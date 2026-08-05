-- 000001_init.up.sql
-- 微盈通 V2 完整数据库结构 — T1 数据库基础设计
-- PostgreSQL 16 · BIGSERIAL 主键 · JSONB 灵活字段 · 软删除 · 多租户
-- 2026-08-04

-- ============================================================================
-- 扩展
-- ============================================================================
CREATE EXTENSION IF NOT EXISTS "pgcrypto";     -- gen_random_uuid, crypt (密码哈希)
CREATE EXTENSION IF NOT EXISTS "pg_trgm";      -- 三元组模糊匹配 (文章标题搜索)

-- ============================================================================
-- 通用触发器: 自动更新 updated_at 字段
-- ============================================================================
CREATE OR REPLACE FUNCTION update_timestamp()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;


-- ============================================================================
-- 1. tenants — 租户（平台用户）
-- ============================================================================
-- 每个注册用户是一个租户，可以管理多个微信公众号
-- 密码使用 bcrypt 哈希存储（pgcrypto 或应用层 bcrypt）
-- 软删除通过 deleted_at 实现，unique 索引排除已删除用户
-- ============================================================================
CREATE TABLE IF NOT EXISTS tenants (
    id              BIGSERIAL       PRIMARY KEY,
    username        VARCHAR(50)     NOT NULL,               -- 登录用户名，全局唯一
    password_hash   VARCHAR(255)    NOT NULL,               -- bcrypt 哈希，长度固定 60 字符
    nickname        VARCHAR(100),                           -- 显示昵称
    email           VARCHAR(200),                           -- 邮箱（可选，用于找回密码）
    phone           VARCHAR(20),                            -- 手机号（可选）
    avatar_url      VARCHAR(500),                           -- 头像 URL
    status          SMALLINT        NOT NULL DEFAULT 0,     -- 0=待审核 1=正常 2=停用
    last_login_at   TIMESTAMPTZ,                            -- 最近登录时间（审计）
    deleted_at      TIMESTAMPTZ,                            -- 软删除标记 (NULL=正常)
    created_at      TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ     NOT NULL DEFAULT NOW()
);

-- 用户名唯一约束（排除已软删除的记录）
CREATE UNIQUE INDEX IF NOT EXISTS idx_tenants_username
    ON tenants(username) WHERE deleted_at IS NULL;

-- 按状态筛选（后台管理常用）
CREATE INDEX IF NOT EXISTS idx_tenants_status
    ON tenants(status) WHERE deleted_at IS NULL;

-- 自动 updated_at
DROP TRIGGER IF EXISTS trg_tenants_updated_at ON tenants;
CREATE TRIGGER trg_tenants_updated_at
    BEFORE UPDATE ON tenants
    FOR EACH ROW EXECUTE FUNCTION update_timestamp();


-- ============================================================================
-- 2. wechat_accounts — 微信公众号
-- ============================================================================
-- 一个租户可以拥有多个公众号
-- 两种接入方式：auth_type=1 手动填写 AppId/AppSecret; auth_type=2 第三方平台授权
-- 同一 AppId 在手动手动模式下不可重复绑定（active 状态）
-- 第三方平台授权的 AppId 可能重复出现在不同环境？因此唯一约束仅针对手动模式
-- ============================================================================
CREATE TABLE IF NOT EXISTS wechat_accounts (
    id              BIGSERIAL       PRIMARY KEY,
    tenant_id       BIGINT          NOT NULL,               -- 所属租户
    name            VARCHAR(100),                           -- 公众号名称
    wx_original_id  VARCHAR(50),                            -- 微信原始 ID (gh_xxx)
    wx_app_id       VARCHAR(50),                            -- 开发者 ID (AppId)
    wx_app_secret   VARCHAR(200),                           -- 开发者密钥（手动接入必填）
    auth_type       SMALLINT        NOT NULL DEFAULT 1,     -- 1=手动接入 2=第三方平台授权
    auth_status     SMALLINT        NOT NULL DEFAULT 0,     -- 0=未接入 1=正常 2=令牌过期
    refresh_token   VARCHAR(512),                           -- 平台授权刷新令牌
    access_token    TEXT,                                   -- 当前有效的 authorizer_access_token
    token_expire_at TIMESTAMPTZ,                            -- access_token 过期时间
    avatar_url      VARCHAR(500),                           -- 公众号头像
    qr_code_url     VARCHAR(500),                           -- 公众号二维码
    description     VARCHAR(500),                           -- 公众号简介
    fans_count      INT             NOT NULL DEFAULT 0,     -- 粉丝数（定期同步）
    deleted_at      TIMESTAMPTZ,                            -- 软删除标记
    created_at      TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ     NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_accounts_tenant
        FOREIGN KEY (tenant_id) REFERENCES tenants(id)
);

-- 租户查询公众号列表（use case: "我的公众号"）
CREATE INDEX IF NOT EXISTS idx_accounts_tenant
    ON wechat_accounts(tenant_id) WHERE deleted_at IS NULL;

-- 手动接入模式：同一 AppId 不可被多个账号重复绑定
CREATE UNIQUE INDEX IF NOT EXISTS idx_accounts_app_id
    ON wechat_accounts(wx_app_id)
    WHERE auth_type = 1 AND auth_status IN (0, 1) AND deleted_at IS NULL;

-- 按接入状态筛选
CREATE INDEX IF NOT EXISTS idx_accounts_status
    ON wechat_accounts(auth_status) WHERE deleted_at IS NULL;

DROP TRIGGER IF EXISTS trg_wechat_accounts_updated_at ON wechat_accounts;
CREATE TRIGGER trg_wechat_accounts_updated_at
    BEFORE UPDATE ON wechat_accounts
    FOR EACH ROW EXECUTE FUNCTION update_timestamp();


-- ============================================================================
-- 3. auto_reply_rules — 自动回复规则
-- ============================================================================
-- 合并旧系统 WYT_MP_DEFAULT（默认回复）+ WYT_MP_KEYWORDS（关键词回复）
-- keyword IS NULL → 默认回复（当用户消息未匹配任何关键词时触发）
-- keyword IS NOT NULL → 关键词回复（匹配后返回对应内容）
-- reply_type: 1=文本 2=图文（标题+描述+封面+链接）
-- 图文回复存多条时，reply_content 可存 JSON 数组
-- ============================================================================
CREATE TABLE IF NOT EXISTS auto_reply_rules (
    id              BIGSERIAL       PRIMARY KEY,
    account_id      BIGINT          NOT NULL,               -- 所属公众号
    keyword         VARCHAR(200),                           -- 触发关键词；NULL=默认回复
    match_type      SMALLINT        NOT NULL DEFAULT 0,     -- 0=完全匹配 1=包含匹配
    reply_type      SMALLINT        NOT NULL DEFAULT 1,     -- 1=文本 2=图文
    -- 文本回复
    reply_content   TEXT            NOT NULL DEFAULT '',    -- 文本内容；图文模式下存 JSON [{title,desc,cover,url}]
    -- 图文回复字段（reply_type=2 时使用，也可序列化到 reply_content JSON 中）
    reply_title     VARCHAR(200),                           -- 图文标题
    reply_desc      VARCHAR(500),                           -- 图文描述/摘要
    reply_cover_url VARCHAR(500),                           -- 图文封面图 URL
    reply_url       VARCHAR(500),                           -- 图文原文链接
    -- 通用
    status          SMALLINT        NOT NULL DEFAULT 1,     -- 0=停用 1=启用
    sort_order      INT             NOT NULL DEFAULT 0,     -- 排序权重（同关键词多条时的优先级）
    deleted_at      TIMESTAMPTZ,                            -- 软删除
    created_at      TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ     NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_auto_reply_account
        FOREIGN KEY (account_id) REFERENCES wechat_accounts(id)
);

-- 查询某公众号下所有启用的规则
CREATE INDEX IF NOT EXISTS idx_auto_reply_account_status
    ON auto_reply_rules(account_id, status) WHERE deleted_at IS NULL;

-- 按关键词查找规则（消息路由时使用）
CREATE INDEX IF NOT EXISTS idx_auto_reply_keyword
    ON auto_reply_rules(account_id, keyword) WHERE status = 1 AND deleted_at IS NULL;

DROP TRIGGER IF EXISTS trg_auto_reply_rules_updated_at ON auto_reply_rules;
CREATE TRIGGER trg_auto_reply_rules_updated_at
    BEFORE UPDATE ON auto_reply_rules
    FOR EACH ROW EXECUTE FUNCTION update_timestamp();


-- ============================================================================
-- 4. cms_channels — 微官网栏目（树形结构）
-- ============================================================================
-- 支持无限层级，通过 parent_id 自引用实现
-- level 字段缓存深度（0=根栏目），便于查询优化
-- 同一公众号下 slug 唯一
-- ============================================================================
CREATE TABLE IF NOT EXISTS cms_channels (
    id              BIGSERIAL       PRIMARY KEY,
    account_id      BIGINT          NOT NULL,               -- 所属公众号
    parent_id       BIGINT,                                 -- 父栏目 ID；NULL=顶级栏目
    name            VARCHAR(100)    NOT NULL,               -- 栏目名称
    slug            VARCHAR(100),                           -- URL 友好标识（英文/拼音）
    level           SMALLINT        NOT NULL DEFAULT 0,     -- 层级深度（0=顶级）
    sort_order      INT             NOT NULL DEFAULT 0,     -- 同级排序
    cover_url       VARCHAR(500),                           -- 栏目封面图
    description     VARCHAR(500),                           -- 栏目描述
    status          SMALLINT        NOT NULL DEFAULT 1,     -- 0=隐藏 1=显示
    deleted_at      TIMESTAMPTZ,                            -- 软删除
    created_at      TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ     NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_channels_account
        FOREIGN KEY (account_id) REFERENCES wechat_accounts(id),
    CONSTRAINT fk_channels_parent
        FOREIGN KEY (parent_id) REFERENCES cms_channels(id)
);

-- 查询某公众号下所有栏目（含层级排序）
CREATE INDEX IF NOT EXISTS idx_channels_account
    ON cms_channels(account_id, parent_id, sort_order) WHERE deleted_at IS NULL;

-- slug + account_id 唯一
CREATE UNIQUE INDEX IF NOT EXISTS idx_channels_slug
    ON cms_channels(account_id, slug) WHERE deleted_at IS NULL;

DROP TRIGGER IF EXISTS trg_cms_channels_updated_at ON cms_channels;
CREATE TRIGGER trg_cms_channels_updated_at
    BEFORE UPDATE ON cms_channels
    FOR EACH ROW EXECUTE FUNCTION update_timestamp();


-- ============================================================================
-- 5. cms_articles — 微官网文章
-- ============================================================================
-- 核心内容表：存储 TipTap 编辑器输出的 JSONB 结构
-- html_cache 用于服务端渲染缓存，避免每次请求都解析 JSONB
-- status: 草稿→发布，支持编辑后重新草稿 → 再发布
-- is_template: 存为模板后可一键复用
-- ============================================================================
CREATE TABLE IF NOT EXISTS cms_articles (
    id              BIGSERIAL       PRIMARY KEY,
    account_id      BIGINT          NOT NULL,               -- 所属公众号
    channel_id      BIGINT,                                 -- 所属栏目（可为空，独立文章）
    title           VARCHAR(200),                           -- 文章标题
    cover_url       VARCHAR(500),                           -- 封面图 URL
    summary         VARCHAR(500),                           -- 摘要 / 朋友圈分享描述
    author          VARCHAR(100),                           -- 作者署名
    -- JSONB 存储 TipTap 编辑器完整文档结构
    -- 格式: { "type": "doc", "content": [{ "type": "heading", ... }, { "type": "imageBlock", ... }, ...] }
    content         JSONB           NOT NULL DEFAULT '{}',
    html_cache      TEXT,                                   -- 预渲染的 HTML 缓存（服务端渲染用）
    status          SMALLINT        NOT NULL DEFAULT 0,     -- 0=草稿 1=已发布
    is_template     BOOLEAN         NOT NULL DEFAULT FALSE, -- 是否存为排版模板
    template_cat    VARCHAR(50),                            -- 模板分类: holiday/activity/industry/news
    sort_order      INT             NOT NULL DEFAULT 0,     -- 栏目内排序
    view_count      INT             NOT NULL DEFAULT 0,     -- 浏览次数（H5 展示页计数）
    published_at    TIMESTAMPTZ,                            -- 发布时间
    deleted_at      TIMESTAMPTZ,                            -- 软删除
    created_at      TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ     NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_articles_account
        FOREIGN KEY (account_id) REFERENCES wechat_accounts(id),
    CONSTRAINT fk_articles_channel
        FOREIGN KEY (channel_id) REFERENCES cms_channels(id)
);

-- 按公众号+状态查文章列表（管理后台最常用查询）
CREATE INDEX IF NOT EXISTS idx_articles_account_status
    ON cms_articles(account_id, status, created_at DESC)
    WHERE deleted_at IS NULL;

-- 按栏目查文章（前端 H5 栏目页渲染）
CREATE INDEX IF NOT EXISTS idx_articles_channel_status
    ON cms_articles(channel_id, status, sort_order)
    WHERE deleted_at IS NULL;

-- JSONB 索引：加速对 content 内部字段的查询（如查找包含某组件的文章）
CREATE INDEX IF NOT EXISTS idx_articles_content_gin
    ON cms_articles USING GIN (content);

-- 标题模糊搜索（管理后台搜索框）
CREATE INDEX IF NOT EXISTS idx_articles_title_trgm
    ON cms_articles USING GIN (title gin_trgm_ops)
    WHERE deleted_at IS NULL;

-- 模板查询
CREATE INDEX IF NOT EXISTS idx_articles_templates
    ON cms_articles(account_id, template_cat)
    WHERE is_template = TRUE AND deleted_at IS NULL;

DROP TRIGGER IF EXISTS trg_cms_articles_updated_at ON cms_articles;
CREATE TRIGGER trg_cms_articles_updated_at
    BEFORE UPDATE ON cms_articles
    FOR EACH ROW EXECUTE FUNCTION update_timestamp();


-- ============================================================================
-- 6. quicknews_channels — 快讯栏目
-- ============================================================================
-- 快讯（一句话新闻）的栏目分类
-- 一个公众号可以有多个快讯栏目
-- ============================================================================
CREATE TABLE IF NOT EXISTS quicknews_channels (
    id              BIGSERIAL       PRIMARY KEY,
    account_id      BIGINT          NOT NULL,               -- 所属公众号
    name            VARCHAR(100)    NOT NULL,               -- 栏目名称
    cover_url       VARCHAR(500),                           -- 栏目封面图（H5 顶部轮播用）
    description     VARCHAR(500),                           -- 栏目简介
    sort_order      INT             NOT NULL DEFAULT 0,     -- 排序
    status          SMALLINT        NOT NULL DEFAULT 1,     -- 0=隐藏 1=显示
    deleted_at      TIMESTAMPTZ,                            -- 软删除
    created_at      TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ     NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_qn_channels_account
        FOREIGN KEY (account_id) REFERENCES wechat_accounts(id)
);

CREATE INDEX IF NOT EXISTS idx_qn_channels_account
    ON quicknews_channels(account_id, sort_order) WHERE deleted_at IS NULL;

DROP TRIGGER IF EXISTS trg_quicknews_channels_updated_at ON quicknews_channels;
CREATE TRIGGER trg_quicknews_channels_updated_at
    BEFORE UPDATE ON quicknews_channels
    FOR EACH ROW EXECUTE FUNCTION update_timestamp();


-- ============================================================================
-- 7. quicknews_users — 快讯用户（微信 OAuth 注册）
-- ============================================================================
-- H5 快讯页面通过微信 OAuth 获取用户 openid 后自动注册
-- 用户在 H5 端可发布快讯、点赞、评论（取决于后台配置）
-- openid 在同一个公众号下唯一
-- ============================================================================
CREATE TABLE IF NOT EXISTS quicknews_users (
    id              BIGSERIAL       PRIMARY KEY,
    account_id      BIGINT          NOT NULL,               -- 所属公众号
    openid          VARCHAR(100)    NOT NULL,               -- 微信 openid
    unionid         VARCHAR(100),                           -- 微信 unionid（开放平台绑定后可用）
    nickname        VARCHAR(100),                           -- 微信昵称
    avatar_url      VARCHAR(500),                           -- 微信头像
    status          SMALLINT        NOT NULL DEFAULT 1,     -- 0=封禁 1=正常
    created_at      TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ     NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_qn_users_account
        FOREIGN KEY (account_id) REFERENCES wechat_accounts(id)
);

-- 公众号 + openid 唯一（一个微信用户在一个公众号下只有一个身份）
CREATE UNIQUE INDEX IF NOT EXISTS idx_qn_users_account_openid
    ON quicknews_users(account_id, openid);

-- 按公众号查用户列表
CREATE INDEX IF NOT EXISTS idx_qn_users_account
    ON quicknews_users(account_id, created_at DESC);

DROP TRIGGER IF EXISTS trg_quicknews_users_updated_at ON quicknews_users;
CREATE TRIGGER trg_quicknews_users_updated_at
    BEFORE UPDATE ON quicknews_users
    FOR EACH ROW EXECUTE FUNCTION update_timestamp();


-- ============================================================================
-- 8. quicknews_news — 快讯内容
-- ============================================================================
-- 一条快讯 = 昵称 + 头像 + 文字内容 + 可选图片组
-- 可由管理后台发布（user_id=NULL）或 H5 用户发布（user_id 关联 quicknews_users）
-- author_name/author_avatar 冗余存储，避免 JOIN，提升 H5 列表渲染性能
-- ============================================================================
CREATE TABLE IF NOT EXISTS quicknews_news (
    id              BIGSERIAL       PRIMARY KEY,
    account_id      BIGINT          NOT NULL,               -- 所属公众号（直接冗余，避免 JOIN channel）
    channel_id      BIGINT          NOT NULL,               -- 所属栏目
    user_id         BIGINT,                                 -- 发布者（H5 用户）；NULL=后台管理员发布
    author_name     VARCHAR(100),                           -- 作者昵称（冗余）
    author_avatar   VARCHAR(500),                           -- 作者头像（冗余）
    content         TEXT            NOT NULL,               -- 正文内容
    like_count      INT             NOT NULL DEFAULT 0,     -- 点赞数（冗余计数，定期同步）
    comment_count   INT             NOT NULL DEFAULT 0,     -- 评论数（冗余计数）
    status          SMALLINT        NOT NULL DEFAULT 1,     -- 0=草稿 1=已发布 2=隐藏
    is_top          BOOLEAN         NOT NULL DEFAULT FALSE, -- 是否置顶
    deleted_at      TIMESTAMPTZ,                            -- 软删除
    created_at      TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ     NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_qn_news_account
        FOREIGN KEY (account_id) REFERENCES wechat_accounts(id),
    CONSTRAINT fk_qn_news_channel
        FOREIGN KEY (channel_id) REFERENCES quicknews_channels(id),
    CONSTRAINT fk_qn_news_user
        FOREIGN KEY (user_id) REFERENCES quicknews_users(id)
);

-- H5 列表查询：按栏目+状态+时间倒序（最常见）
CREATE INDEX IF NOT EXISTS idx_qn_news_channel_status
    ON quicknews_news(channel_id, status, is_top DESC, created_at DESC)
    WHERE deleted_at IS NULL;

-- 管理后台：按公众号查询
CREATE INDEX IF NOT EXISTS idx_qn_news_account
    ON quicknews_news(account_id, status, created_at DESC)
    WHERE deleted_at IS NULL;

-- 按 user_id 查某用户发布的所有快讯
CREATE INDEX IF NOT EXISTS idx_qn_news_user
    ON quicknews_news(user_id, created_at DESC)
    WHERE user_id IS NOT NULL AND deleted_at IS NULL;

DROP TRIGGER IF EXISTS trg_quicknews_news_updated_at ON quicknews_news;
CREATE TRIGGER trg_quicknews_news_updated_at
    BEFORE UPDATE ON quicknews_news
    FOR EACH ROW EXECUTE FUNCTION update_timestamp();


-- ============================================================================
-- 9. quicknews_photos — 快讯图片
-- ============================================================================
-- 一条快讯可附带多张图片，可控制排序
-- ============================================================================
CREATE TABLE IF NOT EXISTS quicknews_photos (
    id              BIGSERIAL       PRIMARY KEY,
    news_id         BIGINT          NOT NULL,               -- 所属快讯
    url             VARCHAR(500)    NOT NULL,               -- 图片 URL
    sort_order      INT             NOT NULL DEFAULT 0,     -- 排序（值小在前）
    created_at      TIMESTAMPTZ     NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_qn_photos_news
        FOREIGN KEY (news_id) REFERENCES quicknews_news(id)
);

CREATE INDEX IF NOT EXISTS idx_qn_photos_news
    ON quicknews_photos(news_id, sort_order);


-- ============================================================================
-- 10. quicknews_likes — 快讯点赞记录
-- ============================================================================
-- 记录每个用户对每条快讯的点赞，用于去重
-- 唯一约束确保同一个用户对同一条快讯只能点赞一次
-- ============================================================================
CREATE TABLE IF NOT EXISTS quicknews_likes (
    id              BIGSERIAL       PRIMARY KEY,
    news_id         BIGINT          NOT NULL,               -- 被点赞的快讯
    user_id         BIGINT,                                 -- 点赞者（已注册用户）
    openid          VARCHAR(100)    NOT NULL,               -- 点赞者 openid（兼容未注册场景）
    created_at      TIMESTAMPTZ     NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_qn_likes_news
        FOREIGN KEY (news_id) REFERENCES quicknews_news(id),
    CONSTRAINT fk_qn_likes_user
        FOREIGN KEY (user_id) REFERENCES quicknews_users(id)
);

-- 唯一约束：同一快讯 + 同一 openid 只能点赞一次
CREATE UNIQUE INDEX IF NOT EXISTS idx_qn_likes_news_openid
    ON quicknews_likes(news_id, openid);

-- 按快讯查点赞列表
CREATE INDEX IF NOT EXISTS idx_qn_likes_news
    ON quicknews_likes(news_id);


-- ============================================================================
-- 11. quicknews_comments — 快讯评论
-- ============================================================================
-- 支持一级回复（parent_id 可嵌套）
-- 评论需关联注册用户
-- ============================================================================
CREATE TABLE IF NOT EXISTS quicknews_comments (
    id              BIGSERIAL       PRIMARY KEY,
    news_id         BIGINT          NOT NULL,               -- 被评论的快讯
    user_id         BIGINT          NOT NULL,               -- 评论者
    parent_id       BIGINT,                                 -- 父评论 ID（回复某人）；NULL=顶级评论
    content         TEXT            NOT NULL,               -- 评论内容
    like_count      INT             NOT NULL DEFAULT 0,     -- 评论点赞数
    status          SMALLINT        NOT NULL DEFAULT 1,     -- 0=隐藏 1=显示
    deleted_at      TIMESTAMPTZ,                            -- 软删除
    created_at      TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ     NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_qn_comments_news
        FOREIGN KEY (news_id) REFERENCES quicknews_news(id),
    CONSTRAINT fk_qn_comments_user
        FOREIGN KEY (user_id) REFERENCES quicknews_users(id),
    CONSTRAINT fk_qn_comments_parent
        FOREIGN KEY (parent_id) REFERENCES quicknews_comments(id)
);

-- H5 快讯详情页：按快讯查评论树
CREATE INDEX IF NOT EXISTS idx_qn_comments_news
    ON quicknews_comments(news_id, parent_id, created_at)
    WHERE deleted_at IS NULL;

-- 按用户查评论历史
CREATE INDEX IF NOT EXISTS idx_qn_comments_user
    ON quicknews_comments(user_id, created_at DESC)
    WHERE deleted_at IS NULL;

DROP TRIGGER IF EXISTS trg_quicknews_comments_updated_at ON quicknews_comments;
CREATE TRIGGER trg_quicknews_comments_updated_at
    BEFORE UPDATE ON quicknews_comments
    FOR EACH ROW EXECUTE FUNCTION update_timestamp();


-- ============================================================================
-- 12. votes — 投票活动
-- ============================================================================
-- 支持单选/多选，时间有效期控制，每用户投票次数限制
-- total_votes 冗余计数器，避免每次统计都 COUNT(vote_records)
-- ============================================================================
CREATE TABLE IF NOT EXISTS votes (
    id              BIGSERIAL       PRIMARY KEY,
    account_id      BIGINT          NOT NULL,               -- 所属公众号
    title           VARCHAR(200)    NOT NULL,               -- 投票标题
    description     TEXT,                                   -- 投票说明 / 规则
    cover_url       VARCHAR(500),                           -- 封面图
    vote_type       SMALLINT        NOT NULL DEFAULT 1,     -- 1=单选 2=多选
    max_choices     INT             NOT NULL DEFAULT 1,     -- 多选时最多可选几项
    max_votes       INT             NOT NULL DEFAULT 1,     -- 每用户最多可投票次数
    start_time      TIMESTAMPTZ,                            -- 开始时间
    end_time        TIMESTAMPTZ,                            -- 结束时间
    total_votes     INT             NOT NULL DEFAULT 0,     -- 总投票人数（冗余）
    status          SMALLINT        NOT NULL DEFAULT 0,     -- 0=草稿 1=进行中 2=已结束
    deleted_at      TIMESTAMPTZ,                            -- 软删除
    created_at      TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ     NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_votes_account
        FOREIGN KEY (account_id) REFERENCES wechat_accounts(id)
);

-- 管理后台：按公众号+状态查询
CREATE INDEX IF NOT EXISTS idx_votes_account_status
    ON votes(account_id, status, created_at DESC)
    WHERE deleted_at IS NULL;

-- H5 展示：按时间范围查询进行中的投票
CREATE INDEX IF NOT EXISTS idx_votes_time_range
    ON votes(status, start_time, end_time)
    WHERE deleted_at IS NULL;

DROP TRIGGER IF EXISTS trg_votes_updated_at ON votes;
CREATE TRIGGER trg_votes_updated_at
    BEFORE UPDATE ON votes
    FOR EACH ROW EXECUTE FUNCTION update_timestamp();


-- ============================================================================
-- 13. vote_options — 投票选项
-- ============================================================================
-- 每个投票活动下可以有多个选项
-- vote_count 冗余计数器（高并发下可用 Redis 缓存，定期回写 DB）
-- ============================================================================
CREATE TABLE IF NOT EXISTS vote_options (
    id              BIGSERIAL       PRIMARY KEY,
    vote_id         BIGINT          NOT NULL,               -- 所属投票
    content         VARCHAR(500)    NOT NULL,               -- 选项文本
    image_url       VARCHAR(500),                           -- 选项配图
    sort_order      INT             NOT NULL DEFAULT 0,     -- 排序
    vote_count      INT             NOT NULL DEFAULT 0,     -- 获票数（冗余）

    CONSTRAINT fk_vote_options_vote
        FOREIGN KEY (vote_id) REFERENCES votes(id)
);

-- 查询某投票下的所有选项（按排序）
CREATE INDEX IF NOT EXISTS idx_vote_options_vote
    ON vote_options(vote_id, sort_order);


-- ============================================================================
-- 14. vote_records — 投票记录
-- ============================================================================
-- 记录每个用户对每个选项的每次投票
-- 通过 openid 识别用户身份（微信 OAuth 获取）
-- ip_address / user_agent 用于防刷和审计
-- ============================================================================
CREATE TABLE IF NOT EXISTS vote_records (
    id              BIGSERIAL       PRIMARY KEY,
    vote_id         BIGINT          NOT NULL,               -- 所属投票活动
    option_id       BIGINT          NOT NULL,               -- 被投选项
    openid          VARCHAR(100)    NOT NULL,               -- 投票者 openid
    ip_address      VARCHAR(50),                            -- 投票者 IP（防刷/审计）
    user_agent      VARCHAR(500),                           -- 投票者 UA（防刷/审计）
    created_at      TIMESTAMPTZ     NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_vote_records_vote
        FOREIGN KEY (vote_id) REFERENCES votes(id),
    CONSTRAINT fk_vote_records_option
        FOREIGN KEY (option_id) REFERENCES vote_options(id)
);

-- 投票次数校验：查询某用户在某投票中已投次数
CREATE INDEX IF NOT EXISTS idx_vote_records_vote_openid
    ON vote_records(vote_id, openid);

-- 按选项统计票数
CREATE INDEX IF NOT EXISTS idx_vote_records_option
    ON vote_records(option_id);


-- ============================================================================
-- 15. materials — 素材库
-- ============================================================================
-- 存储公众号的图片/语音/视频/文件素材
-- media_id: 上传到微信服务器后返回的素材 ID（用于微信 API 调用）
-- 本地/S3 URL 用于后台展示和编辑器拖入
-- ============================================================================
CREATE TABLE IF NOT EXISTS materials (
    id              BIGSERIAL       PRIMARY KEY,
    account_id      BIGINT          NOT NULL,               -- 所属公众号
    media_id        VARCHAR(100),                           -- 微信素材 ID（上传到微信后获得）
    type            VARCHAR(20)     NOT NULL,               -- image / voice / video / thumb / file
    name            VARCHAR(200),                           -- 文件名
    url             VARCHAR(500)    NOT NULL,               -- 访问 URL（本地或 S3）
    thumbnail_url   VARCHAR(500),                           -- 缩略图 URL（视频/文件封面）
    file_size       BIGINT,                                 -- 文件大小（字节）
    width           INT,                                    -- 图片/视频宽度
    height          INT,                                    -- 图片/视频高度
    format          VARCHAR(20),                            -- 文件扩展名: jpg/png/mp4/mp3/pdf
    deleted_at      TIMESTAMPTZ,                            -- 软删除
    created_at      TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ     NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_materials_account
        FOREIGN KEY (account_id) REFERENCES wechat_accounts(id)
);

-- 按公众号+类型查素材（编辑器素材面板）
CREATE INDEX IF NOT EXISTS idx_materials_account_type
    ON materials(account_id, type, created_at DESC)
    WHERE deleted_at IS NULL;

-- 按微信 media_id 查找（调用微信 API 时需要）
CREATE INDEX IF NOT EXISTS idx_materials_media_id
    ON materials(account_id, media_id)
    WHERE media_id IS NOT NULL AND deleted_at IS NULL;

DROP TRIGGER IF EXISTS trg_materials_updated_at ON materials;
CREATE TRIGGER trg_materials_updated_at
    BEFORE UPDATE ON materials
    FOR EACH ROW EXECUTE FUNCTION update_timestamp();


-- ============================================================================
-- 16. system_configs — 系统配置表
-- ============================================================================
-- 存储系统和租户级别的配置项
-- account_id IS NULL → 全局配置（平台级，仅超管可修改）
-- account_id IS NOT NULL → 租户级配置（租户可自定义）
-- 典型用例：AI 模型选择、上传大小限制、水印设置等
-- ============================================================================
CREATE TABLE IF NOT EXISTS system_configs (
    id              BIGSERIAL       PRIMARY KEY,
    account_id      BIGINT,                                 -- NULL=全局配置；非NULL=租户级配置
    key             VARCHAR(100)    NOT NULL,               -- 配置键（如 ai.model, upload.max_size_mb）
    value           TEXT,                                   -- 配置值
    type            VARCHAR(20)     NOT NULL DEFAULT 'string', -- 值类型: string / number / bool / json
    description     VARCHAR(500),                           -- 配置说明
    created_at      TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ     NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_sys_config_account
        FOREIGN KEY (account_id) REFERENCES wechat_accounts(id)
);

-- 每个 scope 下的 key 唯一
CREATE UNIQUE INDEX IF NOT EXISTS idx_sys_config_key
    ON system_configs(COALESCE(account_id, 0), key);

-- 按租户查配置
CREATE INDEX IF NOT EXISTS idx_sys_config_account
    ON system_configs(account_id) WHERE account_id IS NOT NULL;

DROP TRIGGER IF EXISTS trg_system_configs_updated_at ON system_configs;
CREATE TRIGGER trg_system_configs_updated_at
    BEFORE UPDATE ON system_configs
    FOR EACH ROW EXECUTE FUNCTION update_timestamp();


-- ============================================================================
-- 迁移完成标记
-- golang-migrate 会自动管理 schema_migrations 表，记录已执行的迁移版本
-- ============================================================================
