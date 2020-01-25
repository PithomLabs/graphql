package graphql

import (
    "bufio"
    "bytes"
    "context"
    "encoding/json"
    "fmt"
    "github.com/chirino/graphql/errors"
    "github.com/chirino/graphql/internal/exec"
    "github.com/chirino/graphql/internal/introspection"
    "github.com/chirino/graphql/internal/query"
    "github.com/chirino/graphql/internal/validation"
    "github.com/chirino/graphql/schema"
)

type EngineRequest struct {
    Query         string                 `json:"query"`
    OperationName string                 `json:"operationName"`
    Variables     map[string]interface{} `json:"variables"`
}

// Response represents a typical response of a GraphQL server. It may be encoded to JSON directly or
// it may be further processed to a custom response type, for example to include custom error data.
// Errors are intentionally serialized first based on the advice in https://github.com/facebook/graphql/commit/7b40390d48680b15cb93e02d46ac5eb249689876#diff-757cea6edf0288677a9eea4cfc801d87R107
type EngineResponse struct {
    Data       json.RawMessage        `json:"data,omitempty"`
    Errors     []*errors.QueryError   `json:"errors,omitempty"`
    Extensions map[string]interface{} `json:"extensions,omitempty"`
}

func (r *EngineResponse) Error() error {
    errs := []error{}
    for _, err := range r.Errors {
        errs = append(errs, err)
    }
    return errors.Multi(errs...)
}

// Execute the given request.
func (engine *Engine) Execute(ctx context.Context, request *EngineRequest, root interface{}) *EngineResponse {

    doc, qErr := query.Parse(request.Query)
    if qErr != nil {
        return &EngineResponse{Errors: []*errors.QueryError{qErr}}
    }

    validationFinish := engine.ValidationTracer.TraceValidation()
    errs := validation.Validate(engine.Schema, doc, engine.MaxDepth)
    validationFinish(errs)

    if len(errs) != 0 {
        return &EngineResponse{Errors: errs}
    }

    op, err := getOperation(doc, request.OperationName)
    if err != nil {
        return &EngineResponse{Errors: []*errors.QueryError{errors.Errorf("%s", err)}}
    }

    varTypes := make(map[string]*introspection.Type)
    for _, v := range op.Vars {
        t, err := schema.ResolveType(v.Type, engine.Schema.Resolve)
        if err != nil {
            return &EngineResponse{Errors: []*errors.QueryError{err}}
        }
        varTypes[v.Name.Name] = introspection.WrapType(t)
    }

    if root == nil {
        root = engine.Root
    }

    traceContext, finish := engine.Tracer.TraceQuery(ctx, request.Query, request.OperationName, request.Variables, varTypes)
    out := bytes.Buffer{}

    r := exec.Execution{
        Schema:          engine.Schema,
        Tracer:          engine.Tracer,
        Logger:          engine.Logger,
        ResolverFactory: engine.ResolverFactory,
        Doc:             doc,
        Operation:       op,
        Vars:            request.Variables,
        VarTypes:        varTypes,
        Limiter:         make(chan byte, engine.MaxParallelism),
        Context:         traceContext,
        Root:            root,
        Out:             bufio.NewWriter(&out),
    }

    errs = r.Execute()
    finish(errs)

    if len(errs) > 0 {
        return &EngineResponse{
            Errors: errs,
        }
    }

    return &EngineResponse{
        Data: out.Bytes(),
    }
}

func getOperation(document *query.Document, operationName string) (*query.Operation, error) {
    if len(document.Operations) == 0 {
        return nil, fmt.Errorf("no operations in query document")
    }

    if operationName == "" {
        if len(document.Operations) > 1 {
            return nil, fmt.Errorf("more than one operation in query document and no operation name given")
        }
        for _, op := range document.Operations {
            return op, nil // return the one and only operation
        }
    }

    op := document.Operations.Get(operationName)
    if op == nil {
        return nil, fmt.Errorf("no operation with name %q", operationName)
    }
    return op, nil
}

func (engine *Engine) Exec(ctx context.Context, result interface{}, query string, args ...interface{}) error {
    variables := map[string]interface{}{}
    for i := 0; i+1 < len(args); i += 2 {
        variables[args[i].(string)] = args[i+1]
    }

    request := EngineRequest{Query: query, Variables: variables}
    response := engine.Execute(ctx, &request, engine.Root)

    if result != nil {
        switch result := result.(type) {
        case *[]byte:
            *result = response.Data
        case *string:
            *result = string(response.Data)
        default:
            err := json.Unmarshal(response.Data, result)
            if err != nil {
                return errors.Multi(err, response.Error())
            }
        }
    }
    return response.Error()
}