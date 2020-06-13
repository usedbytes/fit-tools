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
	"unicode"
	"unicode/utf8"

	"github.com/tormoder/fit"
)

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
	if method := field.MethodByName("String"); method.IsValid() {
		str := method.Call(nil)[0].String()
		if strings.HasSuffix(str, "Invalid") {
			return
		}
		printIndent(level, "%s: %s\n", name, str);
	} else if invalidFunc, ok := invalidValues[field.Kind()]; ok {
		// FIXME: This doesn't handle the 'z' variants, but I'm not sure
		// there's much that can be done about it as the information on
		// the field type is hidden.
		// This also means that a field might be incorrectly excluded,
		// if it's a 'z' type and holds a value which looks invalid for
		// a non-'z' type.
		// Fixing this without modifying the fit package probably means
		// auto-generating a map of message type -> constructor, to
		// compare against. Alternatively, the fit package could be
		// extended to provide information on invalid values, but I'm
		// not sure what a good interface for that would look like.
		if invalidFunc(field) {
			return
		}
		printIndent(level, "%s: %v\n", name, field);
	} else {
		printIndent(level, "%s: %+v\n", name, field);
	}
}

func exported(name string) bool {
	r, l := utf8.DecodeRune([]byte(name))
	if r == utf8.RuneError && (l <= 1) {
		// I guess this should never be able to happen
		panic("unicode error")
	}

	return unicode.IsUpper(r)
}

func dumpRecursive(val reflect.Value, name string, level int) {
	// TODO: I'm not very happy with all the different conditions/branches
	// here. It's a bit spaghetti
	if method := val.MethodByName("String"); method.IsValid() {
		// For Stringers, dump them right away
		dumpField(val, name, level)
	} else {
		switch val.Kind() {
		case reflect.Struct:
			// TODO: If all fields are invalid or unexported,
			// should we skip it entirely?
			printIndent(level, "%s:\n", name)
			for i := 0; i < val.NumField(); i++ {
				v := val.Field(i)
				name = val.Type().Field(i).Name
				if !exported(name) {
					continue
				}
				dumpRecursive(v, val.Type().Field(i).Name, level+1)
			}
			printIndent(level, "---\n")
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
}

func getFileValue(fitf *fit.File) (reflect.Value, error) {
	// Take care not to shadow these
	var data reflect.Value
	var err error

	// This could be done with reflection based on Field.Name(), but
	// then it would be tied to internal details of the fit package
	// which doesn't sound ideal.
	switch fitf.Type() {
	case fit.FileTypeActivity:
		var activity *fit.ActivityFile
		activity, err = fitf.Activity()
		// Note: == nil, success case
		if err == nil {
			data = reflect.ValueOf(*activity)
		}
	case fit.FileTypeDevice:
		var device *fit.DeviceFile
		device, err = fitf.Device()
		if err == nil {
			data = reflect.ValueOf(*device)
		}
	case fit.FileTypeSettings:
		var settings *fit.SettingsFile
		settings, err = fitf.Settings()
		if err == nil {
			data = reflect.ValueOf(*settings)
		}
	case fit.FileTypeSport:
		var sport *fit.SportFile
		sport, err = fitf.Sport()
		if err == nil {
			data = reflect.ValueOf(*sport)
		}
	case fit.FileTypeWorkout:
		var workout *fit.WorkoutFile
		workout, err = fitf.Workout()
		if err == nil {
			data = reflect.ValueOf(*workout)
		}
	case fit.FileTypeCourse:
		var course *fit.CourseFile
		course, err = fitf.Course()
		if err == nil {
			data = reflect.ValueOf(*course)
		}
	case fit.FileTypeSchedules:
		var schedules *fit.SchedulesFile
		schedules, err = fitf.Schedules()
		if err == nil {
			data = reflect.ValueOf(*schedules)
		}
	case fit.FileTypeWeight:
		var weight *fit.WeightFile
		weight, err = fitf.Weight()
		if err == nil {
			data = reflect.ValueOf(*weight)
		}
	case fit.FileTypeTotals:
		var totals *fit.TotalsFile
		totals, err = fitf.Totals()
		if err == nil {
			data = reflect.ValueOf(*totals)
		}
	case fit.FileTypeGoals:
		var goals *fit.GoalsFile
		goals, err = fitf.Goals()
		if err == nil {
			data = reflect.ValueOf(*goals)
		}
	case fit.FileTypeBloodPressure:
		var bloodPressure *fit.BloodPressureFile
		bloodPressure, err = fitf.BloodPressure()
		if err == nil {
			data = reflect.ValueOf(*bloodPressure)
		}
	case fit.FileTypeMonitoringA:
		var monitoringA *fit.MonitoringAFile
		monitoringA, err = fitf.MonitoringA()
		if err == nil {
			data = reflect.ValueOf(*monitoringA)
		}
	case fit.FileTypeActivitySummary:
		var activitySummary *fit.ActivitySummaryFile
		activitySummary, err = fitf.ActivitySummary()
		if err == nil {
			data = reflect.ValueOf(*activitySummary)
		}
	case fit.FileTypeMonitoringDaily:
		var monitoringDaily *fit.MonitoringDailyFile
		monitoringDaily, err = fitf.MonitoringDaily()
		if err == nil {
			data = reflect.ValueOf(*monitoringDaily)
		}
	case fit.FileTypeMonitoringB:
		var monitoringB *fit.MonitoringBFile
		monitoringB, err = fitf.MonitoringB()
		if err == nil {
			data = reflect.ValueOf(*monitoringB)
		}
	case fit.FileTypeSegment:
		var segment *fit.SegmentFile
		segment, err = fitf.Segment()
		if err == nil {
			data = reflect.ValueOf(*segment)
		}
	case fit.FileTypeSegmentList:
		var segmentList *fit.SegmentListFile
		segmentList, err = fitf.SegmentList()
		if err == nil {
			data = reflect.ValueOf(*segmentList)
		}
	default:
		return data, fmt.Errorf("unknown filetype '%v'", fitf.Type())
	}

	return data, err
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

	// Dump all of the exported fields
	dumpRecursive(reflect.ValueOf(*fitf), flag.Args()[0], 0)

	// Body isn't exported, so we have to handle it separately
	body, err := getFileValue(fitf)
	if err != nil {
		return err
	}
	dumpRecursive(body, body.Type().Name(), 0)

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
