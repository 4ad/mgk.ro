package main

import (
	"bytes"
	"encoding/csv"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strconv"

	_ "mgk.ro/log"
)

func usage() {
	fmt.Fprint(os.Stderr, "usage: fstat file...\n")
	flag.PrintDefaults()
	os.Exit(1)
}

func main() {
	flag.Usage = usage
	flag.Parse()

	if flag.NArg() == 0 {
		usage()
	}

	count := focals(flag.Args()...)
	max := maxval(count)

	fmt.Println(" value  ------------------------ distribution ------------------------ count")

	var keys []int
	for k := range count {
		keys = append(keys, k)
	}
	sort.Ints(keys)
	for _, k := range keys {
		fmt.Printf("%6d |%-62s %4d\n", k, stars(count[k], max), count[k])
	}
}

func focals(files ...string) map[int]int {
	r := csv.NewReader(exiftool(append([]string{"-q", "-FocalLengthIn35mmFormat", "-csv"}, files...)...))
	records, err := r.ReadAll()
	if err != nil {
		log.Fatal(err)
	}
	count := make(map[int]int)
	for _, rec := range records[1:] { // skip header
		re := regexp.MustCompile(`\d+`)
		mm, err := strconv.Atoi(re.FindString(rec[1])) // second column
		if err != nil {
			log.Fatal(err)
		}
		count[mm]++
	}
	return count
}

func exiftool(params ...string) io.Reader {
	cmd := exec.Command("exiftool", params...)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		log.Fatal(err)
	}
	return &out
}

func maxval(m map[int]int) int {
	max := 0
	for _, v := range m {
		if v > max {
			max = v
		}
	}
	return max
}

func stars(val, max int) (s string) {
	for i := 0; i < val*62/max; i++ {
		s = s + "@"
	}
	return s
}
