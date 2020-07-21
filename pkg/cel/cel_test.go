package cel

import (
	"regexp"
	"testing"

	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
	"github.com/google/go-cmp/cmp"
	pipelinev1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
)

const testRepoURL = "https://example.com/example/example.git"

func TestExpressionEvaluation(t *testing.T) {
	tests := []struct {
		name    string
		expr    string
		fixture interface{}
		want    ref.Val
	}{
		{
			name: "simple body value",
			expr: "context.commit.id",
			fixture: map[string]interface{}{
				"commit": map[string]string{
					"id": "testing",
				},
			},
			want: types.String("testing"),
		},
		{
			name:    "repoURL",
			expr:    "repoURL",
			fixture: map[string]interface{}{},
			want:    types.String(testRepoURL),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(rt *testing.T) {
			env, err := makeCelEnv()
			if err != nil {
				rt.Errorf("failed to make env: %s", err)
				return
			}
			ectx, err := makeEvalContext(testRepoURL, tt.fixture)
			if err != nil {
				rt.Errorf("failed to make eval context %s", err)
				return
			}
			got, err := evaluate(tt.expr, env, ectx)
			if err != nil {
				rt.Errorf("evaluate() got an error %s", err)
				return
			}
			_, ok := got.(*types.Err)
			if ok {
				rt.Errorf("error evaluating expression: %s", got)
				return
			}

			if !got.Equal(tt.want).(types.Bool) {
				rt.Errorf("evaluate() = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestExpressionEvaluation_Error(t *testing.T) {
	tests := []struct {
		name string
		expr string
		want string
	}{
		{
			name: "unknown value",
			expr: "context.Unknown",
			want: "no such key: Unknown",
		},
		{
			name: "invalid syntax",
			expr: "body.value = 'testing'",
			want: "Syntax error: token recognition error",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(rt *testing.T) {
			env, err := makeCelEnv()
			if err != nil {
				rt.Errorf("failed to make env: %s", err)
				return
			}
			ectx, err := makeEvalContext(testRepoURL, map[string]string{"this": "tests"})
			if err != nil {
				rt.Errorf("failed to make eval context %s", err)
				return
			}
			_, err = evaluate(tt.expr, env, ectx)
			if !matchError(t, tt.want, err) {
				rt.Errorf("evaluate() got %s, wanted %s", err, tt.want)
			}
		})
	}
}

func TestContextEvaluateToParamValue(t *testing.T) {
	v := map[string]interface{}{
		"push": map[string]interface{}{
			"commits": []map[string]string{
				{"id": "value1"},
				{"id": "value2"},
			},
		},
		"head": "test-value",
	}

	ctx, err := New(testRepoURL, v)
	if err != nil {
		t.Fatal(err)
	}
	result, err := ctx.EvaluateToParamValue("context.push.commits.map(s, s['id'])")
	if err != nil {
		t.Fatal(err)
	}

	want := pipelinev1beta1.ArrayOrString{
		Type:     pipelinev1beta1.ParamTypeArray,
		ArrayVal: []string{"value1", "value2"},
	}
	if diff := cmp.Diff(want, result); diff != "" {
		t.Fatalf("got %#v, want %#v\n", result, want)
	}
}

// TODO move this and share via a specific test package.
func matchError(t *testing.T, s string, e error) bool {
	t.Helper()
	match, err := regexp.MatchString(s, e.Error())
	if err != nil {
		t.Fatal(err)
	}
	return match
}
