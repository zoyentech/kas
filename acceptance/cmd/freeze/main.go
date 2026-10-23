// freeze 生成/校验 GoldenSet 内容 hash 冻结件（Story 7.1.1.1 T3）。
package main

import (
	"fmt"
	"os"

	acc "github.com/kas-runtime/kas/acceptance/internal"
)

func main() {
	cases, err := acc.LoadDir("acceptance/goldenset/cases")
	if err != nil {
		fail(err)
	}
	hash, err := acc.ContentHash(cases)
	if err != nil {
		fail(err)
	}
	p0 := acc.P0Subset(cases)
	if len(p0) < 20 {
		fail(fmt.Errorf("P0 子集 %d 条 < 20 条建议值（O-3）", len(p0)))
	}
	if err := os.WriteFile("acceptance/goldenset/goldenset.version", []byte(hash+"\n"), 0o644); err != nil {
		fail(err)
	}
	fmt.Printf("冻结：合计 %d 条，P0 %d 条，hash %s\n", len(cases), len(p0), hash[:16])
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "freeze:", err)
	os.Exit(1)
}
