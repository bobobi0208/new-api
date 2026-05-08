package service

import (
	"errors"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/bytedance/gopkg/util/gopool"
	"gorm.io/gorm"
)

type SensitiveMonitorInput struct {
	UserId     int
	Username   string
	TokenId    int
	TokenName  string
	ModelName  string
	RequestId  string
	Ip         string
	ChannelId  int
	Group      string
	Path       string
	PromptText string
}

type SensitiveMonitorRuleHit struct {
	RuleId  int    `json:"rule_id"`
	Pattern string `json:"pattern"`
	Action  int    `json:"action"`
	IsRegex bool   `json:"is_regex"`
}

type SensitiveMonitorResult struct {
	Matched bool                      `json:"matched"`
	Blocked bool                      `json:"blocked"`
	Hits    []SensitiveMonitorRuleHit `json:"hits"`
}

type SensitiveWordsSeedResult struct {
	Created int `json:"created"`
	Updated int `json:"updated"`
}

type cachedSensitiveRule struct {
	id           int
	pattern      string
	matchPattern string
	isRegex      bool
	action       int
	regex        *regexp.Regexp
}

var sensitiveMonitorCache = struct {
	sync.RWMutex
	loaded bool
	rules  []cachedSensitiveRule
}{}

var sensitiveMonitorPathPrefixes = []string{
	"/pg/chat/completions",
	"/v1/chat/completions",
	"/v1/messages",
	"/v1/responses",
	"/v1/completions",
	"/v1/embeddings",
	"/v1/moderations",
	"/v1/rerank",
	"/v1beta/models/",
}

func resetSensitiveMonitorCacheForTest() {
	sensitiveMonitorCache.Lock()
	defer sensitiveMonitorCache.Unlock()
	sensitiveMonitorCache.loaded = false
	sensitiveMonitorCache.rules = nil
}

func ReloadSensitiveMonitorRules() error {
	rules, err := loadSensitiveMonitorRules()
	if err != nil {
		return err
	}
	sensitiveMonitorCache.Lock()
	sensitiveMonitorCache.loaded = true
	sensitiveMonitorCache.rules = rules
	sensitiveMonitorCache.Unlock()
	return nil
}

func loadSensitiveMonitorRules() ([]cachedSensitiveRule, error) {
	var rules []model.SensitiveWord
	if err := model.DB.Where("enabled = ?", true).Order("id asc").Find(&rules).Error; err != nil {
		return nil, err
	}
	cached := make([]cachedSensitiveRule, 0, len(rules))
	for _, rule := range rules {
		pattern := strings.TrimSpace(rule.Pattern)
		if pattern == "" {
			continue
		}
		item := cachedSensitiveRule{
			id:           rule.Id,
			pattern:      pattern,
			matchPattern: strings.ToLower(pattern),
			isRegex:      rule.IsRegex,
			action:       rule.Action,
		}
		if item.action == 0 {
			item.action = model.SensitiveWordActionMonitor
		}
		if rule.IsRegex {
			compiled, err := regexp.Compile(pattern)
			if err != nil {
				common.SysLog("sensitive monitor regex skipped: " + err.Error())
				continue
			}
			item.regex = compiled
		}
		cached = append(cached, item)
	}
	return cached, nil
}

func getSensitiveMonitorRules() ([]cachedSensitiveRule, error) {
	sensitiveMonitorCache.RLock()
	if sensitiveMonitorCache.loaded {
		rules := sensitiveMonitorCache.rules
		sensitiveMonitorCache.RUnlock()
		return rules, nil
	}
	sensitiveMonitorCache.RUnlock()

	if err := ReloadSensitiveMonitorRules(); err != nil {
		return nil, err
	}
	sensitiveMonitorCache.RLock()
	defer sensitiveMonitorCache.RUnlock()
	return sensitiveMonitorCache.rules, nil
}

func ShouldRunSensitiveMonitor(path string) bool {
	if path == "" {
		return true
	}
	for _, prefix := range sensitiveMonitorPathPrefixes {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}
	return false
}

func ScheduleSensitiveMonitor(input SensitiveMonitorInput) {
	if strings.TrimSpace(input.PromptText) == "" || !ShouldRunSensitiveMonitor(input.Path) {
		return
	}
	prompt := input.PromptText
	gopool.Go(func() {
		input.PromptText = prompt
		if _, err := CheckSensitiveMonitor(input); err != nil {
			common.SysLog("sensitive monitor failed: " + err.Error())
		}
	})
}

func CheckSensitiveMonitor(input SensitiveMonitorInput) (SensitiveMonitorResult, error) {
	result := SensitiveMonitorResult{}
	if strings.TrimSpace(input.PromptText) == "" {
		return result, nil
	}
	rules, err := getSensitiveMonitorRules()
	if err != nil {
		return result, err
	}
	if len(rules) == 0 {
		return result, nil
	}

	checkText := strings.ToLower(input.PromptText)
	now := time.Now()
	for _, rule := range rules {
		matchStart, matchEnd, matched := rule.matchRange(input.PromptText, checkText)
		if !matched {
			continue
		}
		result.Matched = true
		if rule.action == model.SensitiveWordActionBlock {
			result.Blocked = true
		}
		result.Hits = append(result.Hits, SensitiveMonitorRuleHit{
			RuleId:  rule.id,
			Pattern: rule.pattern,
			Action:  rule.action,
			IsRegex: rule.isRegex,
		})
		hit := model.SensitiveWordHit{
			RuleId:        rule.id,
			Pattern:       rule.pattern,
			IsRegex:       rule.isRegex,
			Action:        rule.action,
			UserId:        input.UserId,
			Username:      input.Username,
			TokenId:       input.TokenId,
			TokenName:     input.TokenName,
			ModelName:     input.ModelName,
			RequestId:     input.RequestId,
			Ip:            input.Ip,
			ChannelId:     input.ChannelId,
			Group:         input.Group,
			Path:          input.Path,
			PromptSnippet: sensitivePromptSnippet(input.PromptText, matchStart, matchEnd),
			CreatedAt:     now,
		}
		if err := model.DB.Create(&hit).Error; err != nil {
			return result, err
		}
		if err := model.IncrementSensitiveHit(rule.id, now); err != nil {
			return result, err
		}
	}
	if result.Blocked {
		if err := model.DisableTokenForSensitiveHit(input.TokenId); err != nil {
			return result, err
		}
	}
	return result, nil
}

func (r cachedSensitiveRule) matches(rawText, lowerText string) bool {
	_, _, matched := r.matchRange(rawText, lowerText)
	return matched
}

func (r cachedSensitiveRule) matchRange(rawText, lowerText string) (int, int, bool) {
	if r.isRegex {
		if r.regex == nil {
			return 0, 0, false
		}
		match := r.regex.FindStringIndex(rawText)
		if len(match) != 2 {
			return 0, 0, false
		}
		return match[0], match[1], true
	}
	start := strings.Index(lowerText, r.matchPattern)
	if start < 0 {
		return 0, 0, false
	}
	return start, start + len(r.matchPattern), true
}

func sensitivePromptSnippet(text string, matchStart, matchEnd int) string {
	const contextRunes = 12
	if matchStart < 0 || matchEnd < matchStart || matchStart > len(text) {
		return ""
	}
	if matchEnd > len(text) {
		matchEnd = len(text)
	}

	prefix := []rune(text[:matchStart])
	matched := []rune(text[matchStart:matchEnd])
	suffix := []rune(text[matchEnd:])

	if len(prefix) > contextRunes {
		prefix = prefix[len(prefix)-contextRunes:]
	}
	if len(suffix) > contextRunes {
		suffix = suffix[:contextRunes]
	}

	return string(prefix) + string(matched) + string(suffix)
}

func SeedDefaultSensitiveWords() (SensitiveWordsSeedResult, error) {
	defaults := []model.SensitiveWord{
		{Pattern: `儿童色情`, Enabled: true, Action: model.SensitiveWordActionMonitor, Description: `【中文·直接】明示请求未成年人不适当影像内容（CSAM 类，高严重度）`},
		{Pattern: `儿童裸照`, Enabled: true, Action: model.SensitiveWordActionMonitor, Description: `【中文·直接】明示请求未成年人裸露身体影像`},
		{Pattern: `儿童裸体照`, Enabled: true, Action: model.SensitiveWordActionMonitor, Description: `【中文·直接】同义变体：未成年人裸露身体影像`},
		{Pattern: `幼女色情`, Enabled: true, Action: model.SensitiveWordActionMonitor, Description: `【中文·直接】针对低龄女性的不适当影像内容`},
		{Pattern: `幼女裸照`, Enabled: true, Action: model.SensitiveWordActionMonitor, Description: `【中文·直接】针对低龄女性的裸露身体影像`},
		{Pattern: `幼女黄图`, Enabled: true, Action: model.SensitiveWordActionMonitor, Description: `【中文·俚语】"黄图"＝不适当图片，针对低龄女性`},
		{Pattern: `幼童色情`, Enabled: true, Action: model.SensitiveWordActionMonitor, Description: `【中文·直接】针对幼龄未成年人的不适当影像内容`},
		{Pattern: `幼童裸照`, Enabled: true, Action: model.SensitiveWordActionBlock, Description: `【中文·直接】针对幼龄未成年人的裸露身体影像`},
		{Pattern: `幼齿色情`, Enabled: true, Action: model.SensitiveWordActionBlock, Description: `【中文·俚语】"幼齿"＝低龄表达，配合不适当内容语境`},
		{Pattern: `幼齿萝莉`, Enabled: true, Action: model.SensitiveWordActionBlock, Description: `【中文·俚语】低龄表达 + 动漫圈代称的组合`},
		{Pattern: `嫩萝莉色情`, Enabled: true, Action: model.SensitiveWordActionBlock, Description: `【中文·俚语】"嫩"＝低龄强调，配合不适当内容语境`},
		{Pattern: `萝莉色情`, Enabled: true, Action: model.SensitiveWordActionBlock, Description: `【ACG 圈·俚语】动漫圈代指未成年女性角色，配合不适当内容语境`},
		{Pattern: `萝莉黄图`, Enabled: true, Action: model.SensitiveWordActionBlock, Description: `【ACG 圈·俚语】动漫圈代称 + 不适当图片表达`},
		{Pattern: `萝莉裸照`, Enabled: true, Action: model.SensitiveWordActionBlock, Description: `【ACG 圈·俚语】动漫圈代称 + 裸露影像`},
		{Pattern: `萝莉本子`, Enabled: true, Action: model.SensitiveWordActionBlock, Description: `【ACG 圈·俚语】"本子"指同人不适当漫画，明确指向未成年角色`},
		{Pattern: `未成年色情`, Enabled: true, Action: model.SensitiveWordActionBlock, Description: `【中文·直接】针对未成年人的不适当内容请求`},
		{Pattern: `未成年裸照`, Enabled: true, Action: model.SensitiveWordActionBlock, Description: `【中文·直接】针对未成年人的裸露身体影像`},
		{Pattern: `未成年性交`, Enabled: true, Action: model.SensitiveWordActionBlock, Description: `【中文·直接】涉及未成年人的性行为表述`},
		{Pattern: `小学生援交`, Enabled: true, Action: model.SensitiveWordActionBlock, Description: `【未成年商业性剥削】"援交"＝有偿性交易；小学生为未成年群体`},
		{Pattern: `初中生援交`, Enabled: true, Action: model.SensitiveWordActionBlock, Description: `【未成年商业性剥削】有偿性交易类表述（初中生群体）`},
		{Pattern: `小学生卖淫`, Enabled: true, Action: model.SensitiveWordActionBlock, Description: `【未成年商业性剥削】未成年人被卷入有偿性交易`},
		{Pattern: `小学生嫖娼`, Enabled: true, Action: model.SensitiveWordActionBlock, Description: `【未成年商业性剥削】未成年人作为有偿性交易对象`},
		{Pattern: `诱骗儿童`, Enabled: true, Action: model.SensitiveWordActionBlock, Description: `【Grooming】对未成年人实施心理诱导（常见性侵预备行为）`},
		{Pattern: `诱拐儿童`, Enabled: true, Action: model.SensitiveWordActionBlock, Description: `【人身侵害】对未成年人实施引诱并非法转移`},
		{Pattern: `诱拐幼女`, Enabled: true, Action: model.SensitiveWordActionBlock, Description: `【人身侵害】针对低龄女性的引诱与非法转移`},
		{Pattern: `诱奸幼女`, Enabled: true, Action: model.SensitiveWordActionBlock, Description: `【性侵 + 诱骗】通过心理诱导对低龄女性实施性侵害`},
		{Pattern: `拐卖儿童`, Enabled: true, Action: model.SensitiveWordActionBlock, Description: `【人口贩卖】将未成年人作为商品进行非法转移交易`},
		{Pattern: `拐卖幼女`, Enabled: true, Action: model.SensitiveWordActionBlock, Description: `【人口贩卖】将低龄女性作为商品进行非法转移交易`},
		{Pattern: `性侵儿童`, Enabled: true, Action: model.SensitiveWordActionMonitor, Description: `【性侵·★高误伤】反性侵科普/法律语境会触发；默认改为监控模式观察`},
		{Pattern: `性侵幼女`, Enabled: true, Action: model.SensitiveWordActionMonitor, Description: `【性侵·★高误伤】同上；默认监控模式`},
		{Pattern: `性侵幼童`, Enabled: true, Action: model.SensitiveWordActionBlock, Description: `【性侵】对幼龄未成年人的性相关侵害行为`},
		{Pattern: `性侵小学生`, Enabled: true, Action: model.SensitiveWordActionBlock, Description: `【性侵】对小学生群体的性相关侵害行为`},
		{Pattern: `猥亵儿童`, Enabled: true, Action: model.SensitiveWordActionBlock, Description: `【猥亵】对未成年人实施不当身体接触（法律术语）`},
		{Pattern: `猥亵幼女`, Enabled: true, Action: model.SensitiveWordActionBlock, Description: `【猥亵】针对低龄女性的不当身体接触`},
		{Pattern: `猥亵幼童`, Enabled: true, Action: model.SensitiveWordActionBlock, Description: `【猥亵】针对幼龄未成年人的不当身体接触`},
		{Pattern: `强奸幼女`, Enabled: true, Action: model.SensitiveWordActionBlock, Description: `【性侵】对低龄女性的强制性侵害行为`},
		{Pattern: `强奸幼童`, Enabled: true, Action: model.SensitiveWordActionBlock, Description: `【性侵】对幼龄未成年人的强制性侵害行为`},
		{Pattern: `强奸小学生`, Enabled: true, Action: model.SensitiveWordActionBlock, Description: `【性侵】对小学生群体的强制性侵害行为`},
		{Pattern: `轮奸幼女`, Enabled: true, Action: model.SensitiveWordActionBlock, Description: `【性侵】对低龄女性的团伙性侵害行为`},
		{Pattern: `虐杀儿童`, Enabled: true, Action: model.SensitiveWordActionBlock, Description: `【暴力】针对未成年人的暴力杀害类表述`},
		{Pattern: `虐杀幼童`, Enabled: true, Action: model.SensitiveWordActionBlock, Description: `【暴力】针对幼龄未成年人的暴力杀害类表述`},
		{Pattern: `偷拍幼女`, Enabled: true, Action: model.SensitiveWordActionBlock, Description: `【隐私侵害】未经同意拍摄低龄女性`},
		{Pattern: `偷拍小学生`, Enabled: true, Action: model.SensitiveWordActionBlock, Description: `【隐私侵害】未经同意拍摄小学生群体`},
		{Pattern: `偷拍儿童裙底`, Enabled: true, Action: model.SensitiveWordActionBlock, Description: `【隐私侵害·性骚扰】未经同意拍摄未成年人私密部位`},
		{Pattern: `恋童癖`, Enabled: true, Action: model.SensitiveWordActionMonitor, Description: `【临床/法律词汇·★高误伤】常见于科普与心理学语境；默认监控模式`},
		{Pattern: `child porn`, Enabled: true, Action: model.SensitiveWordActionMonitor, Description: `【英文·直接】英文最直接表达，等同 CSAM`},
		{Pattern: `child pornography`, Enabled: true, Action: model.SensitiveWordActionBlock, Description: `【英文·学术】学术与法律完整写法`},
		{Pattern: `childporn`, Enabled: true, Action: model.SensitiveWordActionMonitor, Description: `【英文·绕过变体】无空格紧凑写法`},
		{Pattern: `underage porn`, Enabled: true, Action: model.SensitiveWordActionBlock, Description: `【英文·直接】未成年相关不适当内容`},
		{Pattern: `underage nude`, Enabled: true, Action: model.SensitiveWordActionBlock, Description: `【英文·直接】未成年人裸露身体表达`},
		{Pattern: `underage sex`, Enabled: true, Action: model.SensitiveWordActionBlock, Description: `【英文·直接】未成年人性行为表达`},
		{Pattern: `minor pornography`, Enabled: true, Action: model.SensitiveWordActionBlock, Description: `【英文·直接】未成年不适当内容（学术写法）`},
		{Pattern: `minor sexual abuse`, Enabled: true, Action: model.SensitiveWordActionBlock, Description: `【英文·直接】针对未成年人的性相关侵害`},
		{Pattern: `preteen porn`, Enabled: true, Action: model.SensitiveWordActionBlock, Description: `【英文·直接】青春期前（约 12 岁以下）相关不适当内容`},
		{Pattern: `preteen nude`, Enabled: true, Action: model.SensitiveWordActionBlock, Description: `【英文·直接】青春期前年龄段裸露表达`},
		{Pattern: `preteen sex`, Enabled: true, Action: model.SensitiveWordActionBlock, Description: `【英文·直接】青春期前年龄段性行为表达`},
		{Pattern: `preteen erotica`, Enabled: true, Action: model.SensitiveWordActionBlock, Description: `【英文·直接】青春期前年龄段不适当艺术内容`},
		{Pattern: `kiddie porn`, Enabled: true, Action: model.SensitiveWordActionBlock, Description: `【英文·俚语】kiddie＝小孩，针对未成年的英文俚语`},
		{Pattern: `lolicon porn`, Enabled: true, Action: model.SensitiveWordActionBlock, Description: `【英文·ACG】动漫圈用语，涉及低龄女性角色的不适当内容`},
		{Pattern: `lolicon hentai`, Enabled: true, Action: model.SensitiveWordActionMonitor, Description: `【英文·ACG】涉及低龄女性角色的不适当漫画`},
		{Pattern: `shotacon hentai`, Enabled: true, Action: model.SensitiveWordActionBlock, Description: `【英文·ACG】涉及低龄男性角色的不适当漫画`},
		{Pattern: `jailbait`, Enabled: true, Action: model.SensitiveWordActionBlock, Description: `【英文·俚语】专指未成年女性的诱拍/色诱影像`},
		{Pattern: `groom a child`, Enabled: true, Action: model.SensitiveWordActionBlock, Description: `【英文·Grooming】对一名未成年人实施心理诱导`},
		{Pattern: `groom children`, Enabled: true, Action: model.SensitiveWordActionBlock, Description: `【英文·Grooming】对多名未成年人实施心理诱导`},
		{Pattern: `grooming minors`, Enabled: true, Action: model.SensitiveWordActionBlock, Description: `【英文·Grooming】对未成年群体实施心理诱导`},
		{Pattern: `molest a child`, Enabled: true, Action: model.SensitiveWordActionBlock, Description: `【英文·猥亵】对一名未成年人实施不当接触`},
		{Pattern: `molest children`, Enabled: true, Action: model.SensitiveWordActionBlock, Description: `【英文·猥亵】对多名未成年人实施不当接触`},
		{Pattern: `how to molest`, Enabled: true, Action: model.SensitiveWordActionBlock, Description: `【教唆指令】请求"如何"实施不当接触行为`},
		{Pattern: `how to groom child`, Enabled: true, Action: model.SensitiveWordActionBlock, Description: `【教唆指令】请求"如何"对未成年人实施心理诱导`},
		{Pattern: `how to abduct a child`, Enabled: true, Action: model.SensitiveWordActionBlock, Description: `【教唆指令】请求"如何"对未成年人实施非法转移`},
		{Pattern: `child sex tourism`, Enabled: true, Action: model.SensitiveWordActionBlock, Description: `【英文·跨国犯罪】前往他国对未成年人实施性侵害`},
		{Pattern: `child trafficking`, Enabled: true, Action: model.SensitiveWordActionMonitor, Description: `【英文·人口贩卖】将未成年人作为商品进行非法转移交易`},
		{Pattern: `child sexual abuse material`, Enabled: true, Action: model.SensitiveWordActionMonitor, Description: `【英文·CSAM 全称】Child Sexual Abuse Material 完整写法`},
		{Pattern: `(?i)(child|children|kid|kids|minor|underage|pre[\s_-]?teen)\s*(porn|nude|naked|sex|erotic|exploit|abuse\s*image)`, Enabled: true, Action: model.SensitiveWordActionMonitor, Description: `【英文正则·组合】未成年人类词 + 不适当内容词（含 preteen 拼写变体）`},
		{Pattern: `(?i)(儿童|幼女|幼童|小学生|未成年).{0,10}(色情|裸照|裸体|性交|嫖娼|嫖客|卖淫|援交|做爱|强暴)`, IsRegex: true, Enabled: true, Action: model.SensitiveWordActionMonitor, Description: `【中文正则·组合】未成年人类词 与 不适当内容词 在 10 字内同时出现`},
		{Pattern: `(?i)(强|轮)\s*奸(\s*(儿童|幼女|幼童|小学生|未成年))?`, IsRegex: true, Enabled: true, Action: model.SensitiveWordActionMonitor, Description: `【中文正则·组合】强制性侵害动词，可选未成年人类词（容忍中间空格）`},
		{Pattern: `(?i)(诱骗|诱拐|拐卖|拐走|拐带)\s*(儿童|幼女|幼童|小学生|初中生|未成年)`, IsRegex: true, Enabled: true, Action: model.SensitiveWordActionMonitor, Description: `【中文正则·组合】非法转移类动词 + 未成年人类词`},
		{Pattern: `(?i)(虐待|虐杀|残害|杀害)\s*(儿童|幼女|幼童|婴儿|新生儿)`, IsRegex: true, Enabled: true, Action: model.SensitiveWordActionMonitor, Description: `【中文正则·组合】伤害类动词 + 婴幼儿类词`},
		{Pattern: `(?i)(loli|shota)[\s_-]?(con|porn|nude|hentai|art|drawing|animation)`, IsRegex: true, Enabled: true, Action: model.SensitiveWordActionMonitor, Description: `【英文正则·ACG】动漫圈低龄角色代称 + 内容类词，容忍空格/下划线/连字符`},
		{Pattern: `(?i)pedo(phile|philia|philic)?\s*(content|material|porn|images?|videos?)`, IsRegex: true, Enabled: true, Action: model.SensitiveWordActionMonitor, Description: `【英文正则·恋童】恋童词根 + 内容类词上下文（避免 pedo 单词在地理学等场景误伤）`},
		{Pattern: `(?i)(8|9|10|11|12|13|14|15)\s*(yo|y\.o\.?|year[\s_-]?old|岁)\s*(girl|boy|child|kid|的|女孩|男孩).{0,20}(porn|nude|naked|sex|裸|色情|做爱)`, IsRegex: true, Enabled: true, Action: model.SensitiveWordActionMonitor, Description: `【混合正则·年龄】数字年龄 + 性别子词 + 不适当内容词（覆盖中英写法）`},
		{Pattern: `(?i)(rape|sexually\s*abuse|sexually\s*assault)\s*(child|children|kid|minor|underage|toddler|infant)`, IsRegex: true, Enabled: true, Action: model.SensitiveWordActionMonitor, Description: `【英文正则·组合】性侵害类动词 + 未成年类词（含 toddler/infant）`},
		{Pattern: `(?i)(traffic|kidnap|abduct|sell)\s*(child|children|kid|minor|toddler)`, IsRegex: true, Enabled: true, Action: model.SensitiveWordActionMonitor, Description: `【英文正则·组合】非法转移类动词 + 未成年类词`},
		{Pattern: `(?i)(c[\s._-]p)\s*(video|videos|porn|material|collection|images?)`, IsRegex: true, Enabled: true, Action: model.SensitiveWordActionMonitor, Description: `【英文正则·绕过】CSAM 缩写（容忍点/横线/下划线分隔） + 必须有内容词上下文，避免 CP 二字单独误伤`},
		{Pattern: `(?i)(jb|j\.b\.)\s*(porn|nude|naked|girl)`, IsRegex: true, Enabled: true, Action: model.SensitiveWordActionMonitor, Description: `【英文正则·缩写】jailbait 隐写绕过（jb / j.b.） + 性词上下文`},
	}
	result := SensitiveWordsSeedResult{}
	for _, rule := range defaults {
		var existing model.SensitiveWord
		err := model.DB.Where("pattern = ?", rule.Pattern).First(&existing).Error
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return result, err
		}
		if err == nil {
			rule.Id = existing.Id
			if err := model.UpdateSensitiveWord(&rule); err != nil {
				return result, err
			}
			result.Updated++
			continue
		}
		if err := model.CreateSensitiveWord(&rule); err != nil {
			return result, err
		}
		result.Created++
	}
	if err := ReloadSensitiveMonitorRules(); err != nil {
		return result, err
	}
	return result, nil
}
