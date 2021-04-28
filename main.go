// You can edit this code!
// Click here and start typing.
package main

import (
	"fmt"
	"io/ioutil"
	"math/bits"
	"os"
	"strings"
)

func check(e error) {
	if e != nil {
		panic(e)
	}
}

func main() {
	if len(os.Args) != 3 {
		usageString := strings.Join(
			[]string{"Usage:",
				"  -c filePath\t//Compress the file located at filePath using RLE",
				"  -d filePath\t//Decompress the RLE compressed file at filePath"}, "\n")
		fmt.Print(usageString)
		return
	}

	switch os.Args[1] {
	case "-c":

		outputBuffer, outputName := compress(os.Args[2:])
		err := ioutil.WriteFile("outputs/"+outputName, outputBuffer, 0644)
		check(err)
		fmt.Printf("Wrote to %s", "outputs/"+outputName)

	case "-d":
		// fmt.Printf("Decompressing %s\n\tsize: %s\n", fileName, formatFileSize(int(fileSize)))
		_, outputName := decompress(os.Args[2:])
		print(outputName)
		// err = ioutil.WriteFile("outputs/"+fileName+".rle", outputBuffer, 0644)
		// check(err)
		// fmt.Printf("Wrote to %s", "outputs/"+fileName[:len(fileName)-3])

	}

}

func compress(args []string) ([]byte, string) {
	fileName := args[0][strings.LastIndex(args[0], "\\")+1:]

	dat, err := ioutil.ReadFile(args[0])
	check(err)
	info, err := os.Stat(args[0])
	check(err)
	fileSize := info.Size() * 8

	fmt.Printf("Compressing %s\n\tsize: %s\n", fileName, formatFileSize(int(fileSize)))
	buffer := make([]bool, 0, len(dat)*8)
	counts := make([]uint32, 0)

	var initVal, currVal bool
	var counter, maxSize uint32 = 1, 0
	var numBits int
	for _, v := range dat {
		buffer = append(buffer, byte2Bits(v)...)
		// fmt.Printf("Len of buffer: %d\n Len of dat: %d\n", len(buffer), len(dat))

	}
	initVal = buffer[0]
	currVal = initVal
	buffer = buffer[1:]
	progress := 0
	countCounts := make([]int, 32, 32)

	for i, v := range buffer {
		if currVal == v {
			counter++
		} else {
			currVal = v
			counts = append(counts, counter)
			if maxSize < counter {
				maxSize = counter
			}
			countCounts[bits.Len32(counter)]++
			counter = 1
		}
		if i*100/len(buffer) > progress {
			progress = i * 100 / len(buffer)
			// fmt.Printf("Counts: %d, i: %d\n", len(counts), i)
			// fmt.Print("\r", progress, "   ", formatFileSize(len(counts)*8))
		}
	}
	printCountDistribution(countCounts)
	//Number of bits required to represent the largest coefficient
	numBits = bits.Len32(maxSize)
	// fmt.Printf("Counts := %v\nMaxSize := %d ~~ %d bits\nbitStringOfMax := %v\n", counts, maxSize, numBits, num2Bits(maxSize, numBits))
	//Reallocate buffer to house the binary of the number to be written
	requiredBits := 1 + 32 + (numBits * len(counts))
	paddingBits := 8 - (requiredBits % 8)
	fmt.Printf("\nRequired bits: %d, Padding bits: %d\nInitVal: %v\nBitwidth of Coeff: %d\n", requiredBits, paddingBits, initVal, numBits)
	paddingSlice := make([]bool, paddingBits, paddingBits)
	buffer = make([]bool, 0, requiredBits+paddingBits)
	//insert the booleans for the first element and the 32 bits housing the size of each coefficient
	buffer = append(
		append(buffer, initVal),
		num2Bits(uint32(numBits), 32)...)

	for _, v := range counts {
		buffer = append(buffer, num2Bits(uint32(v), numBits)...)
	}
	buffer = append(buffer, paddingSlice...)
	compressionRatio := float64(len(buffer)) / float64(fileSize)
	fmt.Printf("FileSize %v -> %d bits\nCompression Ratio: %f\n", fileSize, len(buffer), compressionRatio)
	outputBuffer := make([]byte, 0, len(buffer)/8)

	numIterations := len(buffer) / 8
	for i := 0; i < numIterations; i++ {
		startIndex := i * 8
		endIndex := startIndex + 8
		outputBuffer = append(outputBuffer, bits2Byte(buffer[startIndex:endIndex]))
	}
	outputName := fileName[:len(fileName)-3]
	return outputBuffer, outputName
}

func decompress(args []string) ([]byte, string) {
	fileName := args[0][strings.LastIndex(args[0], "\\")+1:]

	dat, err := ioutil.ReadFile(args[0])
	check(err)
	info, err := os.Stat(args[0])
	check(err)
	fileSize := info.Size() * 8

	buffer := make([]bool, 0, len(dat)*8)
	for _, v := range dat {
		buffer = append(buffer, byte2Bits(v)...)
	}
	fmt.Printf("Initial buffer size : %d\n", len(buffer))
	paddingBits := bits.TrailingZeros(uint(dat[len(dat)-1]))
	buffer = buffer[:len(buffer)-paddingBits]
	fmt.Printf("Post trailing strip size : %d\n", len(buffer))
	initVal := buffer[0]
	buffer = buffer[1:]
	fmt.Printf("Post init strip size : %d\n", len(buffer))
	coeffSize := bits2Num(buffer[:32])
	fmt.Printf("Decompressing %s\n\tsize: %s\n\tInitVal: %v\n\tPaddingBits: %d\n\tCoeffSize: %d\n", fileName, formatFileSize(int(fileSize)), initVal, paddingBits, coeffSize)

	//Create an output buffer with space for twice as many bits as is contained in the input data
	binBuffer := make([]bool, 0, len(dat)*16)

	buffer = buffer[32:]
	fmt.Printf("Post coeff strip size : %d\n", len(buffer))
	currVal := initVal
	var numToAdd uint32
	numIterations := len(buffer) / coeffSize
	for i := 0; i < numIterations; i++ {
		startIndex := coeffSize * i
		endIndex := startIndex + coeffSize
		numToAdd = uint32(bits2Num(buffer[startIndex:endIndex]))
		for j := 0; j < int(numToAdd); j++ {
			binBuffer = append(binBuffer, currVal)
		}
		// fmt.Printf("Took %d - %d / %d\n", startIndex, endIndex, len(buffer))
	}
	fmt.Printf("binBuffer has len %d", len(binBuffer))
	outputBuffer := make([]byte, 0, len(buffer)/8)
	//TRACK the missing bit
	numIterations = cap(outputBuffer)
	for i := 0; i < numIterations; i++ {
		startIndex := 8 * i
		endIndex := startIndex + 8
		outputBuffer = append(outputBuffer, bits2Byte(binBuffer[startIndex:endIndex]))
	}

	return outputBuffer, ""
}

func printCountDistribution(in []int) {
	fmt.Printf("Count distribution (%d):\n", len(in))
	counter := 0
	for i, v := range in {
		if v > 0 {
			fmt.Printf("\t%d: %d\n", i, v)
			counter += v
		}
	}
	fmt.Printf("\tTotal: %d\n", counter)
}

func byte2Bits(in byte) []bool {
	base := byte(1)
	out := make([]bool, 8)
	for i := range out {
		out[7-i] = (base<<i)&in > 0

	}
	return out
}

func num2Bits(in uint32, size int) []bool {
	out := make([]bool, size)
	for i := range out {
		out[size-1-i] = (1<<i)&in > 0
	}
	return out
}

func bits2Num(in []bool) int {
	if len(in) > 32 {
		panic("Input array too large")
	}
	out := 0
	for i, v := range in {
		if v {
			out |= 1 << (len(in) - 1 - i)
		}
	}
	return out
}

func bits2Byte(in []bool) byte {
	if len(in) > 8 {
		panic("Input array too large")
	}
	out := byte(0)
	for i, v := range in {
		if v {
			out |= 1 << (len(in) - 1 - i)
		}
	}
	return out
}

func formatFileSize(in int) string {
	if in < 1<<10 {
		return fmt.Sprintf("%f b", float64(in))
	} else if in < 1<<20 {
		return fmt.Sprintf("%f Kb", float64(in/(1<<10)))
	} else if in < 1<<30 {
		return fmt.Sprintf("%f Mb", float64(in/(1<<20)))
	}
	return "oof"
}

// func byte2String(in byte) {
// 	var out strings.Builder
// 	for i := range
// }
