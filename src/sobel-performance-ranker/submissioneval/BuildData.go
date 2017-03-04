package submissioneval

import (
	"io/ioutil"
	"path"
	"os"
	"math/big"
	"strings"
	"fmt"
	"os/exec"
	"bytes"
)

/**
Directory structure to read output from
-husky_id
--trial 1
---known-image
----output_image
----stdout.txt
----stderr.txt
---unknown-image
----output_image.ppm
----stdout.txt
----stderr.txt
--trial 2 same as trial 1
 */

type ExecutionTimes struct {
	//all durations in ms
	KernelExecutionTime big.Float `csv:"kernel-exec-ms"`
	TotalExecutionTime  big.Float `csv:"total-exec-ms"`
}

type trialData struct {
	isKnownImage bool
	diffFromKnown big.Float
	ExecutionTimes
	//only include stderr if file is non-empty
	stdErr *string
}

type AggregatePerformance struct {
	StudentID string `csv:"student-id"`
	MaxDiff big.Float `csv:"max-diff"`
	ExecutionTimes
	StdErr string `csv:"std-err"`
	IsKnownImage bool `csv:"known-image"`
}

func computeAggregateForTrials(inputData []trialData) AggregatePerformance{
	totalKernelTime := big.Float{}
	totalExecTime := big.Float{}
	result := AggregatePerformance{}
	largestDiff := big.Float{}
	for _, data := range(inputData) {
		totalKernelTime.Add(&totalKernelTime, &data.ExecutionTimes.KernelExecutionTime)
		totalExecTime.Add(&totalExecTime, &data.ExecutionTimes.TotalExecutionTime)
		if (data.stdErr != nil) {
			result.StdErr = result.StdErr + *data.stdErr + ";"
		}
		if largestDiff.Cmp(&data.diffFromKnown) < 0 {
			largestDiff.Set(&data.diffFromKnown)
		}
	}
	floatRowLen := float64(len(inputData))
	result.KernelExecutionTime = *totalKernelTime.Quo(&totalKernelTime,big.NewFloat(floatRowLen))
	result.TotalExecutionTime = *totalExecTime.Quo(&totalExecTime,big.NewFloat(float64(floatRowLen)))
	result.MaxDiff = largestDiff
	return result
}

func extractNamesFromDirs(dirs []os.FileInfo) []string {
	nameResults := make([]string, 0, len(dirs))
	for _, dir := range dirs {
		dirString := dir.Name()
		firstChar := dirString[0]
		if string(firstChar) != "." {
			nameResults = append(nameResults, dirString)
		}
	}
	return nameResults
}

func getAllStudentIDs(rootDir string) ([]string, error) {
	dirs, err := ioutil.ReadDir(rootDir)
	if err != nil {
		return nil, err
	}
	return extractNamesFromDirs(dirs), nil
}

func getTrialsForStudentID(studentID, rootDir string) ([]string, error) {
	dirs, err := ioutil.ReadDir(path.Join(rootDir, studentID))
	if err != nil {
		return nil, err
	}
	return extractNamesFromDirs(dirs), nil
}

func readOutputOrError(pathToTrialDir, fileName string) (string, error) {
	fileBytes, err := ioutil.ReadFile(path.Join(pathToTrialDir, fileName))
	if err != nil {
		return "", err
	}
	return string(fileBytes), nil
}

type metricNotFoundError struct {
	description string
}

func (m metricNotFoundError) Error() string{
	return m.description
}

func getExecutionTimes(outputPath string) (ExecutionTimes, error){
	stdOut, err := readOutputOrError(outputPath, "stdout.txt")
	if err != nil {
		return ExecutionTimes{}, err
	}

	spaceSeperatedElems := strings.Fields(stdOut)
	const kernelExecToken = "Kernel"
	const totalExecToken = "Total"

	var kernelExecTime, totalExecTime *big.Float
	initFloat := func() *big.Float {
		result := &big.Float{}
		result.SetPrec(8)
		return result
	}

	for idx, elem := range spaceSeperatedElems {
		switch elem {
		case kernelExecToken:
			kernelExecTime = initFloat()
			kernelExecTime.Parse(spaceSeperatedElems[idx+3], 10)
		case totalExecToken:
			totalExecTime = initFloat()
			totalExecTime.Parse(spaceSeperatedElems[idx+4], 10)
		}
	}

	if totalExecTime == nil {
		return ExecutionTimes{}, metricNotFoundError{fmt.Sprintf("Unable to find total execution time in stdout at path %s", outputPath)}
	}
	if kernelExecTime == nil {
		return ExecutionTimes{}, metricNotFoundError{fmt.Sprintf("Unable to find kernel execution time in stdout at path %s", outputPath)}
	}

	return ExecutionTimes{KernelExecutionTime: *kernelExecTime, TotalExecutionTime: *totalExecTime}, nil
}

func getDifferenceToExpectedImage(outputImagePath, expectedImageFilePath string) (big.Float, error) {
	//diffimg --batch outputImagePath expectedImageFilePath
	// ErrorPercent = 3.9e-05 (threshold = 0)
	stdoutWriter := bytes.NewBuffer([]byte{})
	diffImgCommand := exec.Command("diffimg", "--batch", outputImagePath, expectedImageFilePath)
	diffImgCommand.Stdout = stdoutWriter
	err := diffImgCommand.Run()
	if err != nil {
		return big.Float{}, err
	}
	spaceSeperatedElems := strings.Fields(stdoutWriter.String())
	for idx, elem := range spaceSeperatedElems {
		if elem == "ErrorPercent"{
			result := big.Float{}
			result.Parse(spaceSeperatedElems[idx+2], 10)
			return result, nil
		}
	}
	return big.Float{}, nil
}

func getTrialDataForImage(trialPath string, knownImage bool, expectedImage string) (trialData, error) {
	var imageDirName string
	if knownImage {
		imageDirName = "known-image"
	}else {
		imageDirName = "unknown-image"
	}
	trialPath = path.Join(trialPath, imageDirName)

	result := trialData{isKnownImage: knownImage}

	stdErr, err := readOutputOrError(trialPath, "stderr.txt")
	if err != nil {
		return trialData{}, err
	}
	if len(stdErr) > 0 {
		result.stdErr = &stdErr
	}

	execTimes, err := getExecutionTimes(trialPath)
	if err != nil {
		switch err.(type){
		case metricNotFoundError:
			if result.stdErr == nil {
				result.stdErr = new(string)
			}
			*result.stdErr = *result.stdErr + err.Error() + ";"
			return result, nil
		default:
			return trialData{}, err
		}
	}
	result.ExecutionTimes = execTimes

	diffFromExpected, err := getDifferenceToExpectedImage(path.Join(trialPath, "output_image.ppm"), expectedImage)
	if err != nil {
		return trialData{}, err
	}
	result.diffFromKnown = diffFromExpected
	return result, nil
}

func getTrialData(trialPath, expectedKnownImagePath, expectedUnknownImagePath string) ([]trialData, error) {
	result := make([]trialData, 2)

	knownImageData, err := getTrialDataForImage(trialPath, true, expectedKnownImagePath)
	if err != nil {
		return nil, err
	}
	unknownImageData, err := getTrialDataForImage(trialPath, false, expectedUnknownImagePath)
	if err != nil {
		return nil, err
	}

	result[0] = knownImageData
	result[1] = unknownImageData
	return result, nil
}

func BuildInput(rootDir, expectedKnownImagePath, expectedUnknownImagePath string) ([]AggregatePerformance, error) {
	allStudents, err := getAllStudentIDs(rootDir)
	if err != nil {
		return nil, err
	}

	result := []AggregatePerformance{}
	for _, studentID := range (allStudents) {
		trials, err := getTrialsForStudentID(studentID,rootDir)
		if err != nil {
			fmt.Printf("Skipping non-directory entry in root dir %s\n", studentID)
			continue
		}
		if err != nil {
			return nil, err
		}
		knownTrialDataPerStudent := []trialData{}
		unknownTrialDataPerStudent := []trialData{}
		for _, trial := range(trials) {
			trialData, err := getTrialData(path.Join(rootDir, studentID, trial), expectedKnownImagePath, expectedUnknownImagePath)
			if err != nil {
				return nil, err
			}
			knownTrialDataPerStudent = append(knownTrialDataPerStudent, trialData[0])
			unknownTrialDataPerStudent = append(unknownTrialDataPerStudent, trialData[1])
		}
		knownTrialAggs := computeAggregateForTrials(knownTrialDataPerStudent)
		knownTrialAggs.StudentID = studentID
		knownTrialAggs.IsKnownImage = true
		unknownTrialAggs := computeAggregateForTrials(unknownTrialDataPerStudent)
		unknownTrialAggs.StudentID = studentID
		result = append(result, knownTrialAggs, unknownTrialAggs)
	}
	return result, nil
}