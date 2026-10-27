// Package adapters 提供常见付费组件的余额适配器（v1.4，F-016 多源；
// 契约参照 ai-support-platform 管理监控页 #1243 的实现口径）。
package adapters

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/zoyentech/kas/internal/balance"
	"github.com/zoyentech/kas/internal/domain"
)

// DeepSeekDefaultEndpoint 官方余额端点（可用 env 覆盖走网关）。
const DeepSeekDefaultEndpoint = "https://api.deepseek.com/user/balance"

// NewDeepSeek 由 env 名构建 DeepSeek 余额源（Bearer token；端点默认官方，
// endpointEnv 非空且设置时覆盖）。env 未设置返回 ErrInvalidConfig（字段路径
// 指向 env 名，不含 token 值）。
func NewDeepSeek(tokenEnv, endpointEnv string) (balance.Source, error) {
	token := getenv(tokenEnv)
	if token == "" {
		return nil, fmt.Errorf("%w: 环境变量 %s 未设置或为空", domain.ErrInvalidConfig, tokenEnv)
	}
	endpoint := DeepSeekDefaultEndpoint
	if endpointEnv != "" {
		if v := getenv(endpointEnv); v != "" {
			endpoint = v
		}
	}
	return &deepSeek{endpoint: endpoint, token: token, client: &http.Client{Timeout: 5 * time.Second}}, nil
}

type deepSeek struct {
	endpoint string
	token    string
	client   *http.Client
}

// deepSeekBalanceResponse DeepSeek 余额契约（多币种 balance_infos；参考实现
// 同构：is_available=false 表示服务端暂未提供余额信息）。
type deepSeekBalanceResponse struct {
	IsAvailable  bool `json:"is_available"`
	BalanceInfos []struct {
		Currency        string `json:"currency"`
		TotalBalance    string `json:"total_balance"`
		GrantedBalance  string `json:"granted_balance"`
		ToppedUpBalance string `json:"topped_up_balance"`
	} `json:"balance_infos"`
}

// Balance 查询余额：多币种时优先 CNY，否则首条；金额为 total_balance。
func (d *deepSeek) Balance(ctx context.Context) (balance.Info, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, d.endpoint, nil)
	if err != nil {
		return balance.Info{}, fmt.Errorf("kas/balance/deepseek: 构造请求失败: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+d.token)
	resp, err := d.client.Do(req)
	if err != nil {
		return balance.Info{}, fmt.Errorf("kas/balance/deepseek: 查询失败: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return balance.Info{}, fmt.Errorf("kas/balance/deepseek: 上游状态码 %d", resp.StatusCode)
	}
	var body deepSeekBalanceResponse
	dec := json.NewDecoder(io.LimitReader(resp.Body, 1<<20))
	if err := dec.Decode(&body); err != nil {
		return balance.Info{}, fmt.Errorf("%w: deepseek 余额响应解析失败", domain.ErrProviderContract)
	}
	if !body.IsAvailable {
		return balance.Info{}, errors.New("kas/balance/deepseek: 服务端暂未提供余额信息（is_available=false）")
	}
	chosen := -1
	for i, info := range body.BalanceInfos {
		if info.Currency == "CNY" {
			chosen = i
			break
		}
	}
	if chosen < 0 && len(body.BalanceInfos) > 0 {
		chosen = 0
	}
	if chosen < 0 {
		return balance.Info{}, fmt.Errorf("%w: deepseek 余额响应无 balance_infos", domain.ErrProviderContract)
	}
	info := body.BalanceInfos[chosen]
	amount, err := strconv.ParseFloat(info.TotalBalance, 64)
	if err != nil {
		return balance.Info{}, fmt.Errorf("%w: deepseek 余额金额格式异常", domain.ErrProviderContract)
	}
	unit := info.Currency
	if unit == "" {
		unit = "CNY"
	}
	return balance.Info{Source: "deepseek", Amount: amount, Unit: unit, FetchedAt: time.Now()}, nil
}
