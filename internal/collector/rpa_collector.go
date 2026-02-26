package collector

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
	"github.com/datafusion/worker/internal/models"
)

// cookieEntry 保存的 Cookie 条目
type cookieEntry struct {
	cookies []*network.CookieParam
	savedAt time.Time
}

// RPACollector RPA 采集器（基于 Chromedp）
type RPACollector struct {
	headless bool
	timeout  time.Duration
	mu       sync.Mutex
	sessions map[string]*cookieEntry // key: URL host，如 "www.dxy.cn"
}

// NewRPACollector 创建 RPA 采集器
func NewRPACollector(headless bool, timeout int) *RPACollector {
	return &RPACollector{
		headless: headless,
		timeout:  time.Duration(timeout) * time.Second,
		sessions: make(map[string]*cookieEntry),
	}
}

// Type 返回采集器类型
func (r *RPACollector) Type() string {
	return "web-rpa"
}

// Collect 执行数据采集
func (r *RPACollector) Collect(ctx context.Context, config *models.DataSourceConfig) ([]map[string]interface{}, error) {
	log.Printf("开始 RPA 采集: %s", config.URL)

	// 创建 Chrome 上下文，模拟真实浏览器
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", r.headless),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-dev-shm-usage", true),
		chromedp.Flag("disable-blink-features", "AutomationControlled"),
		chromedp.UserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"),
	)

	allocCtx, cancel := chromedp.NewExecAllocator(ctx, opts...)
	defer cancel()

	chromeCtx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()

	// 设置总超时
	chromeCtx, cancel = context.WithTimeout(chromeCtx, r.timeout)
	defer cancel()

	var htmlContent string
	rpaConf := config.RPAConfig

	hasCookies := rpaConf != nil && (len(rpaConf.InitialCookies) > 0 || rpaConf.CookieString != "")

	if hasCookies {
		// ---- 手动 Cookie 注入流程（适用于短信验证码/扫码登录等无法自动模拟的场景）----
		log.Printf("使用手动配置的 Cookie 访问页面")
		initialParams := parseInitialCookies(rpaConf, config.URL)

		// 注入 Cookie → 导航 → 获取 HTML
		if err := chromedp.Run(chromeCtx,
			setCookiesAction(initialParams),
			navigateToDOMReady(config.URL),
			chromedp.OuterHTML("html", &htmlContent),
		); err != nil {
			return nil, fmt.Errorf("访问页面失败: %w", err)
		}

		// 检测会话有效性（若配置了顶层 check_selector）
		checkSel := rpaConf.CheckSelector
		if checkSel != "" && r.isSessionExpired(htmlContent, &models.RPALoginConfig{CheckSelector: checkSel}) {
			return nil, fmt.Errorf("Cookie 已失效，请从浏览器重新复制最新 Cookie 并更新数据源配置")
		}

		// 执行页面动作（若有）
		if len(rpaConf.Actions) > 0 {
			log.Printf("执行 %d 个页面动作", len(rpaConf.Actions))
			if err := chromedp.Run(chromeCtx, r.buildPageActions(rpaConf.Actions)...); err != nil {
				return nil, fmt.Errorf("执行页面动作失败: %w", err)
			}
			// 重新获取动作执行后的 HTML
			if err := chromedp.Run(chromeCtx, chromedp.OuterHTML("html", &htmlContent)); err != nil {
				return nil, fmt.Errorf("获取页面内容失败: %w", err)
			}
		}

	} else if rpaConf != nil && rpaConf.Login != nil {
		// ---- 有登录配置的流程（用户名/密码自动登录）----
		sessionKey := extractHostFromURL(config.URL)

		// Step 1: 尝试注入已保存的 Cookies，然后导航到目标页面
		var setupActions []chromedp.Action
		if params := r.loadCookies(sessionKey); len(params) > 0 {
			log.Printf("复用已保存的 Cookie（session_key=%s）", sessionKey)
			setupActions = append(setupActions, setCookiesAction(params))
		}
		setupActions = append(setupActions,
			navigateToDOMReady(config.URL),
			chromedp.OuterHTML("html", &htmlContent),
		)
		if err := chromedp.Run(chromeCtx, setupActions...); err != nil {
			return nil, fmt.Errorf("访问页面失败: %w", err)
		}

		// Step 2: 检测会话有效性，必要时执行登录
		if r.isSessionExpired(htmlContent, rpaConf.Login) {
			log.Printf("会话已过期或未登录，执行登录流程（session_key=%s）", sessionKey)
			loginActions := r.buildLoginActions(rpaConf.Login, config.URL)
			if err := chromedp.Run(chromeCtx, loginActions...); err != nil {
				return nil, fmt.Errorf("登录失败: %w", err)
			}
			// 保存新 Cookies
			r.captureAndSaveCookies(chromeCtx, sessionKey)
			// 重新导航到目标页面
			if err := chromedp.Run(chromeCtx, navigateToDOMReady(config.URL)); err != nil {
				return nil, fmt.Errorf("登录后访问目标页面失败: %w", err)
			}
		}

		// Step 3: 执行页面动作（搜索/筛选/点击等）
		if len(rpaConf.Actions) > 0 {
			log.Printf("执行 %d 个页面动作", len(rpaConf.Actions))
			pageActions := r.buildPageActions(rpaConf.Actions)
			if err := chromedp.Run(chromeCtx, pageActions...); err != nil {
				return nil, fmt.Errorf("执行页面动作失败: %w", err)
			}
		}

		// Step 4: 获取最终 HTML
		if err := chromedp.Run(chromeCtx, chromedp.OuterHTML("html", &htmlContent)); err != nil {
			return nil, fmt.Errorf("获取页面内容失败: %w", err)
		}
	} else {
		// ---- 无登录配置的原有流程 ----
		err := chromedp.Run(chromeCtx,
			navigateToDOMReady(config.URL),
			chromedp.OuterHTML("html", &htmlContent),
		)
		if err != nil {
			return nil, fmt.Errorf("访问页面失败: %w", err)
		}
	}

	log.Printf("页面加载成功，开始解析数据")
	return r.parseHTML(htmlContent, config.Selectors, config.URL)
}

// navigateToDOMReady 导航到指定 URL，只等待 DOMContentLoaded（不等所有资源）
// chromedp.Navigate() 默认等 Page.loadEventFired，对重型页面会超时
func navigateToDOMReady(rawURL string) chromedp.Action {
	return chromedp.ActionFunc(func(ctx context.Context) error {
		// 监听 DOMContentLoaded 事件
		domReady := make(chan struct{}, 1)
		lctx, cancelListen := context.WithCancel(ctx)
		defer cancelListen()

		chromedp.ListenTarget(lctx, func(ev interface{}) {
			if _, ok := ev.(*page.EventDomContentEventFired); ok {
				select {
				case domReady <- struct{}{}:
				default:
				}
			}
		})

		// 发起导航（不等待 load 事件，只等 DOMContentLoaded）
		_, _, _, err := page.Navigate(rawURL).Do(ctx)
		if err != nil {
			return fmt.Errorf("导航失败: %w", err)
		}

		// 等待 DOMContentLoaded 或超时
		select {
		case <-domReady:
			return nil
		case <-ctx.Done():
			// 超时时仍尝试继续，可能已有部分 HTML
			return nil
		case <-time.After(30 * time.Second):
			return nil // 30s 内没收到事件也继续
		}
	})
}

// isSessionExpired 检测会话是否已过期
// 若配置了 check_selector，则检查该元素是否存在；不存在则认为会话过期
// 若未配置 check_selector，返回 false（不判断，认为有效）
func (r *RPACollector) isSessionExpired(html string, login *models.RPALoginConfig) bool {
	if login.CheckSelector == "" {
		return false
	}
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return true // 解析失败时保守地触发重新登录
	}
	return doc.Find(login.CheckSelector).Length() == 0
}

// buildLoginActions 构建登录动作序列
func (r *RPACollector) buildLoginActions(login *models.RPALoginConfig, mainURL string) []chromedp.Action {
	loginURL := login.URL
	if loginURL == "" {
		loginURL = mainURL
	}

	actions := []chromedp.Action{
		navigateToDOMReady(loginURL),
		chromedp.WaitVisible(login.UsernameSelector, chromedp.ByQuery),
		chromedp.Clear(login.UsernameSelector, chromedp.ByQuery),
		chromedp.SendKeys(login.UsernameSelector, login.Username, chromedp.ByQuery),
		chromedp.WaitVisible(login.PasswordSelector, chromedp.ByQuery),
		chromedp.Clear(login.PasswordSelector, chromedp.ByQuery),
		chromedp.SendKeys(login.PasswordSelector, login.Password, chromedp.ByQuery),
		chromedp.Click(login.SubmitSelector, chromedp.ByQuery),
	}

	if login.WaitAfter != "" {
		actions = append(actions, chromedp.WaitVisible(login.WaitAfter, chromedp.ByQuery))
	} else {
		actions = append(actions, chromedp.Sleep(3*time.Second))
	}

	return actions
}

// buildPageActions 构建页面动作序列（搜索/筛选/点击等）
func (r *RPACollector) buildPageActions(actions []models.RPAPageAction) []chromedp.Action {
	var result []chromedp.Action
	for _, a := range actions {
		switch a.Type {
		case "input":
			result = append(result,
				chromedp.WaitVisible(a.Selector, chromedp.ByQuery),
				chromedp.Clear(a.Selector, chromedp.ByQuery),
				chromedp.SendKeys(a.Selector, a.Value, chromedp.ByQuery),
			)
		case "click":
			result = append(result,
				chromedp.WaitVisible(a.Selector, chromedp.ByQuery),
				chromedp.Click(a.Selector, chromedp.ByQuery),
			)
		case "select":
			result = append(result,
				chromedp.WaitVisible(a.Selector, chromedp.ByQuery),
				chromedp.SetValue(a.Selector, a.Value, chromedp.ByQuery),
			)
		case "wait":
			if a.WaitMs > 0 {
				result = append(result, chromedp.Sleep(time.Duration(a.WaitMs)*time.Millisecond))
			}
		}
		if a.WaitFor != "" {
			result = append(result, chromedp.WaitVisible(a.WaitFor, chromedp.ByQuery))
		}
	}
	return result
}

// captureAndSaveCookies 从当前 Chrome 上下文中抓取 Cookies 并保存
func (r *RPACollector) captureAndSaveCookies(ctx context.Context, key string) {
	var cookies []*network.Cookie
	err := chromedp.Run(ctx, chromedp.ActionFunc(func(ctx context.Context) error {
		var err error
		// GetCookies(nil) 返回当前浏览器上下文中所有 Cookie
		cookies, err = network.GetCookies().Do(ctx)
		return err
	}))
	if err != nil {
		log.Printf("保存 Cookie 失败: %v", err)
		return
	}
	params := cookiesToParams(cookies)
	r.saveCookies(key, params)
	log.Printf("已保存 %d 个 Cookie（session_key=%s）", len(params), key)
}

// saveCookies 保存 Cookie 到内存（mutex 保护）
func (r *RPACollector) saveCookies(key string, params []*network.CookieParam) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.sessions[key] = &cookieEntry{
		cookies: params,
		savedAt: time.Now(),
	}
}

// loadCookies 从内存加载 Cookie（有效期 24h）
func (r *RPACollector) loadCookies(key string) []*network.CookieParam {
	r.mu.Lock()
	defer r.mu.Unlock()
	entry, ok := r.sessions[key]
	if !ok {
		return nil
	}
	if time.Since(entry.savedAt) > 24*time.Hour {
		delete(r.sessions, key)
		return nil
	}
	return entry.cookies
}

// setCookiesAction 返回注入 Cookie 的 chromedp 动作
func setCookiesAction(params []*network.CookieParam) chromedp.Action {
	return chromedp.ActionFunc(func(ctx context.Context) error {
		return network.SetCookies(params).Do(ctx)
	})
}

// cookiesToParams 将 network.Cookie 转换为 network.CookieParam
func cookiesToParams(cookies []*network.Cookie) []*network.CookieParam {
	params := make([]*network.CookieParam, 0, len(cookies))
	for _, c := range cookies {
		params = append(params, &network.CookieParam{
			Name:     c.Name,
			Value:    c.Value,
			Domain:   c.Domain,
			Path:     c.Path,
			Secure:   c.Secure,
			HTTPOnly: c.HTTPOnly,
		})
	}
	return params
}

// extractHostFromURL 从 URL 中提取 host（用作会话 key）
func extractHostFromURL(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}
	return u.Host
}

// parseInitialCookies 解析手动配置的初始 Cookie，合并 InitialCookies 和 CookieString
func parseInitialCookies(rpaConf *models.RPAConfig, pageURL string) []*network.CookieParam {
	var params []*network.CookieParam
	host := extractHostFromURL(pageURL)
	domain := "." + host // 默认 domain

	// 处理结构化 InitialCookies
	for _, c := range rpaConf.InitialCookies {
		d := c.Domain
		if d == "" {
			d = domain
		}
		p := c.Path
		if p == "" {
			p = "/"
		}
		params = append(params, &network.CookieParam{
			Name:   c.Name,
			Value:  c.Value,
			Domain: d,
			Path:   p,
		})
	}

	// 处理 CookieString（格式: "name=val; name2=val2"）
	if rpaConf.CookieString != "" {
		for _, part := range strings.Split(rpaConf.CookieString, ";") {
			kv := strings.SplitN(strings.TrimSpace(part), "=", 2)
			if len(kv) == 2 {
				params = append(params, &network.CookieParam{
					Name:   strings.TrimSpace(kv[0]),
					Value:  strings.TrimSpace(kv[1]),
					Domain: domain,
					Path:   "/",
				})
			}
		}
	}
	return params
}

// parseHTML 解析 HTML 内容
func (r *RPACollector) parseHTML(html string, selectors map[string]string, pageURL string) ([]map[string]interface{}, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return nil, fmt.Errorf("解析 HTML 失败: %w", err)
	}

	var results []map[string]interface{}

	// 如果没有任何选择器，智能提取主要内容
	if len(selectors) == 0 {
		title := strings.TrimSpace(doc.Find("title").Text())
		content := r.extractMainContent(doc)
		item := map[string]interface{}{
			"title":   title,
			"content": content,
			"url":     pageURL,
		}
		results = append(results, item)
		log.Printf("无选择器配置，智能提取主要内容: 标题=%q, 内容长度=%d", title, len(content))
		return results, nil
	}

	// 假设有一个列表容器选择器
	listSelector, ok := selectors["_list"]
	if !ok {
		// 如果没有列表选择器，则提取单条数据
		item := make(map[string]interface{})
		item["url"] = pageURL
		for field, selector := range selectors {
			if field == "_list" {
				continue
			}
			el := doc.Find(selector).First()
			if field == "url" || field == "link" || field == "href" {
				if href, exists := el.Attr("href"); exists {
					item[field] = href
				} else {
					item[field] = strings.TrimSpace(el.Text())
				}
			} else {
				item[field] = strings.TrimSpace(el.Text())
			}
		}
		results = append(results, item)
		return results, nil
	}

	// 遍历列表项
	doc.Find(listSelector).Each(func(i int, s *goquery.Selection) {
		item := make(map[string]interface{})
		item["url"] = pageURL
		for field, selector := range selectors {
			if field == "_list" {
				continue
			}
			el := s.Find(selector)
			if field == "url" || field == "link" || field == "href" {
				if href, exists := el.First().Attr("href"); exists {
					item[field] = href
				} else {
					item[field] = strings.TrimSpace(el.Text())
				}
			} else {
				item[field] = strings.TrimSpace(el.Text())
			}
		}
		results = append(results, item)
	})

	log.Printf("解析完成，提取到 %d 条数据", len(results))
	return results, nil
}

// extractMainContent 智能提取页面主要正文内容，过滤噪音
func (r *RPACollector) extractMainContent(doc *goquery.Document) string {
	// 移除干扰元素
	doc.Find("script, style, nav, header, footer, iframe, noscript, .nav, .header, .footer, .sidebar, .advertisement, .ads, .menu").Remove()

	// 按优先级尝试找主内容区域
	mainSelectors := []string{
		"article",
		"[role=main]",
		"main",
		".rich_media_content", // 微信文章
		"#js_content",         // 微信文章
		".article-content",
		".post-content",
		".entry-content",
		".content",
		"#content",
		".main-content",
		"#main",
	}

	for _, sel := range mainSelectors {
		el := doc.Find(sel)
		if el.Length() > 0 {
			text := strings.TrimSpace(el.Text())
			if len(text) > 100 {
				return cleanText(text)
			}
		}
	}

	// 退而求其次：取 body 中最长的文本块（段落）
	var longestText string
	doc.Find("p, div").Each(func(_ int, s *goquery.Selection) {
		text := strings.TrimSpace(s.Text())
		if len(text) > len(longestText) {
			longestText = text
		}
	})
	if len(longestText) > 0 {
		return cleanText(longestText)
	}

	return cleanText(doc.Find("body").Text())
}

// cleanText 清理文本：合并连续空白字符
func cleanText(text string) string {
	text = strings.TrimSpace(text)
	for strings.Contains(text, "\n\n\n") {
		text = strings.ReplaceAll(text, "\n\n\n", "\n\n")
	}
	for strings.Contains(text, "  ") {
		text = strings.ReplaceAll(text, "  ", " ")
	}
	return text
}
