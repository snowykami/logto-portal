# Yuki ID Portal

`Yuki ID Portal` 是面向轻雪用户的账号门户，部署域名为：

```text
https://account.liteyuki.org
```

它以 Logto / Yuki ID 作为身份底座，提供更完整的用户体验层，包括账号资料、安全管理、应用入口、公告通知、统一退出和帮助支持。

## 目标

Logto 负责身份认证和 IAM 能力，Yuki ID Portal 负责面向用户的账号门户体验。

```text
Logto / Yuki ID:
- 登录认证
- OIDC / OAuth2
- 用户资料
- MFA
- 角色
- 组织
- 会话
- 应用授权

Yuki ID Portal:
- 用户首页
- 资料整理
- 安全中心
- 应用入口
- 公告通知
- 统一退出
- 权限说明
- 帮助支持
```

## 技术栈

项目采用 monorepo + 容器化构建。

后端：

```text
Go
Gin
embed.FS
Logto Account API
Logto Management API
```

前端：

```text
React
Vite
TypeScript
Tailwind CSS
shadcn/ui
```

数据存储：

```text
优先无数据库
可选 PostgreSQL
```

构建与运行形态：

```text
frontend build
  -> 生成静态资源
  -> Go embed.FS 嵌入后端二进制
  -> 单容器运行
```

路由约定：

```text
/      前端应用，由 Go 后端返回嵌入的静态资源
/api   后端 API
```

容器化目标：

```text
单镜像部署
前端静态资源随后端二进制发布
运行时不依赖 Node.js
```

## 数据策略

第一版优先不引入数据库，尽量把 Logto 作为身份与账号资料的事实源。

无数据库模式适合：

```text
用户资料读取与修改
安全中心入口
会话管理
应用入口展示
权限说明
统一退出
帮助页
静态公告
迁移提示
```

无数据库模式下，Portal 自有配置可以来自：

```text
仓库内 YAML / JSON 配置
环境变量
构建时嵌入的配置文件
后端只读配置目录
```

例如应用目录可以先用配置文件维护：

```text
app-catalog.yaml
```

公告也可以先用配置文件维护：

```text
announcements.yaml
```

只有出现以下需求时，再启用 PostgreSQL：

```text
公告已读 / 未读
用户偏好
公告后台管理
应用目录后台管理
审计日志
用户级别的迁移状态
按用户精确投递通知
```

推荐策略：

```text
P0: 无数据库，全部账号数据来自 Logto，Portal 自有数据使用静态配置。
P1: 如需要已读状态、后台管理和审计，再引入 PostgreSQL。
```

即使引入 PostgreSQL，也只保存 Portal 自有状态，不保存 Logto 的密码、社交 token、Management API token 或用户资料副本。

## 官方参考文档

- [Logto Account settings](https://docs.logto.io/end-user-flows/account-settings)
- [Logto Account API](https://docs.logto.io/end-user-flows/account-settings/by-account-api)
- [Logto Management API](https://docs.logto.io/integrate-logto/interact-with-management-api)
- [Logto Sessions](https://docs.logto.io/sessions)
- [Logto Sign-out](https://docs.logto.io/end-user-flows/sign-out)
- [Logto Organization experience](https://docs.logto.io/end-user-flows/organization-experience)
- [Logto User data structure](https://docs.logto.io/user-management/user-data)
- [Logto Application data structure](https://docs.logto.io/integrate-logto/application-data-structure)

## Logto 配置

Logto 地址：

```text
https://auth.liteyuki.org
```

OIDC Issuer：

```text
https://auth.liteyuki.org/oidc
```

在 Logto 中创建应用：

```text
Application name: Yuki ID Portal
Application type: Traditional Web，或 SPA + BFF
Redirect URI:
https://account.liteyuki.org/auth/callback

Post logout redirect URI:
https://account.liteyuki.org/

CORS Allowed Origins:
https://account.liteyuki.org
```

建议请求 scopes：

```text
openid profile email roles urn:logto:scope:organizations urn:logto:scope:organization_roles
```

## 架构

推荐采用 BFF 架构：

```text
Browser
  -> account.liteyuki.org Frontend
  -> account.liteyuki.org Backend / BFF
  -> Logto Account API
  -> Logto Management API
  -> Optional Portal Database
```

前端负责页面展示和交互。

后端负责：

- 校验用户登录态
- 代理需要服务端调用的 Logto API
- 调用 Logto Management API
- 管理公告
- 管理应用目录
- 记录审计日志
- 裁剪返回给前端的数据

## 安全原则

- 浏览器端不得持有 Logto Management API token。
- Management API 只能由后端服务调用。
- 用户唯一标识使用 Logto 的 `sub`，不要使用邮箱作为主键。
- 邮箱、昵称、头像等资料以 Logto 为准。
- Portal 不保存 Logto 密码、社交登录 token 或高权限密钥。
- 所有敏感操作需要二次确认。
- 所有管理操作需要记录审计日志。
- 后端必须校验 token 的签名、`iss`、`aud`、`exp` 和必要 scope。
- 后端不得直接把 Logto Management API 的完整响应透传给前端。

## 功能范围

### P0

#### 登录与会话

- 未登录访问门户时跳转 Logto 登录。
- 登录后展示当前用户信息。
- 支持退出当前门户。
- 支持退出 Yuki ID 中心登录态。

#### 首页 Dashboard

展示：

- 用户头像
- 昵称
- 用户名
- 邮箱
- 当前 roles
- 当前 organizations
- 当前 organization_roles
- 常用应用入口
- 最新公告
- 账号安全提醒

#### 个人资料

支持：

- 查看基础资料
- 修改昵称
- 修改头像
- 修改用户名
- 查看邮箱
- 查看手机号
- 查看社交账号绑定状态

资料修改优先使用 Logto Account API。

#### 安全中心

支持：

- 修改密码入口
- MFA 状态展示
- MFA 管理入口
- 社交账号绑定 / 解绑入口
- 活跃会话列表
- 退出指定会话
- 退出全部会话

#### 应用与授权

展示轻雪已接入应用，例如：

- Gitea
- Grafana
- Harbor
- OpenList
- 社区
- DevOps 平台

每个应用展示：

- 名称
- 描述
- 入口 URL
- 图标
- 所需角色
- 所需组织
- 当前用户是否可访问

同时展示用户已授权的应用，并支持撤销授权。具体能力以 Logto Account API 当前支持为准。

#### 公告通知

支持：

- 系统公告
- 迁移公告
- 维护公告
- 内测通知
- 已读 / 未读
- 重要公告置顶
- 按角色定向
- 按组织定向
- 按用户定向

#### 帮助支持

展示：

```text
contact@liteyuki.org
```

并提供：

- 登录异常帮助
- 账号迁移说明
- 常见问题
- 联系支持入口

### P1

#### 账户迁移助手

用于轻雪通行证迁移：

- 提示 Liteyuki Passport 停止服务时间。
- 提醒用户绑定其他登录方式。
- 展示迁移状态。
- 展示迁移帮助链接。

#### 权限说明页

用用户可理解的语言解释：

- 当前用户拥有哪些 roles
- 当前用户属于哪些 organizations
- 为什么无法访问某个应用
- 如何申请权限

#### 管理员后台

管理员后台只管理 Portal 自己的数据，不替代 Logto Console。

支持：

- 管理公告
- 管理应用目录
- 查看审计日志
- 管理帮助文档

管理员需要拥有：

```text
liteyuki-account-admin
```

## 推荐页面

```text
/
首页

/profile
个人资料

/security
安全中心

/sessions
会话管理

/applications
应用入口

/authorizations
应用授权

/organizations
组织与权限

/notifications
公告通知

/help
帮助支持

/logout
退出
```

## 后端 API 草案

### 当前用户

```text
GET /api/me
GET /api/me/permissions
GET /api/me/applications
```

### 会话

```text
GET /api/me/sessions
DELETE /api/me/sessions/:id
POST /api/me/logout-global
```

### 公告

```text
GET /api/announcements
POST /api/announcements/:id/read
```

### 应用目录

```text
GET /api/app-catalog
```

### 帮助信息

```text
GET /api/support-info
```

### 管理接口

```text
GET /api/admin/announcements
POST /api/admin/announcements
PATCH /api/admin/announcements/:id
DELETE /api/admin/announcements/:id

GET /api/admin/app-catalog
POST /api/admin/app-catalog
PATCH /api/admin/app-catalog/:id
DELETE /api/admin/app-catalog/:id

GET /api/admin/audit-logs
```

## 可选 PostgreSQL 数据表建议

以下数据表只在启用 PostgreSQL 模式时需要。P0 无数据库模式可以先跳过本节，通过静态配置和 Logto API 完成核心能力。

### portal_announcements

```text
id
title
content
severity
target_roles
target_organizations
target_users
pinned
starts_at
ends_at
created_at
updated_at
```

### portal_announcement_reads

```text
announcement_id
user_sub
read_at
```

### portal_applications

```text
id
name
description
url
icon
required_roles
required_organizations
sort_order
enabled
created_at
updated_at
```

### portal_user_preferences

```text
user_sub
locale
theme
dismissed_notice_keys
updated_at
```

### portal_audit_logs

```text
id
actor_sub
action
target_type
target_id
metadata
created_at
```

## 退出设计

提供两个退出入口。

### 退出当前门户

只清除 `account.liteyuki.org` 自己的 session。

### 退出 Yuki ID

清除当前门户 session 后跳转 Logto end session endpoint：

```text
https://auth.liteyuki.org/oidc/session/end
```

如果可以拿到 `id_token`，优先使用：

```text
https://auth.liteyuki.org/oidc/session/end?id_token_hint=<id_token>&post_logout_redirect_uri=https%3A%2F%2Faccount.liteyuki.org%2F
```

`post_logout_redirect_uri` 必须在 Logto 应用配置中登记。

## 用户标识

所有 Portal 自有数据使用 Logto `sub` 作为用户主键。

不要使用：

```text
email
username
name
```

作为唯一主键。

推荐 claims：

```text
sub
email
email_verified
name
preferred_username
picture
roles
organizations
organization_roles
```

## 权限模型

Portal 本身使用 Logto roles 控制权限。

示例：

```text
liteyuki-account-admin
liteyuki-account-user
```

应用访问控制可以使用：

```text
roles
organizations
organization_roles
```

例如：

```text
liteyuki-grafana-admin
liteyuki-grafana-user
liteyuki-harbor-admin
liteyuki-openlist-user
```

组织角色示例：

```text
<organization_id>:owner
<organization_id>:member
<organization_id>:beta-tester
```

## 验收标准

- 未登录访问门户会跳转到 Logto。
- 登录后能正确展示用户资料。
- 登录后能正确展示 roles、organizations 和 organization_roles。
- 用户资料修改不绕过 Logto Account API 权限配置。
- Management API token 不出现在浏览器、日志或接口响应中。
- 用户可以执行退出当前门户。
- 用户可以执行退出 Yuki ID 中心登录态。
- 公告可以按 role、organization、user 定向展示。
- 用户看不到自己无权访问的管理功能。
- 应用目录能根据用户权限展示可访问状态。
- 所有 redirect URI 和 post logout redirect URI 都使用白名单。
- 移动端可用。
- 错误状态友好，不展示原始堆栈。
- 敏感操作有确认流程。
- 管理操作有审计日志。

## 实现原则

- Logto 负责认证，不在 Portal 里重新实现密码登录。
- Portal 只做用户体验增强，不替代 Logto Console。
- 用户自助能力优先走 Account API。
- 管理能力必须走后端 BFF。
- 前端不直接编排 Logto Management API。
- Portal 数据和 Logto 用户数据不要双写。
- 第一版优先实现 P0，P1 后续迭代。

## AI 实现提示词

请实现 `Yuki ID Portal`，它是 `account.liteyuki.org` 的用户账号门户，使用 Logto 作为 OIDC Provider。

要求：

1. 使用 Logto 负责登录、OIDC、用户资料、角色、组织、MFA 和会话。
2. Portal 负责用户首页、资料整理、公告、应用入口、统一退出、权限说明和帮助支持。
3. 前端不得持有 Logto Management API token。
4. 用户自助资料能力优先使用 Logto Account API。
5. 需要 Management API 的能力必须通过后端 BFF 调用。
6. 用户唯一标识使用 Logto `sub`，不要用 email 当主键。
7. 支持 light / dark / system 主题。
8. 支持移动端。
9. 所有敏感操作需要确认。
10. 管理操作需要记录审计日志。
11. 先实现 P0，再实现 P1。
12. 参考 Logto 官方文档中的 Account API、Management API、Sessions、Sign-out、Organization experience。
