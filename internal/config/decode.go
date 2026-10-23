package config

import (
	"errors"
	"fmt"
	"io"

	"gopkg.in/yaml.v3"
)

// decodeStrict 以 KnownFields 严格模式解析 YAML：未知字段拒绝，错误附
// 行号与字段名（Story 2.1.1.1 T2）。
func decodeStrict(r io.Reader, out *Config) error {
	dec := yaml.NewDecoder(r)
	dec.KnownFields(true)
	err := dec.Decode(out)
	if errors.Is(err, io.EOF) {
		return singleError("(root)", "配置为空")
	}
	if err != nil {
		return wrapTypeError(err)
	}
	// 仅允许单文档：第二个 Decode 必须立即 EOF。
	if err := dec.Decode(out); !errors.Is(err, io.EOF) {
		if err == nil {
			return singleError("(root)", "多文档 YAML 不支持，仅接受单个配置文档")
		}
		return wrapTypeError(err)
	}
	return nil
}

// wrapTypeError 将 yaml TypeError 转为违规聚合，其余错误挂 ErrInvalidConfig 链。
func wrapTypeError(err error) error {
	var typeErr *yaml.TypeError
	if errors.As(err, &typeErr) {
		vs := make([]Violation, 0, len(typeErr.Errors))
		for _, e := range typeErr.Errors {
			vs = append(vs, newViolation("(yaml)", "%s", e))
		}
		return &ValidationError{Violations: vs}
	}
	// Duration 等自定义 UnmarshalYAML 的错误：不含配置正文，原样挂链。
	return fmt.Errorf("%w: %v", ErrInvalidConfig, err)
}
