package easysiteprofiles

import "fmt"

var (
	continentSelectors = []string{"AF", "AN", "AS", "EU", "NA", "OC", "SA"}
	groupSelectors     = []string{"APAC", "EMEA", "LATAM", "DACH", "CIS", "GCC", "NORAM"}
)

type CountryCatalog struct {
	Continents []string `json:"continents"`
	Groups     []string `json:"groups"`
	Countries  []string `json:"countries"`
}

func DefaultCountryCatalog() CountryCatalog {
	return CountryCatalog{
		Continents: append([]string(nil), continentSelectors...),
		Groups:     append([]string(nil), groupSelectors...),
		Countries:  allCountryCodeCandidates(),
	}
}

func allCountryCodeCandidates() []string {
	out := make([]string, 0, 26*26)
	for first := 'A'; first <= 'Z'; first++ {
		for second := 'A'; second <= 'Z'; second++ {
			out = append(out, fmt.Sprintf("%c%c", first, second))
		}
	}
	return out
}
