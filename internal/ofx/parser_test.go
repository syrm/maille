package ofx

import (
	"testing"
)

// Benchmarks pour splitOnStmtTrn.
// Exécuter avec :
//   go test -bench=BenchmarkSplitOnStmtTrn -benchmem -count=6 ./...
//
// -count=6 pour avoir assez d'échantillons à passer dans benchstat :
//   go install golang.org/x/perf/cmd/benchstat@latest
//   go test -bench=BenchmarkSplitOnStmtTrn -benchmem -count=6 ./... | tee bench.txt
//   benchstat bench.txt

var (
	// Sink variables pour empêcher le compilateur d'éliminer les appels (dead code elimination).
	sinkAdvance int
	sinkToken   []byte
	sinkErr     error
)

func BenchmarkSplitOnStmtTrn(b *testing.B) {
	p := Parser{} // ajuster l'initialisation si nécessaire

	tests := []struct {
		name  string
		data  []byte
		atEOF bool
	}{
		{
			name:  "match_simple",
			data:  buildInput("<STMT>some transaction data</STMT>"), // adapter les tags réels
			atEOF: false,
		},
		{
			name:  "no_open_tag",
			data:  []byte("garbage data with no statement tag"),
			atEOF: false,
		},
		{
			name:  "open_no_close",
			data:  buildInput("<STMT>partial data without close"),
			atEOF: false,
		},
		{
			name:  "at_eof_no_match",
			data:  []byte("garbage"),
			atEOF: true,
		},
		{
			name:  "large_payload",
			data:  buildLargeInput(64 * 1024), // 64 KB
			atEOF: false,
		},
	}

	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			b.ReportAllocs()
			b.SetBytes(int64(len(tt.data)))

			for b.Loop() {
				sinkAdvance, sinkToken, sinkErr = p.splitOnStmtTrn(tt.data, tt.atEOF)
			}
		})
	}
}

// buildInput construit un input valide avec les vrais tags STMT.
// Adapter STMTopen/STMTclose à tes vraies valeurs.
func buildInput(raw string) []byte {
	return []byte(raw)
}

// buildLargeInput construit un buffer de taille n avec un statement valide à la fin,
// simulant le pire cas où bytes.Index doit scanner loin.
func buildLargeInput(n int) []byte {
	buf := make([]byte, 0, n)
	// Remplir avec du padding
	for len(buf) < n-200 {
		buf = append(buf, "XXXXXXXX"...)
	}
	// Placer un statement valide à la fin
	buf = append(buf, "<STMT>transaction data</STMT>"...) // adapter les tags
	return buf
}
