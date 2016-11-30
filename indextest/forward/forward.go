package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	farm "github.com/dgryski/go-farm"

	"github.com/dgraph-io/dgraph/x"
)

const (
	filename = "/home/jchiu/dgraphtest/goldendata"
)

var (
	graph         map[string]map[uint64][]uint64
	gNames        map[uint64]string
	gReleaseDates map[uint64]string
)

func bracketed(s string) bool {
	return strings.HasPrefix(s, "<") && strings.HasSuffix(s, ">")
}

func removeFirstLast(s string) string {
	return s[1 : len(s)-1]
}

// uniqueUIDs return the unique UIDs in the values of map m. It also returns
// a map from unique UIDs to their counts, i.e., the number of times they
// appear in the values of map m.
func uniqueUIDs(m map[uint64][]uint64) ([]uint64, []int) {
	uniq := make(map[uint64]int)
	for _, v := range m {
		for _, u := range v {
			uniq[u]++
		}
	}
	out := make([]uint64, 0, len(uniq))
	outCount := make([]int, 0, len(uniq))
	for k, v := range uniq {
		out = append(out, k)
		outCount = append(outCount, v)
	}
	return out, outCount
}

func expand(uids []uint64, pred string) []uint64 {
	x.AssertTrue(graph != nil)
	var out []uint64
	for p, m := range graph {
		if pred != p {
			continue
		}
		for _, u := range uids {
			dst := m[u]
			if dst == nil {
				continue
			}
			out = append(out, dst...)
		}
	}
	return out
}

func main() {
	x.Init()

	fin, err := os.Open(filename)
	x.Check(err)
	defer fin.Close()

	scanner := bufio.NewScanner(fin)
	var numLines, numValues, numNames, numReleaseDates int
	graph = make(map[string]map[uint64][]uint64)
	gNames = make(map[uint64]string)
	gReleaseDates = make(map[uint64]string)

	for scanner.Scan() {
		numLines++
		tokens := strings.Split(scanner.Text(), "\t")
		x.AssertTruef(len(tokens) == 4, scanner.Text())

		src := tokens[0]
		x.AssertTrue(bracketed(src))
		src = removeFirstLast(src)
		srcUID := farm.Fingerprint64([]byte(src))

		pred := tokens[1]
		x.AssertTrue(bracketed(pred))
		pred = removeFirstLast(pred)

		value := tokens[2]

		if bracketed(value) {
			// Normal edge.
			value = removeFirstLast(value)
			destUID := farm.Fingerprint64([]byte(value))
			m, found := graph[pred]
			if !found {
				m = make(map[uint64][]uint64)
				graph[pred] = m
			}
			m[srcUID] = append(m[srcUID], destUID)
		} else {
			numValues++
			value = removeFirstLast(value)
			if pred == "type.object.name" {
				numNames++
				gNames[srcUID] = value
			} else if pred == "film.film.initial_release_date" {
				numReleaseDates++
				gReleaseDates[srcUID] = value
			}
		}
	}

	fmt.Printf("Num lines read: %d\n", numLines)
	fmt.Printf("Num predicates: %d\n", len(graph))
	fmt.Printf("Num values read: %d\n", numValues)
	fmt.Printf("Num names read: %d\n", numNames)
	fmt.Printf("Num release dates read: %d\n", numReleaseDates)

	x.AssertTrue(numLines > 0)
	x.AssertTrue(len(graph) > 0)
	x.AssertTrue(numValues > 0)
	x.AssertTrue(numNames > 0)
	x.AssertTrue(numReleaseDates > 0)

	doGood()
}

func doGood() {
	r := expand([]uint64{15161013152876854722}, "film.director.film")
	r = expand(r, "film.film.directed_by")
	r = expand(r, "film.director.film")
	fmt.Printf("Without filter: %d\n", len(r))

	var numHits int
	for _, u := range r {
		name, found := gNames[u]
		if !found {
			continue
		}
		tokens := strings.Split(strings.ToLower(name), " ")
		var found1, found2 bool
		for _, t := range tokens {
			if t == "the" {
				found1 = true
			} else if t == "a" {
				found2 = true
			}
		}
		if found1 && found2 {
			numHits++
		}
	}
	fmt.Printf("With filter: %d\n", numHits)
}
