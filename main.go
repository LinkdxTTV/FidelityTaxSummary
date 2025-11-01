package main

import (
	"encoding/csv"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

type Transaction struct {
	Symbol       string
	Security     string
	Quantity     float64
	DateAcquired time.Time
	DateSold     time.Time
	Proceeds     float64
	CostBasis    float64
	ShortTermNet float64
	LongTermNet  float64
}

var AllTransactions []Transaction = []Transaction{}
var ShortTermNets []Transaction = []Transaction{}
var LongTermNets []Transaction = []Transaction{}

var (
	ErrUnparsedLine error = fmt.Errorf("line did not parse")
)

func main() {
	// Check if a filename was provided as command line argument
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run main.go <csv-file-path>")
		return
	}

	filename := os.Args[1]
	file, err := os.Open(filename)
	if err != nil {
		fmt.Printf("Error opening file: %v\n", err)
		return
	}
	defer file.Close()

	fmt.Println("File opened successfully")

	reader := csv.NewReader(file)
	reader.FieldsPerRecord = -1
	records, err := reader.ReadAll()
	if err != nil {
		fmt.Println("Error reading CSV:", err)
		return
	}

	for _, record := range records {
		transaction, err := parseRecordToTranscation(record)
		if err != nil {
			// Errors are expected as many lines that are not transactions wont parse
			fmt.Println("did not parse record:", record)
			continue
		} else {
			AllTransactions = append(AllTransactions, transaction)
			if transaction.ShortTermNet == 0 {
				LongTermNets = append(LongTermNets, transaction)
			} else {
				ShortTermNets = append(ShortTermNets, transaction)
			}
		}
	}

	// Sort all arrays
	sort.SliceStable(AllTransactions, func(i, j int) bool {
		return AllTransactions[i].DateSold.Before(AllTransactions[j].DateSold)
	})
	sort.SliceStable(ShortTermNets, func(i, j int) bool {
		return ShortTermNets[i].DateSold.Before(ShortTermNets[j].DateSold)
	})
	sort.SliceStable(LongTermNets, func(i, j int) bool {
		return LongTermNets[i].DateSold.Before(LongTermNets[j].DateSold)
	})

	// Separate into quarters
	type Quarter struct {
		Transactions []Transaction
	}

	Quarters := []Quarter{}
	for range 4 {
		Quarters = append(Quarters, Quarter{Transactions: []Transaction{}})
	}

	fmt.Println(len(Quarters))

	// Print out information
	fmt.Println("============ ALL TRANSACTIONS SORTED BY SALE DATE, SPLIT BY QUARTER ============\n")
	for _, t := range AllTransactions {
		Quarters[determineQuarterFromDate(t.DateSold)].Transactions = append(Quarters[determineQuarterFromDate(t.DateSold)].Transactions, t)
	}

	for i, q := range Quarters {
		var quarterShortTermNet float64
		var quarterLongTermNet float64
		for _, t := range q.Transactions {
			fmt.Printf("Sold %.1f shares of %s on %s for a %s \n", t.Quantity, t.Symbol, t.DateSold.Format("01/02/2006"), gainPhrase(t))
			quarterShortTermNet += t.ShortTermNet
			quarterLongTermNet += t.LongTermNet
		}
		fmt.Println(fmt.Sprintf("============ End of Quarter %d || Short Term Net: %.2f, Long Term Net: %.2f \n", i+1, quarterShortTermNet, quarterLongTermNet))
	}
}

func gainPhrase(t Transaction) string {
	if t.ShortTermNet == 0 {
		if t.LongTermNet < 0 {
			return fmt.Sprintf("long term loss of %.2f USD", t.LongTermNet)
		}
		return fmt.Sprintf("long term gain of %.2f USD", t.LongTermNet)
	} else {
		if t.ShortTermNet < 0 {
			return fmt.Sprintf("short term loss of %.2f USD", t.ShortTermNet)

		}
		return fmt.Sprintf("short term gain of %.2f USD", t.ShortTermNet)
	}
}

// Assumes dollars
func parseRecordToTranscation(record []string) (Transaction, error) {
	if len(record) != 10 {
		return Transaction{}, ErrUnparsedLine
	}
	if record[0] == "" {
		return Transaction{}, ErrUnparsedLine
	}

	return Transaction{
		Symbol:       record[0],
		Security:     record[1],
		Quantity:     sanitizeMoneySymbolsAndParseToFloat64(record[2]),
		DateAcquired: parseTime(record[3]),
		DateSold:     parseTime(record[4]),
		Proceeds:     sanitizeMoneySymbolsAndParseToFloat64(record[5]),
		CostBasis:    sanitizeMoneySymbolsAndParseToFloat64(record[6]),
		ShortTermNet: sanitizeMoneySymbolsAndParseToFloat64(record[7]),
		LongTermNet:  sanitizeMoneySymbolsAndParseToFloat64(record[8]),
	}, nil
}

func sanitizeMoneySymbolsAndParseToFloat64(in string) float64 {
	in = strings.TrimSpace(in)
	if in == "--" || in == "" {
		return 0
	}
	// Remove commas
	in = strings.ReplaceAll(in, ",", "")
	re := regexp.MustCompile(`-?\$?([0-9]+(?:\.[0-9]+)?)`)
	match := re.FindStringSubmatch(in)

	if len(match) < 2 {
		fmt.Println("No number found")
		panic("")
	}

	// match[1] holds just the numeric part (e.g., "275.27")
	numStr := match[1]

	val, err := strconv.ParseFloat(numStr, 64)
	if err != nil {
		fmt.Println("Parse error:", err)
		panic("")
	}

	// Apply negative if the string starts with "-"
	if in[0] == '-' {
		val = -val
	}
	return val
}

func parseTime(in string) time.Time {
	if in == "Unknown" {
		return time.Time{}
	}
	out, err := time.Parse("01/02/2006", in)
	if err != nil {
		panic(fmt.Sprintf("cannot parse %s to time", in))
	}

	return out
}

func determineQuarterFromDate(date time.Time) int {
	if date.Month() <= 3 {
		return 0
	}
	if date.Month() <= 6 {
		return 1
	}
	if date.Month() <= 9 {
		return 2
	}
	return 3
}
