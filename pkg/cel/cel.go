package cel

import (
	"encoding/json"
	"fmt"
	"reflect"

	structpb "github.com/golang/protobuf/ptypes/struct"
	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/checker/decls"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
	"github.com/google/cel-go/common/types/traits"
	pipelinev1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
)

var (
	listType = reflect.TypeOf(&structpb.ListValue{})
)

// Context makes it easy to execute CEL expressions on JSON body.
type Context struct {
	env  *cel.Env
	Data map[string]interface{}
}

// New creates and returns a Context for evaluating expressions.
func New(context interface{}) (*Context, error) {
	env, err := makeCelEnv()
	if err != nil {
		return nil, err
	}
	ctx, err := makeEvalContext(context)
	if err != nil {
		return nil, err
	}
	return &Context{
		env:  env,
		Data: ctx,
	}, nil
}

// Evaluate evaluates the provided expression and returns the result.
func (c *Context) Evaluate(expr string) (ref.Val, error) {
	return evaluate(expr, c.env, c.Data)
}

// EvaluateToParamValue evaluates the provided expression, and converts it to a
// pipeline ArrayOrString value for a PipelineRun parameter.
func (c *Context) EvaluateToParamValue(expr string) (pipelinev1beta1.ArrayOrString, error) {
	res, err := c.Evaluate(expr)
	if err != nil {
		return pipelinev1beta1.ArrayOrString{}, err
	}
	return valToParam(res)
}

func evaluate(expr string, env *cel.Env, data map[string]interface{}) (ref.Val, error) {
	parsed, issues := env.Parse(expr)
	if issues != nil && issues.Err() != nil {
		return nil, issues.Err()
	}

	checked, issues := env.Check(parsed)
	if issues != nil && issues.Err() != nil {
		return nil, issues.Err()
	}

	prg, err := env.Program(checked)
	if err != nil {
		return nil, err
	}

	out, _, err := prg.Eval(data)
	return out, err
}

func makeCelEnv() (*cel.Env, error) {
	return cel.NewEnv(
		cel.Declarations(
			decls.NewIdent("context", decls.Dyn, nil)))
}

func makeEvalContext(context interface{}) (map[string]interface{}, error) {
	m, err := contextToMap(context)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{"context": m}, nil
}

func contextToMap(v interface{}) (map[string]interface{}, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	var m map[string]interface{}
	err = json.Unmarshal(b, &m)
	return m, err
}

// TODO: This should probably stringify other ref.Types
func valToParam(v ref.Val) (pipelinev1beta1.ArrayOrString, error) {
	switch val := v.(type) {
	case traits.Lister:
		items := []string{}
		it := val.Iterator()
		for it.HasNext() == types.True {
			str, err := it.Next().ConvertToNative(reflect.TypeOf(""))
			if err != nil {
				return pipelinev1beta1.ArrayOrString{}, fmt.Errorf("failed to convert expression to a string: %w", err)
			}
			items = append(items, str.(string))
		}
		return pipelinev1beta1.NewArrayOrString(items[0], items[1:]...), nil
	case types.String:
		return pipelinev1beta1.NewArrayOrString(val.Value().(string)), nil
	case types.Double:
		return pipelinev1beta1.NewArrayOrString(fmt.Sprintf("%g", val.Value().(float64))), nil
	}
	return pipelinev1beta1.ArrayOrString{}, fmt.Errorf("unknown result type %T, expression must evaluate to a string", v)
}
