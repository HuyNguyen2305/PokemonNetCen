package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/playwright-community/playwright-go"
)

type Stats struct {
	HP         int `json:"HP"`
	Attack     int `json:"Attack"`
	Defense    int `json:"Defense"`
	Speed      int `json:"Speed"`
	Sp_Attack  int `json:"Sp_Attack"`
	Sp_Defense int `json:"Sp_Defense"`
}

type GenderRatio struct {
	MaleRatio   float32 `json:"MaleRatio"`
	FemaleRatio float32 `json:"FemaleRatio"`
}

type Profile struct {
	Height      float32     `json:"Height"`
	Weight      float32     `json:"Weight"`
	CatchRate   float32     `json:"CatchRate"`
	GenderRatio GenderRatio `json:"GenderRatio"`
	EggGroup    string      `json:"EggGroup"`
	HatchSteps  int         `json:"HatchSteps"`
	Abilities   string      `json:"Abilities"`
}

type DamegeWhenAttacked struct {
	Element     string  `json:"Element"`
	Coefficient float32 `json:"Coefficient"`
}

type Moves struct {
	Name        string `json:"Name"`
	Element     string `json:"Element"`
	Power       string `json:"Power"`
	Acc         int    `json:"Acc"`
	PP          int    `json:"PP"`
	Description string `json:"Description"`
}

type Pokemon struct {
	Name               string               `json:"Name"`
	Elements           []string             `json:"Elements"`
	EV                 int                  `json:"EV"`
	Stats              Stats                `json:"Stats"`
	Profile            Profile              `json:"Profile"`
	DamegeWhenAttacked []DamegeWhenAttacked `json:"DamegeWhenAttacked"`
	EvolutionLevel     int                  `json:"EvolutionLevel"`
	NextEvolution      string               `json:"NextEvolution"`
	Moves              []Moves              `json:"Moves"`
}

const (
	numberOfPokemons = 649
	baseURL          = "https://pokedex.org/#/"
)

var pokemons []Pokemon

func main() {
	crawlPokemonsDriver(numberOfPokemons)
}

func crawlPokemonsDriver(numsOfPokemons int) {
	pw, err := playwright.Run()
	if err != nil {
		log.Fatalf("could not start playwright: %v", err)
	}
	browser, err := pw.Chromium.Launch()
	if err != nil {
		log.Fatalf("could not launch browser: %v", err)
	}

	page, err := browser.NewPage()
	if err != nil {
		log.Fatalf("could not create page: %v", err)
	}

	page.Goto(baseURL)

	for i := range numsOfPokemons {
		// simulate clicking the button to open the pokemon details
		locator := fmt.Sprintf("button.sprite-%d", i+1)
		button := page.Locator(locator).First()
		time.Sleep(500 * time.Millisecond)
		button.Click()

		fmt.Print("Pokemon ", i+1, " ")
		crawlPokemons(page)

		page.Goto(baseURL)
		page.Reload()
	}

	// parse the pokemons variable to json file
	js, err := json.MarshalIndent(pokemons, "", "    ")
	if err != nil {
		log.Fatal(err)
	}
	os.WriteFile("../lib/pokedex.json", js, 0644)

	if err = browser.Close(); err != nil {
		log.Fatalf("could not close browser: %v", err)
	}
	if err = pw.Stop(); err != nil {
		log.Fatalf("could not stop Playwright: %v", err)
	}
}

func crawlPokemons(page playwright.Page) {
	pokemon := Pokemon{}

	stats := Stats{}
	entries, _ := page.Locator("div.detail-panel-content > div.detail-header > div.detail-infobox > div.detail-stats > div.detail-stats-row").All()
	for _, entry := range entries {
		title, _ := entry.Locator("span:not([class])").TextContent()
		switch title {
		case "HP":
			hp, _ := entry.Locator("span.stat-bar > div.stat-bar-fg").TextContent()
			stats.HP, _ = strconv.Atoi(hp)
		case "Attack":
			attack, _ := entry.Locator("span.stat-bar > div.stat-bar-fg").TextContent()
			stats.Attack, _ = strconv.Atoi(attack)
		case "Defense":
			defense, _ := entry.Locator("span.stat-bar > div.stat-bar-fg").TextContent()
			stats.Defense, _ = strconv.Atoi(defense)
		case "Speed":
			speed, _ := entry.Locator("span.stat-bar > div.stat-bar-fg").TextContent()
			stats.Speed, _ = strconv.Atoi(speed)
		case "Sp Atk":
			sp_Attack, _ := entry.Locator("span.stat-bar > div.stat-bar-fg").TextContent()
			stats.Sp_Attack, _ = strconv.Atoi(sp_Attack)
		case "Sp Def":
			sp_Defense, _ := entry.Locator("span.stat-bar > div.stat-bar-fg").TextContent()
			stats.Sp_Defense, _ = strconv.Atoi(sp_Defense)
		default:
			fmt.Println("Unknown title: ", title)
		}
	}
	pokemon.Stats = stats

	name, _ := page.Locator("div.detail-panel > h1.detail-panel-header").TextContent()
	pokemon.Name = name

	genderRatio := GenderRatio{}
	profile := Profile{}
	entries, _ = page.Locator("div.detail-panel-content > div.detail-below-header > div.monster-minutia").All()
	for _, entry := range entries {
		title1, _ := entry.Locator("strong:not([class]):nth-child(1)").TextContent()
		stat1, _ := entry.Locator("span:not([class]):nth-child(2)").TextContent()
		switch title1 {
		case "Height:":
			heights := strings.Split(stat1, " ")
			height, _ := strconv.ParseFloat(heights[0], 32)
			profile.Height = float32(height)
		case "Catch Rate:":
			catchRates := strings.Split(stat1, "%")
			catchRate, _ := strconv.ParseFloat(catchRates[0], 32)
			profile.CatchRate = float32(catchRate)
		case "Egg Groups:":
			profile.EggGroup = stat1
		case "Abilities:":
			profile.Abilities = stat1
		}

		title2, _ := entry.Locator("strong:not([class]):nth-child(3)").TextContent()
		stat2, _ := entry.Locator("span:not([class]):nth-child(4)").TextContent()
		switch title2 {
		case "Weight:":
			weights := strings.Split(stat2, " ")
			weight, _ := strconv.ParseFloat(weights[0], 32)
			profile.Weight = float32(weight)
		case "Gender Ratio:":
			if stat2 == "N/A" {
				genderRatio.MaleRatio = 0
				genderRatio.FemaleRatio = 0
			} else {
				ratios := strings.Split(stat2, " ")

				maleRatios := strings.Split(ratios[0], "%")
				maleRatio, _ := strconv.ParseFloat(maleRatios[0], 32)
				genderRatio.MaleRatio = float32(maleRatio)

				femaleRatios := strings.Split(ratios[2], "%")
				femaleRatio, _ := strconv.ParseFloat(femaleRatios[0], 32)
				genderRatio.FemaleRatio = float32(femaleRatio)
			}

			profile.GenderRatio = genderRatio
		case "Hatch Steps:":
			profile.HatchSteps, _ = strconv.Atoi(stat2)
		}
	}
	pokemon.Profile = profile

	damegeWhenAttacked := []DamegeWhenAttacked{}
	entries, _ = page.Locator("div.when-attacked > div.when-attacked-row").All()
	for _, entry := range entries {
		element1, _ := entry.Locator("span.monster-type:nth-child(1)").TextContent()
		coefficient1, _ := entry.Locator("span.monster-multiplier:nth-child(2)").TextContent()
		coefficients1 := strings.Split(coefficient1, "x")
		coef1, _ := strconv.ParseFloat(coefficients1[0], 32)

		element2, _ := entry.Locator("span.monster-type:nth-child(3)").TextContent()
		coefficient2, _ := entry.Locator("span.monster-multiplier:nth-child(4)").TextContent()
		coefficients2 := strings.Split(coefficient2, "x")
		coef2, _ := strconv.ParseFloat(coefficients2[0], 32)

		damegeWhenAttacked = append(damegeWhenAttacked, DamegeWhenAttacked{Element: element1, Coefficient: float32(coef1)})
		damegeWhenAttacked = append(damegeWhenAttacked, DamegeWhenAttacked{Element: element2, Coefficient: float32(coef2)})
	}
	pokemon.DamegeWhenAttacked = damegeWhenAttacked

	entries, _ = page.Locator("div.evolutions > div.evolution-row").All()
	for _, entry := range entries {
		evolutionLabel, _ := entry.Locator("div.evolution-label > span").TextContent()
		evolutionLabels := strings.Split(evolutionLabel, " ")

		if evolutionLabels[0] == name {
			evolutionLevels := strings.Split(evolutionLabels[len(evolutionLabels)-1], ".")
			evolutionLevel, _ := strconv.Atoi(evolutionLevels[0])
			pokemon.EvolutionLevel = evolutionLevel

			nextEvolution := evolutionLabels[3]
			pokemon.NextEvolution = nextEvolution
		}
	}

	moves := []Moves{}
	entries, _ = page.Locator("div.monster-moves > div.moves-row").All()
	for _, entry := range entries {
		// simulate clicking the expand button in the move rows
		expandButton := page.Locator("div.moves-inner-row > button.dropdown-button").First()
		expandButton.Click()

		name, _ := entry.Locator("div.moves-inner-row > span:nth-child(2)").TextContent()
		element, _ := entry.Locator("div.moves-inner-row > span.monster-type").TextContent()

		powers, _ := entry.Locator("div.moves-row-detail > div.moves-row-stats > span:nth-child(1)").TextContent()
		power := strings.Split(powers, ":")

		acc, _ := entry.Locator("div.moves-row-detail > div.moves-row-stats > span:nth-child(2)").TextContent()
		accs := strings.Split(acc, ":")
		accValue := strings.Split(accs[1], "%")
		accInt, _ := strconv.Atoi(accValue[0])

		pps, _ := entry.Locator("div.moves-row-detail > div.moves-row-stats > span:nth-child(3)").TextContent()
		ppVal := strings.Split(pps, ":")
		pp, _ := strconv.Atoi(ppVal[1])

		description, _ := entry.Locator("div.moves-row-detail > div.move-description").TextContent()

		moves = append(moves, Moves{Name: name, Element: element, Power: power[1], Acc: accInt, PP: pp, Description: description})
	}
	pokemon.Moves = moves

	entries, _ = page.Locator("div.detail-types > span.monster-type").All()
	for _, entry := range entries {
		element, _ := entry.TextContent()
		pokemon.Elements = append(pokemon.Elements, element)
	}

	fmt.Println(name, ": ", profile)

	pokemons = append(pokemons, pokemon)
}
