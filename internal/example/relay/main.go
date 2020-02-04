package main

import (
    "fmt"
    "github.com/chirino/graphql"
    "github.com/chirino/graphql/relay"
    "github.com/friendsofgo/graphiql"
    "log"
    "net/http"
)

type query struct {
    Name string `json:"name"`
}
func (q *query) Hello() string { return "Hello, " + q.Name }

func main() {
    engine := graphql.New()
    engine.Root = &query{
        Name: "World!",
    }
    err := engine.Schema.Parse(`
        schema {
            query: Query
        }
        type Query {
            name: String!
            hello: String!
        }
    `)
    if err != nil {
        log.Fatal(err)
    }

    http.Handle("/graphql", &relay.Handler{Engine: engine})
    fmt.Println("GraphQL service running at http://localhost:8080/graphql")

    graphiql, _ := graphiql.NewGraphiqlHandler("/graphql")
    http.Handle("/", graphiql)
    fmt.Println("GraphiQL UI running at http://localhost:8080/")

    log.Fatal(http.ListenAndServe(":8080", nil))
}