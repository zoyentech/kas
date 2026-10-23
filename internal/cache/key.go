package cache

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"

	"github.com/kas-runtime/kas/internal/config"
)

// KeyParts 是 Evidence 缓存 key 的七维（docs/design/04 §2 公式逐项）。
// 各维度以"已摘要化或不透明"形态参与：Question 是规范化原文（经
// QuestionDigest 摘要）、ScopeFingerprint 是宿主生成的不透明指纹、
// 其余为版本/标识字符串。组合阶段即完成定界与摘要，存储侧只保留最终
// 不可逆 key 摘要，不保留问题原文与身份明文。
type KeyParts struct {
	QuestionDigest     string
	ConfigVersion      string
	ScopeFingerprint   string
	AuthorizationEpoch string
	Namespace          string
	CorpusGeneration   string
	PolicyVersion      string
}

// Combine 生成待哈希的确定性字节串：固定字段序 + '|' 定界（hex 摘要不含
// '|'，防拼接歧义）。任一维度变化 → 字节串变化。
func Combine(parts KeyParts) []byte {
	return []byte("q=" + parts.QuestionDigest +
		"|cv=" + parts.ConfigVersion +
		"|sf=" + shortDigest(parts.ScopeFingerprint) +
		"|ae=" + shortDigest(parts.AuthorizationEpoch) +
		"|ns=" + shortDigest(parts.Namespace) +
		"|cg=" + shortDigest(parts.CorpusGeneration) +
		"|pv=" + parts.PolicyVersion)
}

// Digest 返回存储主键：SHA-256(Combine) 全量 hex。
func Digest(parts KeyParts) string {
	sum := sha256.Sum256(Combine(parts))
	return hex.EncodeToString(sum[:])
}

func shortDigest(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])[:24]
}

// KeyInput 是由原始请求维度构建 KeyParts 的输入（问题为规范化原文）。
type KeyInput struct {
	Question           string
	ConfigVersion      string
	ScopeFingerprint   string
	AuthorizationEpoch string
	Namespace          string
	CorpusGeneration   string
	CasePolicy         CasePolicy
	Policy             config.EvidenceConfig
}

// Parts 摘要化问题并计算策略版本，输出七维 Parts。
func (in KeyInput) Parts() KeyParts {
	return KeyParts{
		QuestionDigest:     QuestionDigest(in.Question, in.CasePolicy),
		ConfigVersion:      in.ConfigVersion,
		ScopeFingerprint:   in.ScopeFingerprint,
		AuthorizationEpoch: in.AuthorizationEpoch,
		Namespace:          in.Namespace,
		CorpusGeneration:   in.CorpusGeneration,
		PolicyVersion:      PolicyVersion(in.Policy),
	}
}

// DigestOf 由原始输入直接计算存储主键。
func (in KeyInput) DigestOf() string { return Digest(in.Parts()) }

// PolicyVersion 计算检索/有效期策略子集的规范化哈希：top_k、
// max_candidate_groups 与 validity 节（max-age 表按 JSON 键序稳定序列化）。
// 策略变化自然更换 key，防止策略变化后复用旧条目。
func PolicyVersion(cfg config.EvidenceConfig) string {
	subset := struct {
		TopK               int                   `json:"top_k"`
		MaxCandidateGroups int                   `json:"max_candidate_groups"`
		Validity           config.ValidityConfig `json:"validity"`
	}{TopK: cfg.TopK, MaxCandidateGroups: cfg.MaxCandidateGroups, Validity: cfg.Validity}
	b, err := json.Marshal(subset)
	if err != nil {
		// 仅含配置值与 map[string]Duration，Marshal 不会失败
		return "policy-marshal-error"
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])[:24]
}

// sha256Hex 返回字节串的 SHA-256 hex。
func sha256Hex(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}
