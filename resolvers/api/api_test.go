package api_test

import (
    "github.com/chirino/graphql"
    "github.com/chirino/graphql/resolvers/api"
    "github.com/stretchr/testify/require"
    "io/ioutil"
    "testing"
)

func TestApiResolver(t *testing.T) {
    engine := graphql.New()

    err := api.MountApi(engine, api.ApiResolverOptions{
        Openapi: api.EndpointOptions{
            URL: "k8s.json",
        },
        Logs: ioutil.Discard,
    })
    require.NoError(t, err)

    actual := engine.Schema.String()
    // ioutil.WriteFile("k8s.graphql", []byte(actual), 0644)

    file, err := ioutil.ReadFile("k8s.graphql")
    require.NoError(t, err)
    expected := string(file)

    require.Equal(t, actual, expected)
}