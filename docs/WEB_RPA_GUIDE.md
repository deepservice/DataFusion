# Web RPA 采集器配置指南

## 概述

Web RPA 采集器基于 Chromium（通过 chromedp）实现真实浏览器访问，支持：

- **普通采集**：自动提取正文 / CSS 选择器精确提取
- **登录支持**：自动模拟登录，保持会话
- **动态交互**：配置搜索、过滤、点击等页面动作后再采集

所有配置通过数据源的 `config` JSON 字段指定，无需修改代码。

---

## config 字段结构

```json
{
  "url": "https://example.com/page",
  "method": "GET",
  "headers": {},
  "selectors": {},
  "rpa_config": {
    "login": { ... },
    "actions": [ ... ]
  }
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| `url` | string | 目标页面 URL（必填） |
| `method` | string | HTTP 方法，默认 GET |
| `headers` | object | 自定义请求头（一般留空，浏览器自动处理） |
| `selectors` | object | CSS 选择器映射（详见下文），空则自动提取正文 |
| `rpa_config` | object | 高级配置，含登录和页面动作 |

---

## selectors 字段

`selectors` 控制从页面提取哪些字段，key 为存入数据库的字段名，value 为 CSS 选择器。

### 自动提取模式（推荐默认）

不配置 `selectors` 或配置为空 `{}`，系统自动识别并提取页面主要正文，存储为 `{title, content, url}` 三个字段。

适合：普通文章页面、博客、新闻。

```json
{
  "url": "https://mp.weixin.qq.com/s/xxxxxx"
}
```

### 精确提取模式

配置具体选择器，精确提取页面中指定元素的文本内容。

```json
{
  "url": "https://example.com/news",
  "selectors": {
    "title":   "h1.article-title",
    "content": "#article-body",
    "author":  ".author-name",
    "date":    "time.publish-date"
  }
}
```

### 列表批量采集

使用特殊的 `_list` key 指定列表容器，系统遍历容器内的每个元素并提取。

```json
{
  "url": "https://example.com/list",
  "selectors": {
    "_list":   ".post-item",
    "title":   ".post-title",
    "summary": ".post-summary",
    "link":    "a.post-link"
  }
}
```

- `_list` 的值是列表容器的选择器
- 其他字段在每个列表项内部用相对选择器查找
- 字段名为 `url`/`link`/`href` 时，优先提取 `href` 属性值

> **提示**：使用"预览页面结构"功能可以看到页面中可用的 CSS 选择器列表，直接复制使用。

---

## rpa_config.login — 登录配置

对需要登录才能访问的页面，配置登录信息：

```json
{
  "url": "https://www.dxy.cn/board/articles",
  "rpa_config": {
    "login": {
      "url":               "https://www.dxy.cn/login",
      "username_selector": "#username",
      "password_selector": "#password",
      "submit_selector":   "button[type='submit']",
      "username":          "your-username",
      "password":          "your-password",
      "wait_after":        ".nav-user-avatar",
      "check_selector":    ".nav-user-avatar"
    }
  }
}
```

### login 字段说明

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `url` | string | 否 | 登录页 URL；留空则在主 URL 上操作 |
| `username_selector` | string | 是 | 用户名/邮箱输入框的 CSS 选择器 |
| `password_selector` | string | 是 | 密码输入框的 CSS 选择器 |
| `submit_selector` | string | 是 | 登录按钮的 CSS 选择器 |
| `username` | string | 是 | 账号（用户名或邮箱） |
| `password` | string | 是 | 密码 |
| `wait_after` | string | 否 | 登录成功后等待出现的元素选择器（用于确认登录完成） |
| `check_selector` | string | 否 | 每次采集前检查会话有效性的元素；该元素不在页面上则自动重新登录 |

### 会话保持机制

1. 首次采集：执行登录流程，将浏览器 Cookie 保存到 Worker 内存
2. 后续采集：注入已保存的 Cookie，跳过登录流程
3. 会话检测：若 `check_selector` 指定的元素不存在，认为 Cookie 已失效，重新登录
4. 有效期：Cookie 最长保存 24 小时；Worker 重启后自动重新登录

### 会话 key

会话以目标 URL 的 host（如 `www.dxy.cn`）为 key，同一域名下的多个数据源共享 Cookie。

---

## rpa_config — Cookie 注入（短信/扫码验证等场景）

对于使用短信验证码、扫码登录等**无法自动模拟**的登录方式，可以从浏览器手动复制已登录的 Cookie，直接配置到数据源中，Worker 会在每次采集前注入这些 Cookie。

### 使用方法

1. 在浏览器中**手动登录**目标网站
2. 打开 DevTools（F12）→ Network → 点击任意页面请求 → Headers → Request Headers → 找到 **Cookie** 行
3. 复制 Cookie 的完整值（格式：`name=val; name2=val2; ...`）
4. 填入数据源 config 的 `rpa_config.cookie_string` 字段

### 配置示例

```json
{
  "url": "https://www.dxy.cn/board/articles",
  "selectors": {
    "_list": ".article-item",
    "title": ".article-title",
    "content": ".article-summary"
  },
  "rpa_config": {
    "cookie_string": "session_id=xxx; token=yyy; user_id=123",
    "check_selector": ".nav-user-avatar"
  }
}
```

也可以使用结构化的 `initial_cookies` 列表（适合需要精确指定 Domain 的场景）：

```json
{
  "url": "https://www.dxy.cn/board/articles",
  "rpa_config": {
    "initial_cookies": [
      {"name": "session_id", "value": "xxx", "domain": ".dxy.cn"},
      {"name": "token",      "value": "yyy", "domain": ".dxy.cn"}
    ],
    "check_selector": ".nav-user-avatar"
  }
}
```

### Cookie 注入字段说明

| 字段 | 类型 | 说明 |
|------|------|------|
| `cookie_string` | string | 从浏览器复制的 Cookie 字符串（`name=val; name2=val2`），优先推荐此方式 |
| `initial_cookies` | array | 结构化 Cookie 列表，每项含 `name`、`value`、`domain`（可选）、`path`（可选）|
| `check_selector` | string | Cookie 有效时页面上存在的元素选择器；若该元素不存在，任务会立即报错并提示重新配置 Cookie |

### 与账号密码登录的区别

| | `cookie_string` / `initial_cookies` | `login` |
|--|--|--|
| 适用场景 | 短信验证码、扫码、第三方登录等 | 标准用户名/密码登录表单 |
| Cookie 来源 | 用户手动复制 | 系统自动采集 |
| Cookie 失效后 | 任务报错，需手动更新 | 自动重新登录 |
| 配置复杂度 | 简单 | 需要找到登录表单的 CSS 选择器 |

> **提示**：`cookie_string` 和 `login` 可以不同时配置。若同时配置，优先使用 `cookie_string`（直接注入，不执行登录流程）。

---

## rpa_config.actions — 页面动作序列

在导航到目标页面后、提取内容前，执行一系列页面交互动作。适用于需要搜索、筛选、点击后才能获得目标数据的场景。

```json
{
  "url": "https://example.com/search",
  "rpa_config": {
    "actions": [
      {"type": "input",  "selector": "#keyword-input", "value": "心脏病"},
      {"type": "click",  "selector": "#search-button", "wait_for": ".result-list"},
      {"type": "select", "selector": "#sort-select",   "value": "latest"},
      {"type": "wait",   "wait_ms": 1000}
    ]
  }
}
```

### actions 字段说明

每个动作是一个对象，包含：

| 字段 | 类型 | 说明 |
|------|------|------|
| `type` | string | 动作类型：`input` / `click` / `select` / `wait` |
| `selector` | string | 目标元素的 CSS 选择器（`wait` 类型无需此字段） |
| `value` | string | 输入值（`input`/`select` 类型使用） |
| `wait_for` | string | 动作完成后等待出现的元素选择器（可选，适用于等待页面响应） |
| `wait_ms` | int | 等待毫秒数（`wait` 类型专用） |

### 动作类型详解

**`input` — 输入文本**

等待元素出现 → 清空内容 → 输入文本

```json
{"type": "input", "selector": "#search-input", "value": "关键词"}
```

**`click` — 点击元素**

等待元素出现 → 点击

```json
{"type": "click", "selector": ".submit-btn", "wait_for": ".result-container"}
```

`wait_for` 会在点击后等待该元素出现，适合等待搜索结果加载。

**`select` — 选择下拉选项**

等待元素出现 → 设置选中值

```json
{"type": "select", "selector": "#category", "value": "technology"}
```

**`wait` — 等待指定时间**

```json
{"type": "wait", "wait_ms": 2000}
```

---

## 完整配置示例

### 示例1：丁香园文章列表（需登录 + 搜索）

```json
{
  "url": "https://www.dxy.cn/board/articles",
  "selectors": {
    "_list": ".article-item",
    "title": ".article-title",
    "content": ".article-summary",
    "author": ".author-name"
  },
  "rpa_config": {
    "login": {
      "url": "https://www.dxy.cn/login",
      "username_selector": "#username",
      "password_selector": "#password",
      "submit_selector": "button[type='submit']",
      "username": "your-username",
      "password": "your-password",
      "wait_after": ".user-info",
      "check_selector": ".user-info"
    },
    "actions": [
      {"type": "input",  "selector": ".search-input", "value": "心脏病"},
      {"type": "click",  "selector": ".search-submit", "wait_for": ".article-list"},
      {"type": "select", "selector": ".sort-select", "value": "newest"}
    ]
  }
}
```

### 示例2：微信公众号文章（无需登录）

```json
{
  "url": "https://mp.weixin.qq.com/s/xxxxxx"
}
```

无选择器配置，自动提取标题和正文，存储 `title`、`content`、`url` 三个字段。

### 示例3：需要搜索但无需登录

```json
{
  "url": "https://example.com/forum",
  "selectors": {
    "_list": ".post-row",
    "title": ".post-title a",
    "views": ".view-count",
    "replies": ".reply-count"
  },
  "rpa_config": {
    "actions": [
      {"type": "click",  "selector": "#filter-hot", "wait_for": ".post-list"},
      {"type": "wait",   "wait_ms": 500}
    ]
  }
}
```

---

## 常见问题

### 登录后仍被跳转回登录页

- 检查 `submit_selector` 是否正确（确保点击的是登录按钮，而非取消按钮）
- 增加 `wait_after` 字段，等待登录成功后的标志性元素出现
- 检查账号密码是否正确

### 会话频繁失效

- 检查 `check_selector` 是否指向正确的用户状态标志（如用户头像、用户名显示区域）
- 部分网站会在跳转后改变选择器结构，需要检查实际页面

### 动作执行超时

- 检查 `selector` 是否正确，可使用浏览器开发者工具验证
- 在动作后加 `wait_for` 或 `wait` 给页面响应留出时间
- 如页面加载慢，考虑在数据源关联的任务中增加 `execution_timeout`

### 页面内容为空或不正确

- 检查 CSS 选择器是否正确，使用"预览页面结构"工具辅助
- 若页面是动态渲染的（SPA），需要等待内容加载完成，可在动作中加 `wait` 或 `wait_for`
- 对于列表页面，确认 `_list` 选择器是否覆盖了所有列表项
