package analyzer

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strconv"
)

// Scholarship represents a single scholarship record.
type Scholarship struct {
	Ano      int
	Unidade  string
	Titulo   string
	Vertente string
	Bolsas   int
}

// ReadScholarships reads the CSV file and returns a slice of Scholarship structs.
func ReadScholarships(filename string) ([]Scholarship, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	// Read header
	if _, err := reader.Read(); err != nil {
		return nil, err
	}

	var scholarships []Scholarship
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		ano, err := strconv.Atoi(record[0])
		if err != nil {
			return nil, err
		}
		bolsas, err := strconv.Atoi(record[4])
		if err != nil {
			return nil, err
		}

		scholarships = append(scholarships, Scholarship{
			Ano:      ano,
			Unidade:  record[1],
			Titulo:   record[2],
			Vertente: record[3],
			Bolsas:   bolsas,
		})
	}
	return scholarships, nil
}

// AnalyzeScholarships performs the analysis of the scholarships.
func AnalyzeScholarships(scholarships []Scholarship) {
	if len(scholarships) == 0 {
		fmt.Println("No scholarships found.")
		return
	}

	totalBolsas := 0
	bolsasPorAno := make(map[int]int)
	bolsasPorUnidade := make(map[string]int)

	for _, s := range scholarships {
		totalBolsas += s.Bolsas
		bolsasPorAno[s.Ano] += s.Bolsas
		bolsasPorUnidade[s.Unidade] += s.Bolsas
	}

	fmt.Printf("Total de bolsas: %d\n", totalBolsas)
	fmt.Println("--------------------")
	fmt.Println("Bolsas por ano:")
	for ano, bolsas := range bolsasPorAno {
		fmt.Printf("- %d: %d\n", ano, bolsas)
	}
	fmt.Println("--------------------")
	fmt.Println("Bolsas por unidade:")
	for unidade, bolsas := range bolsasPorUnidade {
		fmt.Printf("- %s: %d\n", unidade, bolsas)
	}
	fmt.Println("--------------------")
	unidadeMaisBolsas := ""
	maxBolsas := 0
	for unidade, bolsas := range bolsasPorUnidade {
		if bolsas > maxBolsas {
			maxBolsas = bolsas
			unidadeMaisBolsas = unidade
		}
	}
	fmt.Printf("Unidade com mais bolsas: %s (%d bolsas)\n", unidadeMaisBolsas, maxBolsas)
}
