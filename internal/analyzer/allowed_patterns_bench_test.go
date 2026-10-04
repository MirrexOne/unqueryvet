package analyzer

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/MirrexOne/unqueryvet/pkg/config"
)

func BenchmarkAllowedPatternsAnalysis(b *testing.B) {
	for _, count := range []int{1, 100, 1000} {
		b.Run(fmt.Sprintf("strings=%d", count), func(b *testing.B) {
			var source strings.Builder
			source.WriteString("package p\nfunc query(string) {}\nfunc queries() {\n")
			for i := range count {
				fmt.Fprintf(&source, "query(\"ordinary message %d\")\n", i)
			}
			source.WriteString("}\n")

			pass := allowedPatternsPass(b, source.String())
			cfg := config.DefaultSettings()
			b.ReportAllocs()
			for b.Loop() {
				_, err := RunWithConfig(pass, &cfg)
				require.NoError(b, err)
			}
		})
	}
}
