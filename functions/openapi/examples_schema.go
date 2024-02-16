// Copyright 2023-2024 Princess Beef Heavy Industries, LLC / Dave Shanley
// https://pb33f.io

package openapi

import (
	"fmt"
	"github.com/daveshanley/vacuum/model"
	vacuumUtils "github.com/daveshanley/vacuum/utils"
	"github.com/pb33f/doctor/model/high/base"
	"github.com/pb33f/libopenapi-validator/schema_validation"
	v3Base "github.com/pb33f/libopenapi/datamodel/high/base"
	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/orderedmap"
	"github.com/pb33f/libopenapi/utils"
	"github.com/sourcegraph/conc"
	"gopkg.in/yaml.v3"
	"strings"
)

// ExamplesSchema will check anything that has an example, has a schema and it's valid.
type ExamplesSchema struct {
}

// GetSchema returns a model.RuleFunctionSchema defining the schema of the ComponentDescription rule.
func (es ExamplesSchema) GetSchema() model.RuleFunctionSchema {
	return model.RuleFunctionSchema{Name: "examples_missing"}
}

var bannedErrors = []string{"if-then failed", "if-else failed", "allOf failed", "oneOf failed"}

// RunRule will execute the ComponentDescription rule, based on supplied context and a supplied []*yaml.Node slice.
func (es ExamplesSchema) RunRule(_ []*yaml.Node, context model.RuleFunctionContext) []model.RuleFunctionResult {

	var results []model.RuleFunctionResult

	if context.DrDocument == nil {
		return results
	}

	buildResult := func(message, path string, key, node *yaml.Node, component base.AcceptsRuleResults) model.RuleFunctionResult {
		result := model.RuleFunctionResult{
			Message:   message,
			StartNode: key,
			EndNode:   vacuumUtils.BuildEndNode(key),
			Path:      path,
			Rule:      context.Rule,
		}
		component.AddRuleFunctionResult(base.ConvertRuleResult(&result))
		return result
	}
	wg := conc.WaitGroup{}

	validator := schema_validation.NewSchemaValidator()
	validateSchema := func(iKey *int,
		sKey, label string,
		s *base.Schema,
		obj base.AcceptsRuleResults,
		node *yaml.Node,
		keyNode *yaml.Node,
		example any) []model.RuleFunctionResult {

		var rx []model.RuleFunctionResult
		if s != nil && s.Value != nil {
			valid, validationErrors := validator.ValidateSchemaObject(s.Value, example)
			if !valid {
				var path string
				if iKey == nil && sKey == "" {
					path = fmt.Sprintf("%s.%s", obj.(base.Foundational).GenerateJSONPath(), label)
				}
				if iKey != nil && sKey == "" {
					path = fmt.Sprintf("%s.%s[%d]", obj.(base.Foundational).GenerateJSONPath(), label, *iKey)
				}
				if iKey == nil && sKey != "" {
					path = fmt.Sprintf("%s.%s['%s']", obj.(base.Foundational).GenerateJSONPath(), label, sKey)
				}
				for _, r := range validationErrors {
					for _, err := range r.SchemaValidationErrors {
						result := buildResult(vacuumUtils.SuppliedOrDefault(context.Rule.Message, err.Reason),
							path, keyNode, node, obj)

						banned := false
						for g := range bannedErrors {
							if strings.Contains(err.Reason, bannedErrors[g]) {
								banned = true
								continue
							}
						}
						if !banned {
							rx = append(rx, result)
						}
					}
				}
			}
		}
		return rx
	}

	if context.DrDocument != nil && context.DrDocument.Schemas != nil {
		for i := range context.DrDocument.Schemas {
			s := context.DrDocument.Schemas[i]
			wg.Go(func() {
				if s.Value.Examples != nil {
					for x, ex := range s.Value.Examples {

						isRef, _, _ := utils.IsNodeRefValue(ex)
						if isRef {
							// extract node
							fNode, _, _, _ := low.LocateRefNodeWithContext(s.Value.ParentProxy.GoLow().GetContext(),
								ex, context.Index)
							if fNode != nil {
								ex = fNode
							} else {
								continue
							}
						}

						var example map[string]interface{}
						_ = ex.Decode(&example)

						result := validateSchema(&x, "", "examples",
							s, s, s.Value.GoLow().Examples.Value[x].ValueNode,
							s.Value.GoLow().Examples.GetKeyNode(), example)

						if result != nil {
							results = append(results, result...)
						}
					}
				}

				if s.Value.Example != nil {

					isRef, _, _ := utils.IsNodeRefValue(s.Value.Example)
					ref := s.Value.Example
					if isRef {
						// extract node
						fNode, _, _, _ := low.LocateRefNodeWithContext(s.Value.ParentProxy.GoLow().GetContext(),
							s.Value.Example, context.Index)
						if fNode != nil {
							ref = fNode
						}
					}

					var example interface{}
					_ = ref.Decode(&example)

					result := validateSchema(nil, "", "example", s, s, s.Value.Example,
						s.Value.GoLow().Example.GetKeyNode(), example)
					if result != nil {
						results = append(results, result...)
					}
				}
			})
		}
	}

	parseExamples := func(s *base.Schema,
		obj base.AcceptsRuleResults,
		examples *orderedmap.Map[string,
			*v3Base.Example]) []model.RuleFunctionResult {

		var rx []model.RuleFunctionResult
		for examplesPairs := examples.First(); examplesPairs != nil; examplesPairs = examplesPairs.Next() {

			example := examplesPairs.Value()
			exampleKey := examplesPairs.Key()

			var ex any
			if example.Value != nil {
				_ = example.Value.Decode(&ex)
				result := validateSchema(nil, exampleKey, "examples", s, obj, example.Value, example.GoLow().KeyNode, ex)
				if result != nil {
					rx = append(rx, result...)
				}
			}
		}
		return rx
	}

	parseExample := func(s *base.Schema, node, key *yaml.Node) []model.RuleFunctionResult {

		var rx []model.RuleFunctionResult
		var ex any
		_ = node.Decode(&ex)

		result := validateSchema(nil, "", "example", s, s, node, key, ex)
		if result != nil {
			rx = append(rx, result...)
		}
		return rx
	}

	if context.DrDocument != nil && context.DrDocument.Parameters != nil {
		for i := range context.DrDocument.Parameters {
			p := context.DrDocument.Parameters[i]
			wg.Go(func() {
				if p.Value.Examples.Len() >= 1 && p.SchemaProxy != nil {
					results = append(results, parseExamples(p.SchemaProxy.Schema, p, p.Value.Examples)...)
				} else {
					if p.Value.Example != nil && p.SchemaProxy != nil {
						results = append(results, parseExample(p.SchemaProxy.Schema, p.Value.Example,
							p.Value.GoLow().Example.GetKeyNode())...)
					}
				}
			})
		}
	}

	if context.DrDocument != nil && context.DrDocument.Headers != nil {
		for i := range context.DrDocument.Headers {
			h := context.DrDocument.Headers[i]
			wg.Go(func() {
				if h.Value.Examples.Len() >= 1 && h.SchemaProxy != nil {
					results = append(results, parseExamples(h.SchemaProxy.Schema, h, h.Value.Examples)...)
				} else {
					if h.Value.Example != nil && h.SchemaProxy != nil {
						results = append(results, parseExample(h.SchemaProxy.Schema, h.Value.Example,
							h.Value.GoLow().Example.GetKeyNode())...)
					}
				}
			})
		}
	}

	if context.DrDocument != nil && context.DrDocument.MediaTypes != nil {
		for i := range context.DrDocument.MediaTypes {
			mt := context.DrDocument.MediaTypes[i]
			wg.Go(func() {
				if mt.Value.Examples.Len() >= 1 && mt.SchemaProxy != nil {
					results = append(results, parseExamples(mt.SchemaProxy.Schema, mt, mt.Value.Examples)...)
				} else {
					if mt.Value.Example != nil && mt.SchemaProxy != nil {
						results = append(results, parseExample(mt.SchemaProxy.Schema, mt.Value.Example,
							mt.Value.GoLow().Example.GetKeyNode())...)
					}
				}
			})
		}
	}
	wg.Wait()
	return results
}
