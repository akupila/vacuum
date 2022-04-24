package cui

import (
	"github.com/daveshanley/vacuum/model"
	ui "github.com/gizak/termui/v3"
	"github.com/gizak/termui/v3/widgets"
	"log"
)

// Dashboard represents the dashboard controlling container
type Dashboard struct {
	C                           chan bool
	run                         bool
	grid                        *ui.Grid
	tabs                        TabbedView
	healthGaugeItems            []ui.GridItem
	categoryHealthGauge         []CategoryGauge
	resultSet                   *model.RuleResultSet
	index                       *model.SpecIndex
	info                        *model.SpecInfo
	selectedTabIndex            int
	ruleCategories              []*model.RuleCategory
	selectedCategory            *model.RuleCategory
	selectedCategoryDescription *ui.GridItem
	selectedRule                *model.Rule
	selectedRuleIndex           int
	selectedViolationIndex      int
	selectedViolation           *model.RuleFunctionResult
	violationViewActive         bool
}

func CreateDashboard(resultSet *model.RuleResultSet, index *model.SpecIndex, info *model.SpecInfo) *Dashboard {
	db := new(Dashboard)
	db.resultSet = resultSet
	db.index = index
	db.info = info
	return db
}

// GenerateTabbedView generates tabs
func (dash *Dashboard) GenerateTabbedView() {
	var labels []string
	for _, cat := range dash.ruleCategories {
		labels = append(labels, cat.Name)
	}
	tv := widgets.NewTabPane(labels...)
	tv.BorderTop = false
	tv.BorderLeft = false
	tv.BorderRight = false
	tv.BorderBottom = true

	dash.tabs = TabbedView{tv: tv, dashboard: dash}
	dash.selectedTabIndex = 0
	dash.selectedCategory = dash.ruleCategories[0]
	dash.tabs.generateDescriptionGridItem()
	dash.tabs.generateRulesInCategory()
	if len(dash.tabs.currentRuleResults.Rules) > 0 {
		dash.selectedRule = dash.tabs.currentRuleResults.Rules[0].Rule
	}
	dash.tabs.generateRuleViolations()
	dash.tabs.setActiveViolation()
	dash.tabs.generateRuleViolationView()

}

// ComposeGauges prepares health gauges for rendering into the main grid.
func (dash *Dashboard) ComposeGauges() {
	var gridItems []ui.GridItem
	for _, gauge := range dash.categoryHealthGauge {
		numCat := float64(len(dash.categoryHealthGauge))
		ratio := 0.8 / (numCat)
		gridItems = append(gridItems, ui.NewRow(ratio, gauge.g))
	}
	dash.healthGaugeItems = gridItems
}

// Render will render the dashboard.
func (dash *Dashboard) Render() {

	if err := ui.Init(); err != nil {
		log.Fatalf("failed to initialize CUI: %v", err)
	}

	dash.run = true
	defer ui.Close()

	// extract categories and calculate coverage.
	var gauges []CategoryGauge
	var cats []*model.RuleCategory
	for _, cat := range model.RuleCategoriesOrdered {
		score := dash.resultSet.CalculateCategoryHealth(cat.Id)
		gauges = append(gauges, NewCategoryGauge(cat.Name, score, cat))
		cats = append(cats, cat)
	}

	// todo: create a formula for this.
	gauges = append(gauges, NewCategoryGauge("Overall Health", 12, model.RuleCategoriesOrdered[0]))

	dash.categoryHealthGauge = gauges
	dash.ruleCategories = cats

	uiEvents := ui.PollEvents()

	dash.grid = ui.NewGrid()
	termWidth, termHeight := ui.TerminalDimensions()
	dash.grid.SetRect(0, 0, termWidth, termHeight)

	dash.GenerateTabbedView()
	dash.ComposeGauges()

	dash.setGrid()
	//dash.tabs.setActiveCategoryIndex(dash.tabs.tv.ActiveTabIndex)

	ui.Render(dash.grid)

	for {
		select {
		case e := <-uiEvents:
			switch e.ID {
			case "q", "<C-c>":
				return
			case "[", "<Left>":
				dash.tabs.tv.FocusLeft()
				dash.tabs.setActiveCategoryIndex(dash.tabs.tv.ActiveTabIndex)
				ui.Clear()
				dash.setGrid()
				ui.Render(dash.grid)
			case "]", "<Right>":
				dash.tabs.tv.FocusRight()
				dash.tabs.setActiveCategoryIndex(dash.tabs.tv.ActiveTabIndex)
				ui.Clear()
				dash.setGrid()
				ui.Render(dash.grid)
			case "j", "<Down>":
				dash.tabs.scrollDown()
				ui.Clear()
				dash.setGrid()
				ui.Render(dash.grid)
			case "k", "<Up>":
				dash.tabs.scrollUp()
				ui.Clear()
				dash.setGrid()
				ui.Render(dash.grid)
			case "<Enter>":
				dash.violationViewActive = true
				ui.Clear()
				dash.setGrid()
				ui.Render(dash.grid)
			case "<Escape>":
				dash.violationViewActive = false
				dash.selectedViolationIndex = 0
				ui.Clear()
				dash.setGrid()
				ui.Render(dash.grid)
			case "<C-d>":
				dash.tabs.rulesList.ScrollHalfPageDown()
			case "<C-u>":
				dash.tabs.rulesList.ScrollHalfPageUp()
			case "<C-f>":
				dash.tabs.rulesList.ScrollPageDown()
			case "<C-b>":
				dash.tabs.rulesList.ScrollPageUp()
			case "g":
				//if previousKey == "g" {
				//	l.ScrollTop()
				//}
			case "<Home>":
				dash.tabs.rulesList.ScrollTop()
			case "G", "<End>":
				dash.tabs.rulesList.ScrollBottom()
			}
			//ui.Clear()
			//dash.setGrid()
			//ui.Render(dash.grid)
		}

	}
}

func (dash *Dashboard) renderActiveTab() {

}

func (dash *Dashboard) setGrid() {
	dash.grid.Set(
		ui.NewRow(1.0,
			ui.NewCol(0.2,
				dash.healthGaugeItems[0], dash.healthGaugeItems[1], dash.healthGaugeItems[2], dash.healthGaugeItems[3],
				dash.healthGaugeItems[4], dash.healthGaugeItems[5], dash.healthGaugeItems[6], dash.healthGaugeItems[7],
				//dash.healthGaugeItems[8],
				ui.NewRow(0.3, NewStatsChart(dash.index, dash.info).bc),
			),
			ui.NewCol(0.01, nil),
			ui.NewCol(0.99,
				ui.NewRow(0.1, dash.tabs.tv),
				ui.NewRow(0.9,
					ui.NewCol(0.5,
						*dash.tabs.descriptionGridItem,
						*dash.tabs.rulesListGridItem,
						*dash.tabs.violationListGridItem,
					),
					ui.NewCol(0.3,
						*dash.tabs.violationViewGridItem,
						*dash.tabs.violationSnippetGridItem,
						*dash.tabs.violationFixGridItem),
				),
			),
		),
	)
}

func getColorForPercentage(percent int) ui.Color {
	if percent <= 30 {
		return ui.ColorRed
	}
	if percent > 30 && percent <= 70 {
		return ui.ColorYellow
	}
	if percent > 70 && percent <= 90 {
		return ui.ColorBlue
	}
	if percent > 80 {
		return ui.ColorGreen
	}
	return ui.ColorClear
}
