package jev

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/kas-runtime/kas/internal/decision"
	"github.com/kas-runtime/kas/internal/domain"
)

// decisionValidate 复用统一响应校验器（信任边界一致，F-010）。
var decisionValidate = decision.ValidateDecision

// parseResponse 严格解析 Jev 响应（Story 5.2.1.2 T1）：未知字段拒绝、
// 畸形体/多文档 → ErrProviderContract；错误不含响应原文。
func parseResponse(r io.Reader) (jevResponse, error) {
	var jr jevResponse
	dec := json.NewDecoder(r)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&jr); err != nil {
		return jr, fmt.Errorf("%w: Jev 响应解析失败（畸形体或未知字段）", domain.ErrProviderContract)
	}
	if err := dec.Decode(new(json.RawMessage)); !errors.Is(err, io.EOF) {
		return jr, fmt.Errorf("%w: Jev 响应含多个 JSON 文档", domain.ErrProviderContract)
	}
	return jr, nil
}
