package validation_test

import (
	"os"
	"reflect"
	"sort"
	"testing"

	"encoding/json"

	"github.com/chirino/graphql/internal/validation"
	"github.com/chirino/graphql/qerrors"
	"github.com/chirino/graphql/schema"
)

type Test struct {
	Name   string
	Rule   string
	Schema int
	Query  string
	Errors qerrors.ErrorList
}

func TestValidate(t *testing.T) {
	f, err := os.Open("testdata/tests.json")
	if err != nil {
		t.Fatal(err)
	}

	var testData struct {
		Schemas []string
		Tests   []*Test
	}
	if err := json.NewDecoder(f).Decode(&testData); err != nil {
		t.Fatal(err)
	}

	schemas := make([]*schema.Schema, len(testData.Schemas))
	for i, schemaStr := range testData.Schemas {
		schemas[i] = schema.New()
		if err := schemas[i].Parse(schemaStr); err != nil {
			t.Fatal(err)
		}
	}

	for _, test := range testData.Tests {
		if test.Name != "Validate: Variables are in allowed positions/String over Boolean" {
			continue
		}
		t.Run(test.Name, func(t *testing.T) {
			d := &schema.QueryDocument{}
			err := d.Parse(test.Query)
			if err != nil {
				t.Fatal(err)
			}
			errs := validation.Validate(schemas[test.Schema], d, 0)
			got := qerrors.ErrorList{}
			for _, err := range errs {
				if err.Rule == test.Rule {
					err.Rule = ""
					err.ClearStack()
					got = append(got, err)
				}
			}
			sortLocations(test.Errors)
			sortLocations(got)
			if !reflect.DeepEqual(test.Errors, got) {
				t.Errorf("wrong errors\nexpected: %v\ngot:      %v", test.Errors, got)
			}
		})
	}
}

func sortLocations(errs qerrors.ErrorList) {
	for _, err := range errs {
		locs := err.Locations
		sort.Slice(locs, func(i, j int) bool { return locs[i].Before(locs[j]) })
	}
}
