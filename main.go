package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"reflect"
	"strconv"
	"sync"
	"time"

	"log"

	"github.com/groovili/gogtrends"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/proto"
)

const (
	locUS  = "US"
	catAll = "all"
	langEn = "EN"
)

var sg = new(sync.WaitGroup)

func main() {
	//Enable debug to see request-response
	//gogtrends.Debug(true)
	log.Println("Fetching previous records")
	filename := "bleptech"
	// fetchData("vipul")
	counter := 0
	ctx := context.Background()

	// take user input of keyword
	//

	for {
		log.Println("Explore Search:")
		keyword := "Go"

		// Fetching best Keywords
		keywords, err := gogtrends.Search(ctx, keyword, langEn)
		for _, v := range keywords {
			log.Println(v)
			if v.Type == "Language" {
				keyword = v.Mid
				break
			}
		}

		log.Println("Explore trends:")
		// get widgets for Golang keyword in programming category
		explore, err := gogtrends.Explore(ctx, &gogtrends.ExploreRequest{
			ComparisonItems: []*gogtrends.ComparisonItem{
				{
					Keyword: keyword,
					Geo:     locUS,
					Time:    "now 4-H",
				},
			},
			Category: 31, // Programming category
			Property: "",
		}, langEn)
		handleError(err, "Failed to explore widgets")
		// printItems(explore)

		log.Println("Interest over time:")
		overTime, err := gogtrends.InterestOverTime(ctx, explore[0], langEn)
		handleError(err, "Failed in call interest over time")
		if counter == 0 {
			// first time, save data and sleep
			saveData(overTime, filename+"-"+strconv.Itoa(counter), 1)
			counter = counter + 1
			time.Sleep(1 * time.Minute) // sleep for 10 mins
			continue
		}
		if counter > 0 {
			// fetch old data and calculate
			fn := filename + "-" + strconv.Itoa(counter-1)
			olddata := fetchData(fn)

			// scaling
			scale := scaleData(olddata, overTime)
			log.Println("scaling the data with: ", scale)

			// save the current data
			saveData(overTime, filename+"-"+strconv.Itoa(counter), scale)

			time.Sleep(1 * time.Minute) // sleep for 10 mins
			counter = counter + 1
		}

	}
}

func handleError(err error, errMsg string) {
	if err != nil {
		log.Fatal(errors.Wrap(err, errMsg))
	}
}

func printItems(items interface{}) {
	ref := reflect.ValueOf(items)

	if ref.Kind() != reflect.Slice {
		log.Fatalf("Failed to print %s. It's not a slice type.", ref.Kind())
	}

	for i := 0; i < ref.Len(); i++ {
		temp := ref.Index(i).Interface()
		log.Println(temp)
	}
}

func printNestedItems(cats []*gogtrends.ExploreCatTree) {
	defer sg.Done()
	for _, v := range cats {
		log.Println(v.Name, v.ID)
		if len(v.Children) > 0 {
			sg.Add(1)
			go printNestedItems(v.Children)
		}
	}
}

func saveData(items interface{}, fname string, scale float32) {

	// Read the existing address book.
	in, err := ioutil.ReadFile(fname)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Printf("%s: File not found.  Creating new file.\n", fname)
		} else {
			log.Fatalln("Error reading file:", err)
		}
	}

	// [START marshal_proto]
	book := &DataBook{}
	// [START_EXCLUDE]
	if err := proto.Unmarshal(in, book); err != nil {
		log.Fatalln("Failed to parse address book:", err)
	}

	ref := reflect.ValueOf(items)
	for i := 0; i < ref.Len(); i++ {
		time := ref.Index(i).Elem().FieldByName("Time")
		formattedAxisTime := ref.Index(i).Elem().FieldByName("FormattedAxisTime")
		formattedValue := ref.Index(i).Elem().FieldByName("FormattedValue").Index(0)
		formattedValueFloat, _ := strconv.ParseFloat(formattedValue.String(), 32)
		log.Println(formattedValueFloat)
		// hasData := ref.Index(i).Elem().FieldByName("hasData")

		res := &TimelineData{}
		res.Time = time.String()
		res.FormattedAxisTime = formattedAxisTime.String()
		res.FormattedValue = float32(formattedValueFloat) * scale

		book.Data = append(book.Data, res)
		log.Println(time, formattedAxisTime, formattedValue)
	}
	out, err := proto.Marshal(book)
	if err != nil {
		log.Fatalln("Failed to encode address book:", err)
	}
	if err := ioutil.WriteFile(fname, out, 0644); err != nil {
		log.Fatalln("Failed to write address book:", err)
	}
}

func fetchData(filename string) map[string]float32 {
	map_1 := make(map[string]float32)

	in, err := ioutil.ReadFile(filename)
	if err != nil {
		log.Fatalln("Error reading file:", err)
	}
	book := &DataBook{}
	if err := proto.Unmarshal(in, book); err != nil {
		log.Fatalln("Failed to parse address book:", err)
	}
	for _, p := range book.Data {
		log.Println(p.Time, p.FormattedAxisTime, p.FormattedValue)
		map_1[p.Time] = p.FormattedValue
	}
	return map_1
}

func scaleData(oldData map[string]float32, NewData interface{}) float32 {
	mean := float32(0.0)
	sum := float32(0.0)
	elems := float32(0.0)

	ref := reflect.ValueOf(NewData)
	var dataset [250]float32
	for i := 0; i < ref.Len(); i++ {
		time := ref.Index(i).Elem().FieldByName("Time").String()
		formattedValue := ref.Index(i).Elem().FieldByName("FormattedValue").Index(0)
		formattedValueFloat, _ := strconv.ParseFloat(formattedValue.String(), 32)
		value, ok := oldData[time]
		if ok {
			if value != 0 {
				delta := float32(formattedValueFloat) - value
				dataset[0] = delta
			}
		}
	}

	for i := 0; i < 250; i++ {
		sum += dataset[i]
		if dataset[i] > 0.0 {
			elems += 1
		}
	}

	mean = sum / elems

	return mean
}
