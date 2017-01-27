package main

import (
	"errors"
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"
	"unsafe"

	"github.com/davecgh/go-spew/spew"
	"github.com/lxn/win"
)

func main() {

	counterName := `\.NET CLR Memory(*)\# Gen 0 Collections`

	c, err := readPerformanceCounter(counterName, 10)

	if err != nil {
		log.Printf("unable to read perf counter \n")
	}

	for m := range c {
		spew.Dump(m)
	}
}

func readPerformanceCounter(counter string, sleepInterval int) (chan []Metric, error) {

	var queryHandle win.PDH_HQUERY
	var counterHandle win.PDH_HCOUNTER

	ret := win.PdhOpenQuery(0, 0, &queryHandle)
	if ret != win.ERROR_SUCCESS {
		return nil, errors.New("Unable to open query through DLL call")
	}

	// test path
	ret = win.PdhValidatePath(counter)
	if ret == win.PDH_CSTATUS_BAD_COUNTERNAME {
		return nil, errors.New("Unable to fetch counter (this is unexpected)")
	}

	ret = win.PdhAddEnglishCounter(queryHandle, counter, 0, &counterHandle)
	if ret != win.ERROR_SUCCESS {
		return nil, errors.New(fmt.Sprintf("Unable to add process counter. Error code is %x\n", ret))
	}

	ret = win.PdhCollectQueryData(queryHandle)
	if ret != win.ERROR_SUCCESS {
		return nil, errors.New(fmt.Sprintf("Got an error: 0x%x\n", ret))
	}

	out := make(chan []Metric)

	go func() {
		for {
			ret = win.PdhCollectQueryData(queryHandle)
			if ret == win.ERROR_SUCCESS {

				var metric []Metric

				var bufSize uint32
				var bufCount uint32
				var size = uint32(unsafe.Sizeof(win.PDH_FMT_COUNTERVALUE_ITEM_DOUBLE{}))
				var emptyBuf [1]win.PDH_FMT_COUNTERVALUE_ITEM_DOUBLE // need at least 1 addressable null ptr.

				ret = win.PdhGetFormattedCounterArrayDouble(counterHandle, &bufSize, &bufCount, &emptyBuf[0])
				if ret == win.PDH_MORE_DATA {
					filledBuf := make([]win.PDH_FMT_COUNTERVALUE_ITEM_DOUBLE, bufCount*size)
					ret = win.PdhGetFormattedCounterArrayDouble(counterHandle, &bufSize, &bufCount, &filledBuf[0])
					if ret == win.ERROR_SUCCESS {
						for i := 0; i < int(bufCount); i++ {
							c := filledBuf[i]
							s := win.UTF16PtrToString(c.SzName)

							metricName := normalizePerfCounterMetricName(counter)
							if len(s) > 0 {
								metricName = fmt.Sprintf("%s.%s", normalizePerfCounterMetricName(counter), normalizePerfCounterMetricName(s))
							}

							metric = append(metric, Metric{
								metricName,
								fmt.Sprintf("%v", c.FmtValue.DoubleValue),
								time.Now().Unix()})
						}
					}
				}
				out <- metric
			}

			time.Sleep(time.Duration(sleepInterval) * time.Second)
		}
	}()

	return out, nil
}

func normalizePerfCounterMetricName(rawName string) (normalizedName string) {

	normalizedName = rawName

	// thanks to Microsoft Windows,
	// we have performance counter metric like `\\Processor(_Total)\\% Processor Time`
	// which we need to convert to `processor_total.processor_time` see perfcounter_test.go for more beautiful examples
	r := strings.NewReplacer(
		".", "",
		"\\", ".",
		" ", "_",
	)
	normalizedName = r.Replace(normalizedName)

	normalizedName = normalizeMetricName(normalizedName)
	return
}

func normalizeMetricName(rawName string) (normalizedName string) {

	normalizedName = strings.ToLower(rawName)

	// remove trailing and leading non alphanumeric characters
	re1 := regexp.MustCompile(`(^[^a-z0-9]+)|([^a-z0-9]+$)`)
	normalizedName = re1.ReplaceAllString(normalizedName, "")

	// replace whitespaces with underscore
	re2 := regexp.MustCompile(`\s`)
	normalizedName = re2.ReplaceAllString(normalizedName, "_")

	// remove non alphanumeric characters except underscore and dot
	re3 := regexp.MustCompile(`[^a-z0-9._]`)
	normalizedName = re3.ReplaceAllString(normalizedName, "")

	return
}
