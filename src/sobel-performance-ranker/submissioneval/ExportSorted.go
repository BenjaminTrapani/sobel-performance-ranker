package submissioneval

import (
	"math/big"
	"sort"
	"os"
	"path"
	"github.com/gocarina/gocsv"
)

type ExportedAggregatePerformance struct {
	AggregatePerformance
	OutputIsValid bool `csv:"valid-output"`
}

type AggregatePerformanceSlice []ExportedAggregatePerformance
func (aggs AggregatePerformanceSlice) Len() int {
	return len(aggs)
}
func (aggs AggregatePerformanceSlice) Swap(i, j int) {
	aggs[i], aggs[j] = aggs[j], aggs[i]
}
func (aggs AggregatePerformanceSlice) Less(i, j int) bool {
	if aggs[i].OutputIsValid != aggs[j].OutputIsValid {
		if aggs[j].OutputIsValid {
			return false
		}else {
			return true
		}
	}else {
		return aggs[i].KernelExecutionTime.Cmp(&aggs[j].KernelExecutionTime) < 0
	}
}

func convertUnexportedToExportedWithFilter(unexportedAggs []AggregatePerformance,
	shouldInclude func(AggregatePerformance)bool, diffThreshold big.Float) AggregatePerformanceSlice {
	resultsExported := make(AggregatePerformanceSlice, 0, len(unexportedAggs))
	for idx := range unexportedAggs {
		if shouldInclude(unexportedAggs[idx]) {
			isValid := true
			if len(unexportedAggs[idx].StdErr) != 0 || unexportedAggs[idx].MaxDiff.Cmp(&diffThreshold) > 0 {
				isValid = false
			}
			resultRow := ExportedAggregatePerformance{AggregatePerformance: unexportedAggs[idx],
				OutputIsValid: isValid}
			resultsExported = append(resultsExported, resultRow)
		}
	}
	sort.Sort(resultsExported)
	return resultsExported
}

func writeSortedOutputToCSV(outputPath, fileName string, values AggregatePerformanceSlice) error {
	fo, err := os.Create(path.Join(outputPath, fileName))
	if err != nil {
		return err
	}
	defer func() error {
		if err := fo.Close(); err != nil {
			return err
		}
		return nil
	}()
	return gocsv.Marshal(values, fo)
}

func SortAndWriteAggregates(outputPath string, unexportedAggs []AggregatePerformance, diffThreshold big.Float) error {
	resultForKnownImageSlice := convertUnexportedToExportedWithFilter(unexportedAggs, func(val AggregatePerformance) bool {
		return val.IsKnownImage
	}, diffThreshold)
	resultForUnknownImageSlice := convertUnexportedToExportedWithFilter(unexportedAggs, func(val AggregatePerformance) bool {
		return !val.IsKnownImage
	}, diffThreshold)

	if err := writeSortedOutputToCSV(outputPath, "known-image-ranks.csv", resultForKnownImageSlice); err != nil {
		return err
	}
	if err := writeSortedOutputToCSV(outputPath, "unknown-image-ranks.csv", resultForUnknownImageSlice); err != nil {
		return err
	}
	return nil
}
