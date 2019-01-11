// Copyright (c) 2018 Aram Hăvărneanu <aram@mgk.ro>
//
// Permission to use, copy, modify, and distribute this software for any
// purpose with or without fee is hereby granted, provided that the above
// copyright notice and this permission notice appear in all copies.
//
// THE SOFTWARE IS PROVIDED "AS IS" AND THE AUTHOR DISCLAIMS ALL WARRANTIES
// WITH REGARD TO THIS SOFTWARE INCLUDING ALL IMPLIED WARRANTIES OF
// MERCHANTABILITY AND FITNESS. IN NO EVENT SHALL THE AUTHOR BE LIABLE FOR
// ANY SPECIAL, DIRECT, INDIRECT, OR CONSEQUENTIAL DAMAGES OR ANY DAMAGES
// WHATSOEVER RESULTING FROM LOSS OF USE, DATA OR PROFITS, WHETHER IN AN
// ACTION OF CONTRACT, NEGLIGENCE OR OTHER TORTIOUS ACTION, ARISING OUT OF
// OR IN CONNECTION WITH THE USE OR PERFORMANCE OF THIS SOFTWARE.

/*
Fovgen generates https://xw.is/wiki/Focal_length_equivalents_between_formats.
*/
package main

import (
	"fmt"
	"math"
	"os"
)

type sensor struct {
	h, v float64
	name string
}

func (s sensor) String() string {
	return s.name
}

func (s sensor) AspectRatio() float64 {
	return s.h / s.v
}

var (
	sensorAPSC     = sensor{25.1, 16.7, "APS-C"}
	sensorFF       = sensor{36, 24, "35mm"}
	sensorMF4433   = sensor{43.8, 32.8, "Fuji GFX/Pentax 645"}
	sensorMF4937   = sensor{49.1, 36.8, "HxD-39/50"}
	sensorMF5440   = sensor{53.7, 40.2, "HxD-60/100"}
	sensorMF4433TS = sensor{43.8 / 1.5, 32.8 / 1.5, "Fuji GFX (HTS)"}
	sensorMF4937TS = sensor{49.1 / 1.5, 36.8 / 1.5, "HxD-39/50(HTS)"}
	sensorMF5440TS = sensor{53.7 / 1.5, 40.2 / 1.5, "HxD-60/100(HTS)"}
	sensorLF45     = sensor{127, 102, "4x5"}
	sensorLF57     = sensor{177.8, 127, "5x7"}
	sensorLF617    = sensor{170, 60, "6x17"}
	sensorLF610    = sensor{254, 152, "6x10"}
	sensorLF810    = sensor{254, 203, "8x10"}
	sensorLF1114   = sensor{355, 279, "11x14"}
	sensorLF1220   = sensor{508, 304, "12x20"}
)

var sensors = []sensor{
	sensorAPSC,
	sensorFF,
	sensorMF4433,
	sensorMF4937,
	sensorMF5440,
	sensorMF4433TS,
	sensorMF4937TS,
	sensorMF5440TS,
	sensorLF45,
	sensorLF57,
	sensorLF617,
	// sensorLF610 is intentionally missing; we only care about conversions *from* 6x10
	sensorLF810,
	sensorLF1114,
	sensorLF1220,
}

type Lens struct {
	focal float64
	tc    float64 // like HTS 1.5
	mfg   string
}

func (l Lens) String() string {
	var s string
	if l.mfg != "" {
		s = l.mfg + " "
	}
	s += fmt.Sprintf("%.0fmm", l.focal)
	if l.tc == 0 {
		return s
	}
	return s + fmt.Sprintf(" (*%.1f)", l.tc)
}

func (l Lens) Focal() float64 {
	if l.tc != 0 {
		return l.focal * l.tc
	}
	return l.focal
}

var lensesAPSC = []Lens{
	{focal: 10},
	{focal: 14},
	{focal: 16},
	{focal: 18},
	{focal: 35},
	{focal: 50},
	{focal: 55},
	{focal: 56},
	{focal: 60},
	{focal: 80},
	{focal: 90},
	{focal: 200},
}

var lensesFF = []Lens{
	{focal: 14},
	{focal: 16},
	{focal: 17},
	{focal: 19},
	{focal: 20},
	{focal: 24},
	{focal: 28},
	{focal: 35},
	{focal: 45},
	{focal: 50},
	{focal: 85},
	{focal: 105},
	{focal: 120},
	{focal: 200},
}

var lensesGFX = []Lens{
	{focal: 23},
	{focal: 32},
	{focal: 45},
	{focal: 50},
	{focal: 63},
	{focal: 100},
	{focal: 110},
	{focal: 200},
	{focal: 250},
}

var lensesPentax = []Lens{
	{focal: 25},
	{focal: 28},
	{focal: 33},
	{focal: 45},
	{focal: 55},
	{focal: 80},
	{focal: 85},
	{focal: 110},
	{focal: 150},
	{focal: 160},
	{focal: 300},
}

var lensesHasselblad = []Lens{
	{focal: 24},
	{focal: 28},
	{focal: 35},
	{focal: 50},
	{focal: 80},
	{focal: 90},
	{focal: 110},
	{focal: 150},
	{focal: 210},
	{focal: 300},
}

func init() {
	for _, v := range lensesHasselblad {
		v1 := v
		v1.tc = 1.5
		lensesHasselblad = append(lensesHasselblad, v1)
	}
}

func init() {
	for _, v := range lensesHasselblad {
		v1 := v
		v1.mfg = "Hasselblad"
		lensesGFX = append(lensesGFX, v1)
	}
}

var lensesLF45 = []Lens{
	{focal: 47},
	{focal: 65},
	{focal: 72},
	{focal: 90},
	{focal: 150},
	{focal: 180},
	{focal: 210},
	{focal: 240},
	{focal: 300},
}

var lensesLF57 = []Lens{
	{focal: 72},
	{focal: 90},
	{focal: 150},
	{focal: 180},
	{focal: 210},
	{focal: 300},
	{focal: 450},
}

var lensesLF617 = []Lens{
	{focal: 47},
	{focal: 58},
	{focal: 72},
	{focal: 90},
	{focal: 150},
	{focal: 180},
	{focal: 210},
	{focal: 240},
	{focal: 300},
}

var lensesLF610 = []Lens{
	{focal: 72},
	{focal: 90},
	{focal: 120},
	{focal: 150},
}

var lensesLF810 = []Lens{
	{focal: 120},
	{focal: 150},
	{focal: 210},
	{focal: 240},
	{focal: 300},
	{focal: 360},
	{focal: 450},
	{focal: 600},
}

var lensesLF1114 = []Lens{
	{focal: 150},
	{focal: 210},
	{focal: 300},
	{focal: 360},
	{focal: 480},
	{focal: 600},
	{focal: 800},
}

var lensesLF1220 = []Lens{
	{focal: 210},
}

type camera struct {
	sensor
	lenses *[]Lens
	Name   string
}

var (
	cameraAPSC         = camera{sensorAPSC, &lensesAPSC, "APS-C"}
	cameraFF           = camera{sensorFF, &lensesFF, "35mm full frame"}
	cameraGFX          = camera{sensorMF4433, &lensesGFX, "Fuji GFX"}
	cameraPentax       = camera{sensorMF4433, &lensesPentax, "Pentax 645"}
	cameraHasselblad60 = camera{sensorMF5440, &lensesHasselblad, "Hasselblad H5D-60"}
	cameraHasselblad50 = camera{sensorMF4937, &lensesHasselblad, "Hasselblad H5D-50"}
	cameraLF45         = camera{sensorLF45, &lensesLF45, "Large format (4x5)"}
	cameraLF57         = camera{sensorLF57, &lensesLF57, "Large format (5x7)"}
	cameraLF617        = camera{sensorLF617, &lensesLF617, "Large format (6x17)"}
	cameraLF610        = camera{sensorLF610, &lensesLF610, "Large format (6x10)"}
	cameraLF810        = camera{sensorLF810, &lensesLF810, "Large format (8x10)"}
	cameraLF1114       = camera{sensorLF1114, &lensesLF1114, "Large format (11x14)"}
	cameraLF1220       = camera{sensorLF1220, &lensesLF1220, "Large format (12x20)"}
)

var cameras = []camera{
	cameraAPSC,
	cameraFF,
	cameraGFX,
	cameraPentax,
	cameraHasselblad50,
	cameraHasselblad60,
	cameraLF45,
	cameraLF57,
	cameraLF617,
	cameraLF610,
	cameraLF810,
	cameraLF1114,
	cameraLF1220,
}

func fov(ssize, focal float64) float64 {
	return 2 * math.Atan(ssize/(2*focal)) * 180 / math.Pi
}

type lensInfo struct {
	Lens
	HFoV float64
	VFoV float64
	EqW  map[sensor]float64
	EqH  map[sensor]float64
	EqV  map[sensor]float64
}

func equivalentW(l Lens, s sensor, starg sensor) float64 {
	if s.AspectRatio() <= starg.AspectRatio() {
		return l.Focal() * starg.v / s.v
	}
	return l.Focal() * starg.h / s.h
}

func equivalentH(l Lens, s sensor, starg sensor) float64 {
	return l.Focal() * starg.h / s.h
}

func equivalentV(l Lens, s sensor, starg sensor) float64 {
	return l.Focal() * starg.v / s.v
}

type cameraInfo struct {
	camera
	Lenses []lensInfo
}

type tables struct {
	CameraInfo []cameraInfo
	Sensors    []sensor
}

func main() {
	tbl := tables{Sensors: sensors}
	for _, c := range cameras {
		ci := cameraInfo{c, nil}
		lensInfos := []lensInfo{}
		for _, lens := range *c.lenses {
			li := lensInfo{Lens: lens}
			li.HFoV = fov(c.sensor.h, lens.Focal())
			li.VFoV = fov(c.sensor.v, lens.Focal())

			li.EqW = make(map[sensor]float64)
			li.EqH = make(map[sensor]float64)
			li.EqV = make(map[sensor]float64)
			for _, m := range sensors {
				li.EqW[m] = equivalentW(lens, c.sensor, m)
				li.EqH[m] = equivalentH(lens, c.sensor, m)
				li.EqV[m] = equivalentV(lens, c.sensor, m)
			}

			lensInfos = append(lensInfos, li)
		}
		ci.Lenses = lensInfos
		tbl.CameraInfo = append(tbl.CameraInfo, ci)
	}
	err := wiki.Execute(os.Stdout, tbl)
	if err != nil {
		panic(err)
	}
}
