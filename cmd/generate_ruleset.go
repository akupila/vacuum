// Copyright 2022 Dave Shanley / Quobix
// SPDX-License-Identifier: MIT

package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/daveshanley/vacuum/rulesets"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
	"io/ioutil"
)

func GetGenerateRulesetCommand() *cobra.Command {

	cmd := &cobra.Command{
		Use:     "generate-ruleset",
		Short:   "Generate a vacuum RuleSet",
		Long:    "Generate a YAML ruleset containing 'all', or 'recommended' rules",
		Example: "vacuum generate-ruleset recommended | all <ruleset-output-name>",
		RunE: func(cmd *cobra.Command, args []string) error {

			// check for file args
			if len(args) < 1 {
				errText := "please supply 'recommended' or 'all' and a file path to output the ruleset."
				pterm.Error.Println(errText)
				pterm.Println()
				return errors.New(errText)
			}

			if args[0] != "recommended" && args[0] != "all" {
				errText := fmt.Sprintf("please use 'all' or 'recommended' your choice '%s' is not valid", args[0])
				pterm.Error.Println(errText)
				pterm.Println()
				return errors.New(errText)
			}

			extension := ".yaml"
			reportOutput := "ruleset"

			if len(args) == 2 {
				reportOutput = args[1]
			}

			// read spec and parse to dashboard.
			defaultRuleSets := rulesets.BuildDefaultRuleSets()

			var selectedRuleSet *rulesets.RuleSet

			// default is recommended rules, based on spectral (for now anyway)
			selectedRuleSet = defaultRuleSets.GenerateOpenAPIRecommendedRuleSet()

			if args[0] == "all" {
				selectedRuleSet = defaultRuleSets.GenerateOpenAPIDefaultRuleSet()
			}

			// this bit needs a re-think, but it works for now.
			// because Spectral has an ass backwards schema design, this disco dance here
			// is to re-encode from rules to ruleDefinitions (which is a proxy property)
			encoded, _ := json.Marshal(selectedRuleSet.Rules)
			encodedMap := make(map[string]interface{})
			json.Unmarshal(encoded, &encodedMap)
			selectedRuleSet.RuleDefinitions = encodedMap

			pterm.Info.Printf("Generating RuleSet rules: %s\n\n%s\n", args[0], selectedRuleSet.Description)
			pterm.Println()

			yamlBytes, _ := yaml.Marshal(selectedRuleSet)

			reportOutputName := fmt.Sprintf("%s-%s%s", reportOutput, args[0], extension)

			err := ioutil.WriteFile(reportOutputName, yamlBytes, 0664)

			if err != nil {
				pterm.Error.Printf("Unable to write RuleSet file: '%s': %s\n", reportOutputName, err.Error())
				pterm.Println()
				return err
			}

			pterm.Info.Printf("RuleSet generated for '%s', written to '%s'\n", args[0], reportOutputName)
			pterm.Println()

			return nil
		},
	}
	return cmd
}
