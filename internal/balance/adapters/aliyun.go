package adapters

import (
	"context"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/zoyentech/kas/internal/balance"
	"github.com/zoyentech/kas/internal/domain"
)

// Aliyun 常量（BSS QueryAccountBalance，POP RPC V1 签名；零 SDK 依赖）。
const (
	aliyunBSSEndpoint = "https://business.aliyuncs.com"
	aliyunBSSVersion  = "2017-12-14"
)

// NewAliyunBSS 由 env 名构建阿里云账户现金余额源（Qwen/百炼等模型的账号级
// 计费口径——参考实现同结论：非 API Key 专属余额）。需 AK/SK env 均已设置；
// SecurityToken env 可选（STS）。endpointEnv 可覆盖（默认 business.aliyuncs.com）。
func NewAliyunBSS(akEnv, skEnv, tokenEnv, endpointEnv string) (balance.Source, error) {
	ak := getenv(akEnv)
	sk := getenv(skEnv)
	if ak == "" || sk == "" {
		return nil, fmt.Errorf("%w: 环境变量 %s/%s 未设置或为空", domain.ErrInvalidConfig, akEnv, skEnv)
	}
	endpoint := aliyunBSSEndpoint
	if endpointEnv != "" {
		if v := getenv(endpointEnv); v != "" {
			endpoint = v
		}
	}
	return &aliyunBSS{
		ak: ak, sk: sk, token: getenv(tokenEnv), endpoint: endpoint,
		client: &http.Client{Timeout: 5 * time.Second},
	}, nil
}

type aliyunBSS struct {
	ak, sk, token, endpoint string
	client                  *http.Client
}

// aliyunBSSResponse QueryAccountBalance 响应（AvailableCashAmount：账户现金余额 CNY）。
type aliyunBSSResponse struct {
	Success *bool `json:"Success"`
	Data    *struct {
		AvailableCashAmount *string `json:"AvailableCashAmount"`
		Currency            *string `json:"Currency"`
	} `json:"Data"`
}

// Balance 查询账户现金余额（RPC 签名请求）。
func (a *aliyunBSS) Balance(ctx context.Context) (balance.Info, error) {
	params := map[string]string{
		"Action":           "QueryAccountBalance",
		"Version":          aliyunBSSVersion,
		"Format":           "JSON",
		"AccessKeyId":      a.ak,
		"SignatureMethod":  "HMAC-SHA1",
		"SignatureVersion": "1.0",
		"SignatureNonce":   nonce(),
		"Timestamp":        time.Now().UTC().Format("2006-01-02T15:04:05Z"),
	}
	if a.token != "" {
		params["SecurityToken"] = a.token
	}
	params["Signature"] = a.sign(params)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, a.endpoint+"/?"+encodeQuery(params), nil)
	if err != nil {
		return balance.Info{}, fmt.Errorf("kas/balance/aliyun: 构造请求失败: %w", err)
	}
	resp, err := a.client.Do(req)
	if err != nil {
		return balance.Info{}, fmt.Errorf("kas/balance/aliyun: 查询失败: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return balance.Info{}, fmt.Errorf("kas/balance/aliyun: 上游状态码 %d（检查 RAM 凭据与 BSS 只读权限）", resp.StatusCode)
	}
	var body aliyunBSSResponse
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&body); err != nil {
		return balance.Info{}, fmt.Errorf("%w: aliyun 余额响应解析失败", domain.ErrProviderContract)
	}
	if body.Success == nil || !*body.Success || body.Data == nil || body.Data.AvailableCashAmount == nil ||
		strings.TrimSpace(*body.Data.AvailableCashAmount) == "" {
		return balance.Info{}, errors.New("kas/balance/aliyun: 查询失败（RAM 凭据或 BSS 只读权限异常）")
	}
	amount, err := strconv.ParseFloat(strings.TrimSpace(*body.Data.AvailableCashAmount), 64)
	if err != nil {
		return balance.Info{}, fmt.Errorf("%w: aliyun 余额金额格式异常", domain.ErrProviderContract)
	}
	unit := "CNY"
	if body.Data.Currency != nil && strings.TrimSpace(*body.Data.Currency) != "" {
		unit = strings.TrimSpace(*body.Data.Currency)
	}
	return balance.Info{Source: "aliyun", Amount: amount, Unit: unit, FetchedAt: time.Now()}, nil
}

// sign 计算 POP RPC V1 签名（HMAC-SHA1 over "GET&%2F&" + 编码查询串）。
func (a *aliyunBSS) sign(params map[string]string) string {
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	pairs := make([]string, 0, len(keys))
	for _, k := range keys {
		pairs = append(pairs, percentEncode(k)+"="+percentEncode(params[k]))
	}
	stringToSign := "GET&" + percentEncode("/") + "&" + percentEncode(strings.Join(pairs, "&"))
	mac := hmac.New(sha1.New, []byte(a.sk+"&"))
	mac.Write([]byte(stringToSign))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

// percentEncode RFC3986 参数编码（POP 口径：+→%20、*→%2A、%7E→~）。
func percentEncode(s string) string {
	return strings.NewReplacer("+", "%20", "*", "%2A", "%7E", "~").Replace(url.QueryEscape(s))
}

// encodeQuery 生成含签名的查询串（签名值本身也要编码）。
func encodeQuery(params map[string]string) string {
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	pairs := make([]string, 0, len(keys))
	for _, k := range keys {
		pairs = append(pairs, percentEncode(k)+"="+percentEncode(params[k]))
	}
	return strings.Join(pairs, "&")
}

// nonce 简易唯一串（纳秒时间戳组合；POP 只要求单请求唯一）。
func nonce() string {
	return fmt.Sprintf("kas-%d-%d", time.Now().UnixNano(), time.Now().UnixMicro())
}

// getenv 避免 adapters 直接依赖 os.Getenv 便于测试替换。
var getenv = os.Getenv

// SetEnvFunc 测试注入（生产勿用）。
func SetEnvFunc(fn func(string) string) { getenv = fn }
