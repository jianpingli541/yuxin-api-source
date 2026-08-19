package routing

import (
	"fmt"
	"sort"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/model_setting"

	"github.com/gin-gonic/gin"
)

// AutoCandidate 智能调度候选：一个真实模型 + 其下一个可用渠道。
type AutoCandidate struct {
	ModelName string
	Channel   *model.Channel
}

// ResolveAutoModel 智能调度核心：model 为注册别名时，解析候选池并在
// (模型 × 渠道) 并集上按策略选定最终组合。
//
// 调用点：middleware/distributor.go（token 模型限制校验之后、affinity/随机选渠道之前）。
// 返回 (选定渠道, 实际模型名, true)；非别名或池内无可用渠道返回 (nil, "", false)，
// 调用方继续走原有分发逻辑（别名将按未知模型报 503，行为与现状一致）。
//
// 分组语义：token 指定分组（含 auto 展开）→ 用该分组过滤候选；否则用用户默认分组。
// 效果/速度权衡：X-Yuxin-Route 头（balanced/quality/speed），见 GetRouteBiasFromContext。
func ResolveAutoModel(c *gin.Context, aliasModel, usingGroup string) (*model.Channel, string, bool) {
	pool, ok := model_setting.ResolveAlias(aliasModel)
	if !ok {
		return nil, "", false
	}

	// 解析有效分组：token 分组优先；auto 分组展开为首个可用分组（候选过滤用）。
	group := usingGroup
	if group == "auto" || group == "" {
		userGroup := common.GetContextKeyString(c, constant.ContextKeyUserGroup)
		autoGroups := service.GetRequestAutoGroups(c, userGroup)
		if len(autoGroups) > 0 {
			group = autoGroups[0]
		} else {
			group = userGroup
		}
	}
	if group == "" {
		return nil, "", false
	}

	// 候选并集：池内每个模型的可用渠道
	var candidates []AutoCandidate
	for _, m := range pool {
		abilities, err := model.GetGroupEnabledAbilitiesByModel(group, m)
		if err != nil {
			continue
		}
		for _, ab := range abilities {
			ch, err := model.CacheGetChannel(ab.ChannelId)
			if err != nil || ch == nil || ch.Status != common.ChannelStatusEnabled {
				continue
			}
			candidates = append(candidates, AutoCandidate{ModelName: m, Channel: ch})
		}
	}
	if len(candidates) == 0 {
		return nil, "", false
	}

	// 单候选直接命中
	if len(candidates) == 1 {
		markAutoResolved(c, candidates[0].ModelName)
		return candidates[0].Channel, candidates[0].ModelName, true
	}

	// 多候选：按 (模型, 渠道) 唯一对聚合（同渠道多模型视为不同组合）
	uniq := dedupeCandidates(candidates)

	// 用渠道列表跑策略；同渠道多模型时按模型质量分再裁决。
	channels := candidateChannels(uniq)
	ctx := &RoutingContext{
		ModelName: aliasModel,
		UserGroup: group,
		Strategy:  StrategyBalanced, // 智能调度固定双维策略；X-Yuxin-Route 微调权重
		RouteBias: GetRouteBiasFromContext(c),
	}
	selected, err := GetRouter().SelectChannel(ctx, channels)
	if err != nil || selected == nil {
		// 策略失败兜底：第一个候选
		markAutoResolved(c, uniq[0].ModelName)
		return uniq[0].Channel, uniq[0].ModelName, true
	}

	// 同渠道多模型：按 canary 质量分选该渠道下最优模型
	best := uniq[0]
	bestScore := -1.0
	for _, cand := range uniq {
		if cand.Channel.Id != selected.Id {
			continue
		}
		score := GetRouter().GetQualityScore(cand.Channel.Id)
		if score > bestScore {
			bestScore = score
			best = cand
		}
	}

	common.SysLog(fmt.Sprintf("[AutoRoute] alias=%s pool=%d candidates=%d -> model=%s channel=#%d(%s) bias=%s",
		aliasModel, len(pool), len(uniq), best.ModelName, best.Channel.Id, best.Channel.Name, ctx.RouteBias))

	markAutoResolved(c, best.ModelName)
	return best.Channel, best.ModelName, true
}

// markAutoResolved 写入调度结果上下文：original_model=实际模型（计费/日志/响应头链），
// auto_route_resolved 标记供 relay 层短路二次渠道选择；X-Yuxin-Alias 头回显别名。
func markAutoResolved(c *gin.Context, resolvedModel string) {
	c.Set("original_model", resolvedModel)
	c.Set("auto_route_resolved", true)
	if c.Writer != nil {
		c.Writer.Header().Set("X-Yuxin-Alias", "auto")
	}
}

func dedupeCandidates(in []AutoCandidate) []AutoCandidate {
	seen := make(map[string]bool, len(in))
	out := make([]AutoCandidate, 0, len(in))
	for _, cand := range in {
		key := fmt.Sprintf("%d|%s", cand.Channel.Id, cand.ModelName)
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, cand)
	}
	// 稳定排序，保证同配置下调度结果可复现（策略同分时取序首）
	sort.Slice(out, func(i, j int) bool { return keyLess(out[i], out[j]) })
	return out
}

func keyLess(a, b AutoCandidate) bool {
	if a.Channel.Id != b.Channel.Id {
		return a.Channel.Id < b.Channel.Id
	}
	return a.ModelName < b.ModelName
}

func candidateChannels(in []AutoCandidate) []*model.Channel {
	seen := make(map[int]bool, len(in))
	out := make([]*model.Channel, 0, len(in))
	for _, cand := range in {
		if seen[cand.Channel.Id] {
			continue
		}
		seen[cand.Channel.Id] = true
		out = append(out, cand.Channel)
	}
	return out
}
