// SPDX-License-Identifier: MIT
// Copyright (c) 2020 Brian Starkey <stark3y@gmail.com>

package main

import (
	"flag"
	"fmt"
	"os"
	"math"
	"reflect"
	"strings"

	"github.com/tormoder/fit"
)

func dumpUnknownMessages(msgs []fit.UnknownMessage) {
	if len(msgs) == 0 {
		return
	}

	fmt.Printf("Unknown Messages:\n")
	for i, msg := range(msgs) {
		fmt.Printf("  [%d] %#v\n", i, msg)
	}
}

func dumpUnknownFields(fields []fit.UnknownField) {
	if len(fields) == 0 {
		return
	}

	fmt.Printf("Unknown Fields:\n")
	for i, field := range(fields) {
		fmt.Printf("  [%d] %#v\n", i, field)
	}
}

func printIndent(level int, format string, args ...interface{}) {
	fmt.Printf("%s", strings.Repeat("\t", level))
	fmt.Printf(format, args...)
}

var invalidValues = map[reflect.Kind]func(reflect.Value) bool {
	reflect.Bool: func(v reflect.Value) bool {
		return v.Bool() == false
	},

	reflect.Int8: func(v reflect.Value) bool {
		return v.Int() == 0x7f
	},

	reflect.Int16: func(v reflect.Value) bool {
		return v.Int() == 0x7fff
	},

	reflect.Int32: func(v reflect.Value) bool {
		return v.Int() == 0x7fffffff
	},

	reflect.Int64: func(v reflect.Value) bool {
		return v.Int() == 0x7fffffffffffffff
	},

	reflect.Uint8: func(v reflect.Value) bool {
		return v.Uint() == 0xff
	},

	reflect.Uint16: func(v reflect.Value) bool {
		return v.Uint() == 0xffff
	},

	reflect.Uint32: func(v reflect.Value) bool {
		return v.Uint() == 0xffffffff
	},

	reflect.Uint64: func(v reflect.Value) bool {
		return v.Uint() == 0xffffffffffffffff
	},

	reflect.Float32: func(v reflect.Value) bool {
		return float32(v.Float()) == math.Float32frombits(0xFFFFFFFF)
	},

	reflect.Float64: func(v reflect.Value) bool {
		return v.Float() == math.Float64frombits(0xFFFFFFFFFFFFFFFF)
	},

	reflect.String: func(v reflect.Value) bool {
		return v.String() == ""
	},
}

func dumpField(field reflect.Value, name string, level int) {
	if _, ok := field.Type().MethodByName("String"); ok {
		method := field.MethodByName("String")
		str := method.Call(nil)[0].String()
		if strings.HasSuffix(str, "Invalid") {
			return
		}
		printIndent(level, "%s: %s\n", name, str);
	} else if invalidFunc, ok := invalidValues[field.Kind()]; ok {
		if invalidFunc(field) {
			return
		}
		printIndent(level, "%s: %v\n", name, field);
	} else {
		printIndent(level, "%s: %+v\n", name, field);
	}
}

func dumpRecursive(val reflect.Value, name string, level int) {
	switch val.Kind() {
	case reflect.Struct:
		if _, ok := val.Type().MethodByName("String"); ok {
			method := val.MethodByName("String")
			str := method.Call(nil)[0].String()
			printIndent(level, "%s: %s\n", name, str);
		} else {
			printIndent(level, "%s:\n", name)
			for i := 0; i < val.NumField(); i++ {
				v := val.Field(i)
				name = val.Type().Field(i).Name
				// FIXME: Unicode.
				if len(name) > 0 && strings.ToLower(name[:1]) == name[:1] {
					continue
				}

				dumpRecursive(v, val.Type().Field(i).Name, level+1)
			}
		}
	case reflect.Ptr:
		if val.IsNil() {
			break
		}
		dumpRecursive(reflect.Indirect(val), name, level)
	case reflect.Slice:
		if val.Len() == 0 {
			break
		}
		printIndent(level, "%s (%d elems):\n", name, val.Len())
		for i := 0; i < val.Len(); i++ {
			name = fmt.Sprintf("[%d]", i)
			dumpRecursive(reflect.Indirect(val.Index(i)), name, level+1)
		}
	default:
		dumpField(val, name, level)
	}

}

func dumpFile(file reflect.Value) {
	dumpRecursive(file, "File", 0)
}

func run() error {
	if flag.NArg() != 1 {
		return fmt.Errorf("Expected a single argument: FILE")
	}

	f, err := os.Open(flag.Args()[0])
	if err != nil {
		return err
	}
	defer f.Close()

	fitf, err := fit.Decode(f, fit.WithUnknownMessages())
	if err != nil {
		return err
	}

	dumpRecursive(reflect.ValueOf(fitf.Header), "Header", 0)
	dumpRecursive(reflect.ValueOf(fitf.CRC), "File CRC", 0)
	dumpRecursive(reflect.ValueOf(fitf.FileId), "FileId", 0)
	dumpRecursive(reflect.ValueOf(fitf.FileCreator), "FileCreator", 0)
	dumpRecursive(reflect.ValueOf(fitf.TimestampCorrelation), "TimestampCorrelation", 0)
	dumpRecursive(reflect.ValueOf(fitf.UnknownMessages), "UnknownMessages", 0)
	dumpRecursive(reflect.ValueOf(fitf.UnknownFields), "UnknownFields", 0)

	var data reflect.Value
	switch fitf.Type() {
	case fit.FileTypeActivity:
		activity, err := fitf.Activity()
		if err != nil {
			return fmt.Errorf("get file failed: %v", err)
		}
		data = reflect.ValueOf(*activity)
	case fit.FileTypeDevice:
		device, err := fitf.Device()
		if err != nil {
			return fmt.Errorf("get file failed: %v", err)
		}
		data = reflect.ValueOf(*device)
	case fit.FileTypeSettings:
		settings, err := fitf.Settings()
		if err != nil {
			return fmt.Errorf("get file failed: %v", err)
		}
		data = reflect.ValueOf(*settings)
	case fit.FileTypeSport:
		sport, err := fitf.Sport()
		if err != nil {
			return fmt.Errorf("get file failed: %v", err)
		}
		data = reflect.ValueOf(*sport)
	case fit.FileTypeWorkout:
		workout, err := fitf.Workout()
		if err != nil {
			return fmt.Errorf("get file failed: %v", err)
		}
		data = reflect.ValueOf(*workout)
	case fit.FileTypeCourse:
		course, err := fitf.Course()
		if err != nil {
			return fmt.Errorf("get file failed: %v", err)
		}
		data = reflect.ValueOf(*course)
	case fit.FileTypeSchedules:
		schedules, err := fitf.Schedules()
		if err != nil {
			return fmt.Errorf("get file failed: %v", err)
		}
		data = reflect.ValueOf(*schedules)
	case fit.FileTypeWeight:
		weight, err := fitf.Weight()
		if err != nil {
			return fmt.Errorf("get file failed: %v", err)
		}
		data = reflect.ValueOf(*weight)
	case fit.FileTypeTotals:
		totals, err := fitf.Totals()
		if err != nil {
			return fmt.Errorf("get file failed: %v", err)
		}
		data = reflect.ValueOf(*totals)
	case fit.FileTypeGoals:
		goals, err := fitf.Goals()
		if err != nil {
			return fmt.Errorf("get file failed: %v", err)
		}
		data = reflect.ValueOf(*goals)
	case fit.FileTypeBloodPressure:
		bloodPressure, err := fitf.BloodPressure()
		if err != nil {
			return fmt.Errorf("get file failed: %v", err)
		}
		data = reflect.ValueOf(*bloodPressure)
	case fit.FileTypeMonitoringA:
		monitoringA, err := fitf.MonitoringA()
		if err != nil {
			return fmt.Errorf("get file failed: %v", err)
		}
		data = reflect.ValueOf(*monitoringA)
	case fit.FileTypeActivitySummary:
		activitySummary, err := fitf.ActivitySummary()
		if err != nil {
			return fmt.Errorf("get file failed: %v", err)
		}
		data = reflect.ValueOf(*activitySummary)
	case fit.FileTypeMonitoringDaily:
		monitoringDaily, err := fitf.MonitoringDaily()
		if err != nil {
			return fmt.Errorf("get file failed: %v", err)
		}
		data = reflect.ValueOf(*monitoringDaily)
	case fit.FileTypeMonitoringB:
		monitoringB, err := fitf.MonitoringB()
		if err != nil {
			return fmt.Errorf("get file failed: %v", err)
		}
		data = reflect.ValueOf(*monitoringB)
	case fit.FileTypeSegment:
		segment, err := fitf.Segment()
		if err != nil {
			return fmt.Errorf("get file failed: %v", err)
		}
		data = reflect.ValueOf(*segment)
	case fit.FileTypeSegmentList:
		segmentList, err := fitf.SegmentList()
		if err != nil {
			return fmt.Errorf("get file failed: %v", err)
		}
		data = reflect.ValueOf(*segmentList)
	default:
		return fmt.Errorf("get file failed: Unknown filetype '%v'", fitf.Type())
	}

	dumpRecursive(data, data.Type().Name(), 0)

	return nil
}

func main() {

	flag.Parse()

	err := run()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	os.Exit(0)
}
