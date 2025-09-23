package builder

import (
	"fmt"
	"math/rand"
	"strings"
	"testing"
	"time"

	"google.golang.org/protobuf/proto"

	pb "go.unistack.org/micro-client-http/v4/builder/proto"
)

// sink prevents the compiler from optimizing away parsePathTemplate results.
var sink *pathTemplate

func BenchmarkParsePathTemplate(b *testing.B) {
	r := rand.New(rand.NewSource(1))

	benchInput := func(size int) string {
		sb := strings.Builder{}
		sb.Grow(size * 10)

		for i := 0; i < size; i++ {
			name := fmt.Sprintf("var%d", r.Intn(1000))

			if r.Intn(5) == 0 {
				sb.WriteString(fmt.Sprintf("{%s=**}", name))
			} else {
				sb.WriteString(fmt.Sprintf("{%s}", name))
			}
		}
		return sb.String()
	}

	sizes := []int{1_000, 10_000, 50_000, 100_000}
	for _, size := range sizes {
		input := benchInput(size)
		b.Run(fmt.Sprintf("N=%d", size), func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				var err error
				sink, err = parsePathTemplate(input)
				if err != nil && testing.Verbose() {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkRequestBuilder(b *testing.B) {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	makeMsg := func(fieldCount int) proto.Message {
		switch fieldCount {
		case 5:
			return &pb.Benchmark_Case5{
				Field1: fmt.Sprintf("value%d", r.Intn(1000)),
				Field2: fmt.Sprintf("value%d", r.Intn(1000)),
				Field3: fmt.Sprintf("value%d", r.Intn(1000)),
				Field4: fmt.Sprintf("value%d", r.Intn(1000)),
				Field5: fmt.Sprintf("value%d", r.Intn(1000)),
			}
		case 10:
			return &pb.Benchmark_Case10{
				Field1:  fmt.Sprintf("value%d", r.Intn(1000)),
				Field2:  fmt.Sprintf("value%d", r.Intn(1000)),
				Field3:  fmt.Sprintf("value%d", r.Intn(1000)),
				Field4:  fmt.Sprintf("value%d", r.Intn(1000)),
				Field5:  fmt.Sprintf("value%d", r.Intn(1000)),
				Field6:  fmt.Sprintf("value%d", r.Intn(1000)),
				Field7:  fmt.Sprintf("value%d", r.Intn(1000)),
				Field8:  fmt.Sprintf("value%d", r.Intn(1000)),
				Field9:  fmt.Sprintf("value%d", r.Intn(1000)),
				Field10: fmt.Sprintf("value%d", r.Intn(1000)),
			}
		case 30:
			return &pb.Benchmark_Case30{
				Field1:  fmt.Sprintf("value%d", r.Intn(1000)),
				Field2:  fmt.Sprintf("value%d", r.Intn(1000)),
				Field3:  fmt.Sprintf("value%d", r.Intn(1000)),
				Field4:  fmt.Sprintf("value%d", r.Intn(1000)),
				Field5:  fmt.Sprintf("value%d", r.Intn(1000)),
				Field6:  fmt.Sprintf("value%d", r.Intn(1000)),
				Field7:  fmt.Sprintf("value%d", r.Intn(1000)),
				Field8:  fmt.Sprintf("value%d", r.Intn(1000)),
				Field9:  fmt.Sprintf("value%d", r.Intn(1000)),
				Field10: fmt.Sprintf("value%d", r.Intn(1000)),
				Field11: fmt.Sprintf("value%d", r.Intn(1000)),
				Field12: fmt.Sprintf("value%d", r.Intn(1000)),
				Field13: fmt.Sprintf("value%d", r.Intn(1000)),
				Field14: fmt.Sprintf("value%d", r.Intn(1000)),
				Field15: fmt.Sprintf("value%d", r.Intn(1000)),
				Field16: fmt.Sprintf("value%d", r.Intn(1000)),
				Field17: fmt.Sprintf("value%d", r.Intn(1000)),
				Field18: fmt.Sprintf("value%d", r.Intn(1000)),
				Field19: fmt.Sprintf("value%d", r.Intn(1000)),
				Field20: fmt.Sprintf("value%d", r.Intn(1000)),
				Field21: fmt.Sprintf("value%d", r.Intn(1000)),
				Field22: fmt.Sprintf("value%d", r.Intn(1000)),
				Field23: fmt.Sprintf("value%d", r.Intn(1000)),
				Field24: fmt.Sprintf("value%d", r.Intn(1000)),
				Field25: fmt.Sprintf("value%d", r.Intn(1000)),
				Field26: fmt.Sprintf("value%d", r.Intn(1000)),
				Field27: fmt.Sprintf("value%d", r.Intn(1000)),
				Field28: fmt.Sprintf("value%d", r.Intn(1000)),
				Field29: fmt.Sprintf("value%d", r.Intn(1000)),
				Field30: fmt.Sprintf("value%d", r.Intn(1000)),
			}
		default:
			b.Fatal("undefined field count")
			return nil
		}
	}

	tests := []struct {
		name       string
		pathTmpl   string
		bodyOption string
	}{
		{"all fields in path", "/resource/{field1}/{field2}", ""},
		{"single field body", "/resource/{field1}", "field4"},
		{"full body", "/resource", "*"},
	}

	for _, fields := range []int{5, 10, 30} {
		for _, tt := range tests {
			b.Run(fmt.Sprintf("%s_%d_fields", tt.name, fields), func(b *testing.B) {
				b.ReportAllocs()
				b.ResetTimer()

				for i := 0; i < b.N; i++ {
					msg := makeMsg(fields)
					rb, err := NewRequestBuilder(tt.pathTmpl, "POST", tt.bodyOption, msg)
					if err != nil {
						b.Fatalf("new request builder: %v", err)
					}

					_, _, err = rb.Build()
					if err != nil {
						b.Fatalf("build: %v", err)
					}
				}
			})
		}
	}
}
