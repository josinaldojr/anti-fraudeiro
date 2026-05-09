package fraud

import "testing"

func TestNormalizeBucketStrategy(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		input string
		want  BucketStrategy
	}{
		{name: "default", input: "", want: BucketStrategyWindow},
		{name: "window", input: "window", want: BucketStrategyWindow},
		{name: "ordered", input: "ordered", want: BucketStrategyOrdered},
		{name: "mixed case", input: " OrDeReD ", want: BucketStrategyOrdered},
		{name: "unknown", input: "foo", want: BucketStrategyWindow},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			if got := NormalizeBucketStrategy(testCase.input); got != testCase.want {
				t.Fatalf("NormalizeBucketStrategy(%q) = %q, want %q", testCase.input, got, testCase.want)
			}
		})
	}
}
