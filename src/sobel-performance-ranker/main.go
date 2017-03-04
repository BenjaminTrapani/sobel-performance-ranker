package main
import (
	"flag"
	"sobel-performance-ranker/submissioneval"
	"math/big"
)

func main() {
	var rootDir string
	var expectedKnownImagePath string
	var expectedUnknownImagePath string
	var outputDir string
	var maxDiffThreshold big.Float
	var maxDiffThreshString string
	flag.StringVar(&rootDir,"root-dir", ".", "The root directory of sobel outputs")
	flag.StringVar(&expectedKnownImagePath, "exp-known-image", ".", "The path to the expected output of known image")
	flag.StringVar(&expectedUnknownImagePath, "exp-unknown-image", ".", "The path to the expected output of the unknown image")
	flag.StringVar(&outputDir, "output-dir", ".", "The directory to place the generated CSV files in")
	flag.StringVar(&maxDiffThreshString, "max-diff", "1.0", "The maximum acceptable percent difference")
	flag.Parse()

	_, _, err := maxDiffThreshold.Parse(maxDiffThreshString, 10)
	if err != nil {
		panic(err)
	}

	allData, err := submissioneval.BuildInput(rootDir, expectedKnownImagePath, expectedUnknownImagePath)
	if err != nil {
		panic(err)
	}
	if err := submissioneval.SortAndWriteAggregates(outputDir,allData,maxDiffThreshold); err != nil {
		panic(err)
	}
}

